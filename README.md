# SNAP - Secure Nameserver Active Probing

SNAP (Secure Nameserver Active Probing) is an advanced tool designed to detect DNS poisoning through the active probing of nameservers. It is highly effective in identifying servers that provide valid DNS responses for well-established domains.

## Features
- Active probing of nameservers for precise validation
- Identification of DNS poisoning through comparison techniques
- Utilization of baseline DNS answers for accurate results
- Multi-threaded workload for enhanced performance

## Prerequisites
- Go programming language

## Utilization Procedure

1. Compile the project by executing the following command:
```
go build snap.go
```

2. Execute the compiled binary using the subsequent flags:

- `-f`: Mandatory flag representing the file containing a list of targets
- `-t`: Optional flag denoting the number of threads (default: 10)
- `-d`: Optional flag signifying a comma-separated list of known domains (default: "terra.com.br")

Example:
```
./snap -f targets.txt -t 20 -d "example.com,example.org"
```

## Licensing Information

This project is distributed as open-source software under the terms of the MIT License.