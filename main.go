package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-resty/resty/v2"
	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
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

	// Configure SSH client
	config := &ssh.ClientConfig{
		User: func() string {
			if *username != "" {
				return *username
			}
			// Try to get current user if no username provided
			if user := os.Getenv("USER"); user != "" {
				return user
			}
			return "root" // fallback
		}(),
		Auth: []ssh.AuthMethod{
			ssh.PasswordCallback(func() (string, error) {
				fmt.Print("Password: ")
				password, err := term.ReadPassword(int(syscall.Stdin))
				fmt.Println()
				return string(password), err
			}),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // Consider using proper host key verification in production
	}

	// Connect to SSH server
	conn, err := ssh.Dial("tcp", primaryIP+":22", config)
	if err != nil {
		log.Fatalf("Failed to connect to SSH server: %v", err)
	}
	defer conn.Close()

	// Create a session
	session, err := conn.NewSession()
	if err != nil {
		log.Fatalf("Failed to create SSH session: %v", err)
	}
	defer session.Close()

	// Set up terminal
	session.Stdout = os.Stdout
	session.Stderr = os.Stderr
	session.Stdin = os.Stdin

	// Get terminal size and mode
	if term.IsTerminal(int(os.Stdin.Fd())) {
		width, height, err := term.GetSize(int(os.Stdin.Fd()))
		if err == nil {
			if err := session.RequestPty("xterm", height, width, ssh.TerminalModes{}); err != nil {
				log.Fatalf("Failed to request PTY: %v", err)
			}
		}
	}

	// Handle terminal resize signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGWINCH)
	go func() {
		for range sigChan {
			if term.IsTerminal(int(os.Stdin.Fd())) {
				width, height, err := term.GetSize(int(os.Stdin.Fd()))
				if err == nil {
					session.WindowChange(height, width)
				}
			}
		}
	}()

	// Start interactive shell
	if err := session.Shell(); err != nil {
		log.Fatalf("Failed to start shell: %v", err)
	}

	// Wait for session to complete
	if err := session.Wait(); err != nil {
		log.Fatalf("SSH session ended with error: %v", err)
	}
}
