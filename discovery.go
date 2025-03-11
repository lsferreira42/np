package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/grandcat/zeroconf"
)

// mDNS service discovery constants
const (
	SERVICE_TYPE      = "_np._tcp"      // mDNS service type
	SERVICE_DOMAIN    = "local."        // mDNS service domain
	DISCOVERY_TIMEOUT = 5 * time.Second // Default timeout for service discovery
)

// ServiceInfo contains detailed information about a discovered service
type ServiceInfo struct {
	Name      string   // Service name
	Host      string   // Host name
	Port      int      // Port number
	Protocol  string   // "tcp" or "udp"
	Addresses []string // List of IP addresses
	Text      []string // TXT record contents
	TTL       uint32   // Time to live
	IsTCP     bool     // Whether the service uses TCP
}

// DiscoveryService manages service discovery and service announcement
// using multicast DNS (mDNS/Bonjour/Avahi)
type DiscoveryService struct {
	config     *Config                // Application configuration
	server     *zeroconf.Server       // mDNS server for service announcement
	mutex      sync.Mutex             // Mutex for thread-safe access
	services   map[string]ServiceInfo // Discovered services by name
	isRunning  bool                   // Whether discovery is active
	stopBrowse context.CancelFunc     // Function to stop service discovery
}

// NewDiscoveryService creates a new service discovery instance
func NewDiscoveryService(config *Config) *DiscoveryService {
	return &DiscoveryService{
		config:    config,
		services:  make(map[string]ServiceInfo),
		isRunning: false,
	}
}

// StartAnnounce broadcasts this service on the local network via mDNS
// allowing other NP instances to discover it automatically
func (ds *DiscoveryService) StartAnnounce(serviceName string, port int, isTCP bool) error {
	ds.mutex.Lock()
	defer ds.mutex.Unlock()

	if ds.isRunning {
		return fmt.Errorf("service announcement already running")
	}

	// Determine the protocol to announce in TXT metadata
	proto := "udp"
	if isTCP {
		proto = "tcp"
	}

	// Register the service with mDNS
	server, err := zeroconf.Register(
		serviceName,                // Service name
		SERVICE_TYPE,               // Service type
		SERVICE_DOMAIN,             // Domain
		port,                       // Port
		[]string{"proto=" + proto}, // TXT records
		nil,                        // Interfaces (all)
	)

	if err != nil {
		return fmt.Errorf("failed to register mDNS service: %v", err)
	}

	ds.server = server
	ds.isRunning = true

	return nil
}

// StopAnnounce stops the service announcement
func (ds *DiscoveryService) StopAnnounce() {
	if ds.server != nil {
		ds.server.Shutdown()
		ds.server = nil
		fmt.Fprintf(os.Stderr, "mDNS service announcement stopped\n")
	}
}

// StartBrowse begins looking for NP services on the local network
func (ds *DiscoveryService) StartBrowse() error {
	if ds.isRunning {
		return fmt.Errorf("discovery is already running")
	}

	// Create a cancelable context
	ctx, cancel := context.WithCancel(context.Background())
	ds.stopBrowse = cancel

	// Configure the resolver
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		return fmt.Errorf("failed to create mDNS resolver: %v", err)
	}

	// Channel to receive search results
	entries := make(chan *zeroconf.ServiceEntry, 10)

	// Start the search in a goroutine
	go func() {
		for entry := range entries {
			ds.addService(entry)
		}
	}()

	// Start searching for services
	err = resolver.Browse(ctx, SERVICE_TYPE, SERVICE_DOMAIN, entries)
	if err != nil {
		return fmt.Errorf("failed to start mDNS search: %v", err)
	}

	ds.isRunning = true
	fmt.Fprintf(os.Stderr, "Starting discovery of NP services via mDNS...\n")
	return nil
}

// StopBrowse stops service discovery
func (ds *DiscoveryService) StopBrowse() {
	if ds.isRunning && ds.stopBrowse != nil {
		ds.stopBrowse()
		ds.isRunning = false
		fmt.Fprintf(os.Stderr, "mDNS service discovery stopped\n")
	}
}

// addService adds a discovered service to the list
func (ds *DiscoveryService) addService(entry *zeroconf.ServiceEntry) {
	ds.mutex.Lock()
	defer ds.mutex.Unlock()

	// Extract service information
	service := ServiceInfo{
		Name:      entry.Instance,
		Host:      entry.HostName,
		Port:      entry.Port,
		Protocol:  "udp", // Default to UDP
		Addresses: make([]string, 0),
		Text:      entry.Text,
		TTL:       entry.TTL,
		IsTCP:     false, // Default to UDP
	}

	// Check for protocol information
	for _, text := range entry.Text {
		if text == "proto=tcp" {
			service.Protocol = "tcp"
			service.IsTCP = true
		}
	}

	// Get IP addresses
	for _, addr := range entry.AddrIPv4 {
		service.Addresses = append(service.Addresses, addr.String())
	}
	for _, addr := range entry.AddrIPv6 {
		service.Addresses = append(service.Addresses, addr.String())
	}

	// Add the service to the list
	serviceId := fmt.Sprintf("%s:%d", service.Name, service.Port)
	ds.services[serviceId] = service

	fmt.Fprintf(os.Stderr, "Discovered NP service: %s at %s:%d (%s)\n",
		service.Name, service.Host, service.Port, service.Protocol)
}

// GetServices returns the list of discovered services
func (ds *DiscoveryService) GetServices() []ServiceInfo {
	ds.mutex.Lock()
	defer ds.mutex.Unlock()

	result := make([]ServiceInfo, 0, len(ds.services))
	for _, service := range ds.services {
		result = append(result, service)
	}
	return result
}

// FindService searches for a service with a timeout
func (ds *DiscoveryService) FindService(timeout time.Duration) ([]ServiceInfo, error) {
	// Start discovery
	err := ds.StartBrowse()
	if err != nil {
		return nil, err
	}

	// Wait for timeout
	time.Sleep(timeout)

	// Stop discovery
	ds.StopBrowse()

	// Return found services
	services := ds.GetServices()
	if len(services) == 0 {
		return nil, fmt.Errorf("no NP services found on the network")
	}

	return services, nil
}

// ChooseServiceInteractive allows the user to choose a service interactively
func (ds *DiscoveryService) ChooseServiceInteractive() (*ServiceInfo, error) {
	fmt.Println("Searching for NP services on the local network...")

	services, err := ds.FindService(DISCOVERY_TIMEOUT)
	if err != nil {
		return nil, err
	}

	// Show found services
	fmt.Println("\nAvailable NP services:")
	for i, service := range services {
		addr := ""
		if len(service.Addresses) > 0 {
			addr = service.Addresses[0]
		}
		fmt.Printf("%d) %s at %s:%d (%s)\n", i+1, service.Name, addr, service.Port, service.Protocol)
	}

	// Ask the user which service to connect to
	var choice int
	for {
		fmt.Print("\nChoose a service (1-" + strconv.Itoa(len(services)) + "): ")
		_, err := fmt.Scanf("%d", &choice)
		if err == nil && choice >= 1 && choice <= len(services) {
			break
		}
		fmt.Println("Invalid choice. Try again.")
	}

	// Return the chosen service
	return &services[choice-1], nil
}

// Close releases discovery service resources
func (ds *DiscoveryService) Close() error {
	ds.StopAnnounce()
	ds.StopBrowse()
	return nil
}
