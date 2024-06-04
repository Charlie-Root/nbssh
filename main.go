package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"

	"github.com/go-resty/resty/v2"
)

// NetboxAPIResponse represents the response from the Netbox API for a device or VM
type NetboxAPIResponse struct {
	Results []struct {
		PrimaryIP struct {
			Address string `json:"address"`
		} `json:"primary_ip"`
	} `json:"results"`
}

func main() {
	// Define flags
	username := flag.String("u", "", "Username for SSH connection")
	flag.Parse()

	if flag.NArg() != 1 {
		fmt.Println("Usage: nbssh [-u username] <hostname>")
		return
	}
	hostname := flag.Arg(0)

	// Configure Netbox API
		// Read environment variables
	netboxURL := os.Getenv("NETBOX_URL")
	apiToken := os.Getenv("NETBOX_API_TOKEN")

	if netboxURL == "" || apiToken == "" {
		log.Fatalf("NETBOX_URL and NETBOX_API_TOKEN environment variables must be set")
	}

	// Create a Resty client
	client := resty.New()

	// Function to get the primary IP from Netbox
	getPrimaryIP := func(endpoint, hostname string) (string, error) {
		resp, err := client.R().
			SetHeader("Authorization", "Token "+apiToken).
			SetQueryParam("name", hostname).
			SetQueryParam("limit", "1").
			Get(endpoint)

		if err != nil {
			return "", fmt.Errorf("error fetching data from Netbox: %v", err)
		}

		if resp.StatusCode() != 200 {
			return "", fmt.Errorf("non-200 response from Netbox: %s", resp.Status())
		}

		var apiResponse NetboxAPIResponse
		if err := json.Unmarshal(resp.Body(), &apiResponse); err != nil {
			return "", fmt.Errorf("error unmarshaling response: %v", err)
		}

		if len(apiResponse.Results) == 0 {
			return "", nil // No result found
		}

		primaryIP := apiResponse.Results[0].PrimaryIP.Address
		ip, _, err := net.ParseCIDR(primaryIP)
		if err != nil {
			return "", fmt.Errorf("error parsing primary IP address: %v", err)
		}

		return ip.String(), nil
	}

	// Check both devices and virtual machines
	endpoints := []string{
		netboxURL + "/api/dcim/devices/",
		netboxURL + "/api/virtualization/virtual-machines/",
	}

	var primaryIP string
	for _, endpoint := range endpoints {
		ip, err := getPrimaryIP(endpoint, hostname)
		if err != nil {
			log.Fatalf("Error: %v", err)
		}
		if ip != "" {
			primaryIP = ip
			break
		}
	}

	if primaryIP == "" {
		log.Fatalf("Host %s not found in Netbox", hostname)
	}

	// Build the SSH command
	var sshCmd *exec.Cmd
	if *username == "" {
		sshCmd = exec.Command("ssh", primaryIP)
	} else {
		sshCmd = exec.Command("ssh", *username+"@"+primaryIP)
	}
	sshCmd.Stdin = os.Stdin
	sshCmd.Stdout = os.Stdout
	sshCmd.Stderr = os.Stderr

	if err := sshCmd.Run(); err != nil {
		log.Fatalf("Error initiating SSH connection: %v", err)
	}
}