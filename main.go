package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strings"
	"strconv"
	"sync"
	"log"
)

// add arguments
var (
	help         bool
	ipRange      string
	searchPorts  string
	threads	  int
)

func init() {
	flag.BoolVar(&help, "h", false, "help")
	flag.StringVar(&ipRange, "r", "", "ip range")
	flag.StringVar(&searchPorts, "p", "", "ports to search for")
	flag.IntVar(&threads, "t", 1, "number of threads")
	flag.Parse()
}

func isValidCIDR(input string) bool {
	_, _, err := net.ParseCIDR(input)
	return err == nil
}

func validateAndGetIPs(input string) []string {
	if isValidCIDR(input) {
		ips := []string{}
		ip, ipNet, _ := net.ParseCIDR(input)
		for ip := ip.Mask(ipNet.Mask); ipNet.Contains(ip); inc(ip) {
			ips = append(ips, ip.String())
		}
		// Exclude network and broadcast IPs
		if len(ips) > 2 {
			return ips[1 : len(ips)-1]
		}
		return ips
	} else if net.ParseIP(input) != nil {
		return []string{input}
	}
	return nil
}

func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

func fetchDataFromShodan(ip string) (*ShodanResponse, error) {
	url := "https://internetdb.shodan.io/" + ip
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var shodanResponse ShodanResponse
	if err := json.Unmarshal(body, &shodanResponse); err != nil {
		return nil, err
	}

	return &shodanResponse, nil
}

func processResponse(response *ShodanResponse, searchPorts string) []string {
	result := []string{}

	if searchPorts != "" {
		availablePorts := strings.Split(searchPorts, ",")
		for _, portStr := range availablePorts {
			port := strings.TrimSpace(portStr)
			portInt := parseInt(port)
			if contains(response.Ports, portInt) {
				result = append(result, fmt.Sprintf("%s:%s", response.IP, port))
			}
		}
	} else {
		for _, port := range response.Ports {
			result = append(result, fmt.Sprintf("%s:%d", response.IP, port))
		}
	}

	return result
}

type ShodanResponse struct {
	Hostnames []string `json:"hostnames"`
	IP        string   `json:"ip"`
	Ports     []int    `json:"ports"`
}

func parseInt(portStr string) int {
	port, _ := strconv.Atoi(portStr)
	return port
}

func contains(arr []int, val int) bool {
	for _, item := range arr {
		if item == val {
			return true
		}
	}
	return false
}

func main() {
	// check if help is requested
	if help {
		flag.PrintDefaults()
		os.Exit(0)
	}
	
	// ascii art saying internetDB and a small text saying "Wreck it"
	logo := `
	···················································································
	:██╗███╗   ██╗████████╗███████╗██████╗ ███╗   ██╗███████╗████████╗██████╗ ██████╗ :
	:██║████╗  ██║╚══██╔══╝██╔════╝██╔══██╗████╗  ██║██╔════╝╚══██╔══╝██╔══██╗██╔══██╗:
	:██║██╔██╗ ██║   ██║   █████╗  ██████╔╝██╔██╗ ██║█████╗     ██║   ██║  ██║██████╔╝:
	:██║██║╚██╗██║   ██║   ██╔══╝  ██╔══██╗██║╚██╗██║██╔══╝     ██║   ██║  ██║██╔══██╗:
	:██║██║ ╚████║   ██║   ███████╗██║  ██║██║ ╚████║███████╗   ██║   ██████╔╝██████╔╝:
	:╚═╝╚═╝  ╚═══╝   ╚═╝   ╚══════╝╚═╝  ╚═╝╚═╝  ╚═══╝╚══════╝   ╚═╝   ╚═════╝ ╚═════╝ :
	·····························································nerrorsec - NSL·······
	`
	log.Println(logo)

	if ipRange != "" {
		// validate the ipRange
		result := validateAndGetIPs(ipRange)
		if result != nil {
			// Use a WaitGroup to wait for all goroutines to finish
			var wg sync.WaitGroup

			// buffered channel to control the number of concurrent goroutines
			semaphore := make(chan struct{}, threads)

			for _, ip := range result {
				wg.Add(1)
				semaphore <- struct{}{} // acquire semaphore
				go func(ip string) {
					defer func() {
						wg.Done()
						<-semaphore // release semaphore
					}()

					shodanResponse, err := fetchDataFromShodan(ip)
					if err != nil {
						fmt.Println("Error:", err)
						return
					}

					result := processResponse(shodanResponse, searchPorts)
					if len(result) > 0 {
						fmt.Println(strings.Join(result, "\n"))
					}
				}(ip)
			}

			wg.Wait() // Wait for all goroutines to finish
		} else {
			fmt.Println("Invalid input")
		}
	}
}
