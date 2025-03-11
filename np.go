package main

import (
	"bufio"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

// Network configuration defaults
const (
	DEFAULT_PORT = 4242
	DEFAULT_HOST = "127.0.0.1"
	DEFAULT_BIND = "0.0.0.0"
	BUFFER_SIZE  = 4096
)

// Authentication constants
const (
	AUTH_COMMAND  = "ISNP"
	AUTH_RESPONSE = "OK"
	AUTH_TIMEOUT  = 2 * time.Second
)

// Web UI defaults
const (
	DEFAULT_WEB_PORT = 8080
)

// Config holds all application configuration parameters
type Config struct {
	mode          string // "sender" or "receiver"
	port          int    // Port for the network connection
	host          string // Host to connect to (for sender mode)
	bindAddr      string // Address to bind to (for receiver mode)
	webUI         bool   // Whether to enable the web UI
	webUIPort     int    // Port for the web UI
	webUIBind     string // Address to bind web UI to
	useTCP        bool   // Use TCP instead of UDP
	enableMDNS    bool   // Enable multicast DNS discovery
	compression   string // Compression algorithm (none, gzip, zlib, zstd)
	compressLevel int    // Compression level (1-9)
	multiConn     bool   // Enable multiple connections
}

// ConnHandler is an interface for different connection types
type ConnHandler interface {
	Start() error
	Close() error
}

// NetworkPipe is the original (UDP) implementation
type NetworkPipe struct {
	config     *Config
	conn       *net.UDPConn
	bufferSize int
}

// askForMode prompts the user to select operational mode
func askForMode() string {
	fmt.Println("Select mode:")
	fmt.Println("1) Receiver - Listen for incoming connections")
	fmt.Println("2) Sender - Connect to a remote host")
	fmt.Print("\nChoice (1/2): ")

	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		choice := strings.TrimSpace(scanner.Text())
		if choice == "1" {
			return "receiver"
		} else if choice == "2" {
			return "sender"
		}
	}
	fmt.Println("Invalid choice, defaulting to receiver mode")
	return "receiver"
}

// parseFlags processes command line arguments and returns configuration
func parseFlags() *Config {
	config := &Config{}

	// Define command sets
	receiverCmd := flag.NewFlagSet("receiver", flag.ExitOnError)
	senderCmd := flag.NewFlagSet("sender", flag.ExitOnError)

	// Receiver flags
	receiverPort := receiverCmd.Int("p", DEFAULT_PORT, "Port to listen on")
	receiverPortLong := receiverCmd.Int("port", DEFAULT_PORT, "Port to listen on")
	receiverBind := receiverCmd.String("b", DEFAULT_BIND, "Address to bind to")
	receiverBindLong := receiverCmd.String("bind", DEFAULT_BIND, "Address to bind to")
	receiverWebUI := receiverCmd.Bool("web-ui", false, "Enable web interface")
	receiverWebUIPort := receiverCmd.Int("web-port", DEFAULT_WEB_PORT, "Port for web interface")
	receiverWebUIBind := receiverCmd.String("web-bind", DEFAULT_BIND, "Address to bind web interface to")
	receiverUseTCP := receiverCmd.Bool("tcp", false, "Use TCP instead of UDP")
	receiverEnableMDNS := receiverCmd.Bool("mdns", false, "Enable mDNS service announcement")
	receiverMultiConn := receiverCmd.Bool("multi", false, "Enable multiple connections")
	receiverCompression := receiverCmd.String("compression", "none", "Compression algorithm (none, gzip, zlib, zstd)")
	receiverCompressLevel := receiverCmd.Int("compress-level", 6, "Compression level (1-9)")

	// Sender flags
	senderPort := senderCmd.Int("p", DEFAULT_PORT, "Port to connect to")
	senderPortLong := senderCmd.Int("port", DEFAULT_PORT, "Port to connect to")
	senderHost := senderCmd.String("H", DEFAULT_HOST, "Host to connect to")
	senderHostLong := senderCmd.String("host", DEFAULT_HOST, "Host to connect to")
	senderWebUI := senderCmd.Bool("web-ui", false, "Enable web interface")
	senderWebUIPort := senderCmd.Int("web-port", DEFAULT_WEB_PORT, "Port for web interface")
	senderWebUIBind := senderCmd.String("web-bind", DEFAULT_BIND, "Address to bind web interface to")
	senderUseTCP := senderCmd.Bool("tcp", false, "Use TCP instead of UDP")
	senderEnableMDNS := senderCmd.Bool("mdns", false, "Enable mDNS service discovery")
	senderMultiConn := senderCmd.Bool("multi", false, "Enable connection to multiple servers")
	senderCompression := senderCmd.String("compression", "none", "Compression algorithm (none, gzip, zlib, zstd)")
	senderCompressLevel := senderCmd.Int("compress-level", 6, "Compression level (1-9)")

	// Check if any arguments were provided
	if len(os.Args) == 1 {
		config.mode = askForMode()
	} else {
		switch os.Args[1] {
		case "--receiver":
			config.mode = "receiver"
			receiverCmd.Parse(os.Args[2:])
		case "--sender":
			config.mode = "sender"
			senderCmd.Parse(os.Args[2:])
		default:
			fmt.Println("Error: Invalid mode specified")
			os.Exit(1)
		}
	}

	// Set configuration based on mode
	if config.mode == "receiver" {
		if receiverCmd.Parsed() {
			config.port = *receiverPort
			if *receiverPortLong != DEFAULT_PORT {
				config.port = *receiverPortLong
			}
			config.bindAddr = *receiverBind
			if *receiverBindLong != DEFAULT_BIND {
				config.bindAddr = *receiverBindLong
			}
			config.webUI = *receiverWebUI
			config.webUIPort = *receiverWebUIPort
			config.webUIBind = *receiverWebUIBind
			config.useTCP = *receiverUseTCP
			config.enableMDNS = *receiverEnableMDNS
			config.multiConn = *receiverMultiConn
			config.compression = *receiverCompression
			config.compressLevel = *receiverCompressLevel
		} else {
			config.port = DEFAULT_PORT
			config.bindAddr = DEFAULT_BIND
			config.webUI = false
			config.webUIPort = DEFAULT_WEB_PORT
			config.webUIBind = DEFAULT_BIND
			config.useTCP = false
			config.enableMDNS = false
			config.multiConn = false
			config.compression = "none"
			config.compressLevel = 6
		}
	} else {
		if senderCmd.Parsed() {
			config.port = *senderPort
			if *senderPortLong != DEFAULT_PORT {
				config.port = *senderPortLong
			}
			config.host = *senderHost
			if *senderHostLong != DEFAULT_HOST {
				config.host = *senderHostLong
			}
			config.webUI = *senderWebUI
			config.webUIPort = *senderWebUIPort
			config.webUIBind = *senderWebUIBind
			config.useTCP = *senderUseTCP
			config.enableMDNS = *senderEnableMDNS
			config.multiConn = *senderMultiConn
			config.compression = *senderCompression
			config.compressLevel = *senderCompressLevel
		} else {
			config.port = DEFAULT_PORT
			config.host = DEFAULT_HOST
			config.webUI = false
			config.webUIPort = DEFAULT_WEB_PORT
			config.webUIBind = DEFAULT_BIND
			config.useTCP = false
			config.enableMDNS = false
			config.multiConn = false
			config.compression = "none"
			config.compressLevel = 6
		}
	}

	return config
}

// NewNetworkPipe creates an instance of the original UDP pipe
func NewNetworkPipe(config *Config) (*NetworkPipe, error) {
	np := &NetworkPipe{
		config:     config,
		bufferSize: BUFFER_SIZE,
	}

	var bindAddr string
	if config.mode == "receiver" {
		bindAddr = config.bindAddr
	} else {
		bindAddr = "0.0.0.0"
	}

	addr := &net.UDPAddr{
		IP:   net.ParseIP(bindAddr),
		Port: config.port,
	}

	var err error
	np.conn, err = net.ListenUDP("udp", addr)
	if err != nil {
		if config.mode == "receiver" {
			if isNPRunning(bindAddr, config.port) {
				fmt.Fprintf(os.Stderr, "Another NP instance is already running and listening\n")
				os.Exit(1)
			}
			return nil, fmt.Errorf("port %d is in use by another application", config.port)
		}
		// For sending mode, use any available port
		np.conn, err = net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP(bindAddr), Port: 0})
		if err != nil {
			return nil, fmt.Errorf("failed to bind to any port: %v", err)
		}
	}

	return np, nil
}

// isNPRunning checks if an NP instance is already running
func isNPRunning(host string, port int) bool {
	conn, err := net.DialTimeout("udp", fmt.Sprintf("%s:%d", host, port), AUTH_TIMEOUT)
	if err != nil {
		return false
	}
	defer conn.Close()

	_, err = conn.Write([]byte(AUTH_COMMAND))
	if err != nil {
		return false
	}

	buffer := make([]byte, 64)
	conn.SetReadDeadline(time.Now().Add(AUTH_TIMEOUT))
	n, err := conn.Read(buffer)
	if err != nil {
		return false
	}

	return string(buffer[:n]) == AUTH_RESPONSE
}

func (np *NetworkPipe) handleAuth(data []byte, addr *net.UDPAddr) bool {
	if string(data) == AUTH_COMMAND {
		np.conn.WriteToUDP([]byte(AUTH_RESPONSE), addr)
		return true
	}
	return false
}

func (np *NetworkPipe) handleReceive(wg *sync.WaitGroup) {
	defer wg.Done()

	buffer := make([]byte, np.bufferSize)
	for {
		n, addr, err := np.conn.ReadFromUDP(buffer)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading: %v\n", err)
			return
		}

		if np.handleAuth(buffer[:n], addr) {
			continue
		}

		// Record for the web interface
		if np.config.webUI {
			content := string(buffer[:n])
			RecordReceivedData(uint64(n), addr.String())
			RecordMessage(content, "in", n, addr.String(), np.conn.LocalAddr().String())
		}

		os.Stdout.Write(buffer[:n])
		if !strings.HasSuffix(string(buffer[:n]), "\n") {
			os.Stdout.Write([]byte{'\n'})
		}
	}
}

func (np *NetworkPipe) handleSend(wg *sync.WaitGroup) {
	defer wg.Done()

	if !isNPRunning(np.config.host, np.config.port) {
		fmt.Fprintf(os.Stderr, "Warning: Remote host is not running NP or is unreachable\n")
		return
	}

	scanner := bufio.NewScanner(os.Stdin)
	remoteAddr := &net.UDPAddr{
		IP:   net.ParseIP(np.config.host),
		Port: np.config.port,
	}

	for scanner.Scan() {
		data := scanner.Bytes()
		_, err := np.conn.WriteToUDP(data, remoteAddr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error sending: %v\n", err)
			return
		}

		// Record for the web interface
		if np.config.webUI {
			content := string(data)
			size := len(data)
			RecordSentData(uint64(size), remoteAddr.String())
			RecordMessage(content, "out", size, np.conn.LocalAddr().String(), remoteAddr.String())
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading from stdin: %v\n", err)
	}
}

func (np *NetworkPipe) Start() error {
	var wg sync.WaitGroup
	wg.Add(2)

	// Initialize the web interface, if enabled
	if np.config.webUI {
		webConfig := &WebUIConfig{
			Address: np.config.webUIBind,
			Port:    np.config.webUIPort,
			Enabled: true,
		}
		StartWebUI(webConfig, np.config)
	}

	go np.handleReceive(&wg)

	if np.config.mode == "sender" {
		go np.handleSend(&wg)
	}

	wg.Wait()
	return nil
}

func (np *NetworkPipe) Close() error {
	if np.conn != nil {
		return np.conn.Close()
	}
	return nil
}

// getCompressType gets the compression type from the string
func getCompressType(compression string) CompressionType {
	switch strings.ToLower(compression) {
	case "gzip":
		return GzipCompression
	case "zlib":
		return ZlibCompression
	case "zstd":
		return ZstdCompression
	default:
		return NoCompression
	}
}

// createConnHandler creates the appropriate connection handler based on the configuration
func createConnHandler(config *Config) (ConnHandler, error) {
	// If using TCP
	if config.useTCP {
		tcpPipe, err := NewTCPPipe(config)
		if err != nil {
			return nil, err
		}

		// If multiple connections, configure the multiplex
		if config.multiConn {
			manager := NewMultiplexManager(config)

			// Configure compression, if requested
			if config.compression != "none" {
				compType := getCompressType(config.compression)
				manager.SetCompression(compType, config.compressLevel)
			}

			// For TCP, the multiplex manager is managed by TCPPipe
			tcpPipe.SetMultiplexManager(manager)
		}

		// Configure mDNS discovery, if requested
		if config.enableMDNS {
			discovery := NewDiscoveryService(config)

			if config.mode == "receiver" {
				// Announce the service on the network
				serviceName := fmt.Sprintf("NP Server (%s)", config.bindAddr)
				err := discovery.StartAnnounce(serviceName, config.port, config.useTCP)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Warning: Failed to announce mDNS service: %v\n", err)
				} else {
					fmt.Fprintf(os.Stderr, "Announced service via mDNS\n")
				}
			} else {
				// Discover services on the network
				err := discovery.StartBrowse()
				if err != nil {
					fmt.Fprintf(os.Stderr, "Warning: Failed to start mDNS discovery: %v\n", err)
				} else {
					fmt.Fprintf(os.Stderr, "Started mDNS discovery for NP services\n")

					// If no specific host is provided, try to find one via mDNS
					if config.host == DEFAULT_HOST {
						fmt.Fprintf(os.Stderr, "Looking for NP services on the network...\n")
						// Wait a few seconds to discover services
						time.Sleep(2 * time.Second)

						services := discovery.GetServices()
						if len(services) > 0 {
							// Use the first service found
							service := services[0]
							fmt.Fprintf(os.Stderr, "Found NP service: %s at %s:%d\n",
								service.Name, service.Host, service.Port)

							config.host = service.Host
							config.port = service.Port
							config.useTCP = service.IsTCP
						}
					}
				}
			}

			// Set the discovery service for the TCPPipe
			tcpPipe.SetDiscoveryService(discovery)
		}

		return tcpPipe, nil
	}

	// Otherwise, use the original NetworkPipe (UDP)
	return NewNetworkPipe(config)
}

func main() {
	config := parseFlags()

	// Create the appropriate connection handler
	handler, err := createConnHandler(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer handler.Close()

	// Display configuration information
	if config.mode == "receiver" {
		protocol := "UDP"
		if config.useTCP {
			protocol = "TCP"
		}

		fmt.Fprintf(os.Stderr, "Listening on %s:%d (%s)\n", config.bindAddr, config.port, protocol)

		if config.multiConn {
			fmt.Fprintf(os.Stderr, "Multiple connections mode enabled\n")
		}

		if config.compression != "none" {
			fmt.Fprintf(os.Stderr, "Compression enabled: %s (level %d)\n",
				config.compression, config.compressLevel)
		}

		if config.enableMDNS {
			fmt.Fprintf(os.Stderr, "mDNS service announcement enabled\n")
		}

		if config.webUI {
			fmt.Fprintf(os.Stderr, "Web interface available at http://%s:%d\n",
				config.webUIBind, config.webUIPort)
		}
	} else {
		protocol := "UDP"
		if config.useTCP {
			protocol = "TCP"
		}

		fmt.Fprintf(os.Stderr, "Connected to %s:%d (%s)\n", config.host, config.port, protocol)

		if config.multiConn {
			fmt.Fprintf(os.Stderr, "Multiple connections mode enabled\n")
		}

		if config.compression != "none" {
			fmt.Fprintf(os.Stderr, "Compression enabled: %s (level %d)\n",
				config.compression, config.compressLevel)
		}

		if config.enableMDNS {
			fmt.Fprintf(os.Stderr, "mDNS service discovery enabled\n")
		}

		if config.webUI {
			fmt.Fprintf(os.Stderr, "Web interface available at http://%s:%d\n",
				config.webUIBind, config.webUIPort)
		}
	}

	if err := handler.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
