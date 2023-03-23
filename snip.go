package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/phuslu/fastdns"
	"github.com/schollz/progressbar/v3"
	"math/rand"
	"net"
	"net/netip"
	"os"
	"strings"
	"sync"
	"time"
)

var threads int
var (
	targets              []string
	baseServers          = []string{"1.1.1.1", "8.8.8.8"}
	knownDomains         = []string{"terra.com.br"}
	discoveredServers    []string
	discoveredServersMux sync.Mutex

	baseAnswers map[string][]string
)

func randString(length int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz"
	b := make([]byte, length)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func validIP(ip string) net.IP {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return nil
	}
	if parsedIP.To4() != nil {
		return parsedIP.To4()
	}
	return nil
}

func checkDNS(server string) {
	targetIP := validIP(server)
	if targetIP == nil {
		return
	}

	c := fastdns.Client{
		AddrPort:    netip.AddrPortFrom(netip.MustParseAddr(server), 53),
		ReadTimeout: 3 * time.Second,
		MaxConns:    1000,
	}

	for _, domain := range knownDomains {

		req, resp := fastdns.AcquireMessage(), fastdns.AcquireMessage()
		req.SetRequestQustion(domain, fastdns.TypeA, fastdns.ClassINET)
		err := c.Exchange(req, resp)
		if err != nil {
			// check if error was timeout
			if strings.Contains(err.Error(), "i/o timeout") {
				break
			}
			// fmt.Printf("Error when checking for %s: %s\n", domain, err.Error())

			continue
		}

		found := false
		_ = resp.Walk(func(name []byte, typ fastdns.Type, class fastdns.Class, ttl uint32, data []byte) bool {
			if typ == fastdns.TypeA {
				ip, _ := netip.AddrFromSlice(data)
				for _, baseAnswer := range baseAnswers[domain] {
					if baseAnswer == ip.String() {
						found = true
						break
					}
				}
			}
			return !found
		})

		if found {
			discoveredServersMux.Lock()
			discoveredServers = append(discoveredServers, targetIP.String())
			discoveredServersMux.Unlock()
		}
	}
}

func resolveBaseServer(server string) error {
	c := fastdns.Client{
		AddrPort:    netip.AddrPortFrom(netip.MustParseAddr(server), 53),
		ReadTimeout: 3 * time.Second,
		MaxConns:    1000,
	}
	isTimeout := false

	for _, domain := range knownDomains {
		if isTimeout {
			break
		}
		req, resp := fastdns.AcquireMessage(), fastdns.AcquireMessage()
		req.SetRequestQustion(domain, fastdns.TypeA, fastdns.ClassINET)
		err := c.Exchange(req, resp)
		if err != nil {
			// check if error was timeout
			if strings.Contains(err.Error(), "i/o timeout") {
				isTimeout = true
			}
			fmt.Printf("Error when checking for %s: %s\n", domain, err.Error())
			continue
		} else {
			_ = resp.Walk(func(name []byte, typ fastdns.Type, class fastdns.Class, ttl uint32, data []byte) bool {
				if typ == fastdns.TypeA {
					ip, _ := netip.AddrFromSlice(data)
					if _, ok := baseAnswers[domain]; !ok {
						baseAnswers[domain] = []string{}
					}
					baseAnswers[domain] = append(baseAnswers[domain], ip.String())
				}
				return true
			})
		}
	}

	return nil
}

func resolveBaseServers() {
	for _, server := range baseServers {
		if err := resolveBaseServer(server); err != nil {
			os.Exit(1)
		}
	}
}

func checkDNSTargets() {
	var wg sync.WaitGroup
	// create progress bar
	bar := progressbar.Default(int64(len(targets)))
	ch := make(chan string, len(targets))
	sem := make(chan struct{}, threads)

	for _, target := range targets {
		ch <- target
	}
	close(ch)

	for target := range ch {
		sem <- struct{}{}
		wg.Add(1)
		go func(target string) {
			defer wg.Done()
			checkDNS(target)
			bar.Add(1)

			<-sem
		}(target)
	}

	wg.Wait()
}

func main() {
	baseAnswers = make(map[string][]string)

	var targetsFile, domainsStr string
	flag.StringVar(&targetsFile, "f", "", "File containing list of targets")
	flag.IntVar(&threads, "t", 10, "Number of threads")
	flag.StringVar(&domainsStr, "d", "terra.com.br", "Comma separated list of known domains")
	flag.Parse()

	knownDomains = strings.Split(domainsStr, ",")

	if targetsFile != "" {
		file, err := os.Open(targetsFile)
		if err != nil {
			fmt.Printf("Error opening targets file: %s\n", err.Error())
			os.Exit(1)
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			targets = append(targets, scanner.Text())
		}

		if err := scanner.Err(); err != nil {
			fmt.Printf("Error reading targets file: %s\n", err.Error())
			os.Exit(1)
		}
	}
	if len(targets) == 0 {
		fmt.Printf("No targets provided\n")
		flag.PrintDefaults()
		os.Exit(1)
	}

	resolveBaseServers()
	checkDNSTargets()

	discoveredServersMux.Lock()
	fmt.Printf("Finished. Discovered %d servers\n", len(discoveredServers))
	for _, server := range discoveredServers {
		fmt.Println(server)
	}
	discoveredServersMux.Unlock()
}
