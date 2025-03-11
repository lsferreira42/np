package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"time"
)

// TCPPipe implements TCP communication for the Network Pipe
// It handles connection establishment, data transfer, and cleanup
type TCPPipe struct {
	config       *Config             // Application configuration
	listener     net.Listener        // TCP listener for receiver mode
	conn         net.Conn            // Single TCP connection for sender mode
	bufferSize   int                 // Buffer size for data transfer
	clients      map[string]net.Conn // Connected clients (for receiver mode)
	clientsMutex sync.RWMutex        // Mutex for thread-safe client map access
	multiplexer  *MultiplexManager   // Optional multiplexing manager
	discovery    *DiscoveryService   // Optional service discovery
}

// NewTCPPipe creates a new TCP pipe instance based on configuration
func NewTCPPipe(config *Config) (*TCPPipe, error) {
	pipe := &TCPPipe{
		config:     config,
		bufferSize: BUFFER_SIZE,
		clients:    make(map[string]net.Conn),
	}

	// For receiver mode, create a TCP listener
	if config.mode == "receiver" {
		var err error
		addr := fmt.Sprintf("%s:%d", config.bindAddr, config.port)
		pipe.listener, err = net.Listen("tcp", addr)
		if err != nil {
			return nil, fmt.Errorf("failed to start TCP listener: %v", err)
		}
	} else {
		// For sender mode, establish a connection to the server
		var err error
		addr := fmt.Sprintf("%s:%d", config.host, config.port)
		pipe.conn, err = net.Dial("tcp", addr)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to TCP server: %v", err)
		}
	}

	return pipe, nil
}

// SetMultiplexManager assigns a multiplexing manager to this TCP pipe
func (pipe *TCPPipe) SetMultiplexManager(manager *MultiplexManager) {
	pipe.multiplexer = manager
}

// SetDiscoveryService assigns a discovery service to this TCP pipe
func (pipe *TCPPipe) SetDiscoveryService(discovery *DiscoveryService) {
	pipe.discovery = discovery
}

// Start initializes the TCP pipe operation based on configured mode
func (pipe *TCPPipe) Start() error {
	// Initialize web interface if enabled
	if pipe.config.webUI {
		webConfig := &WebUIConfig{
			Address: pipe.config.webUIBind,
			Port:    pipe.config.webUIPort,
			Enabled: true,
		}
		StartWebUI(webConfig, pipe.config)
	}

	// Execute mode-specific startup
	if pipe.config.mode == "receiver" {
		return pipe.acceptConnections()
	}

	// Sender mode
	return pipe.handleSend()
}

// acceptConnections handles incoming TCP connections in receiver mode
// This is a blocking function that runs until the application is terminated
func (pipe *TCPPipe) acceptConnections() error {
	fmt.Fprintf(os.Stderr, "TCP: Accepting connections on %s\n", pipe.listener.Addr())

	var wg sync.WaitGroup

	for {
		// Accept a new connection
		conn, err := pipe.listener.Accept()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error accepting connection: %v\n", err)
			continue
		}

		// Register the client
		clientID := conn.RemoteAddr().String()
		pipe.clientsMutex.Lock()
		pipe.clients[clientID] = conn
		pipe.clientsMutex.Unlock()

		fmt.Fprintf(os.Stderr, "New connection from %s\n", clientID)

		// If using multiplex, add to the manager
		if pipe.multiplexer != nil {
			pipe.multiplexer.AddConnection(clientID, conn)
		}

		// Record for the web interface, if enabled
		if pipe.config.webUI {
			RecordMessage("New TCP connection", "system", 0, conn.RemoteAddr().String(), conn.LocalAddr().String())
		}

		// Start goroutine to handle the client
		wg.Add(1)
		go func(c net.Conn, id string) {
			defer wg.Done()
			pipe.handleClient(c, id)
		}(conn, clientID)
	}
}

// handleClient manages communication with an individual client
func (pipe *TCPPipe) handleClient(conn net.Conn, clientID string) {
	defer func() {
		conn.Close()
		pipe.clientsMutex.Lock()
		delete(pipe.clients, clientID)
		pipe.clientsMutex.Unlock()

		// If using multiplex, remove from the manager
		if pipe.multiplexer != nil {
			pipe.multiplexer.RemoveConnection(clientID)
		}

		// Record for the web interface, if enabled
		if pipe.config.webUI {
			RecordMessage("TCP connection closed", "system", 0, conn.RemoteAddr().String(), conn.LocalAddr().String())
		}

		fmt.Fprintf(os.Stderr, "Connection from %s closed\n", clientID)
	}()

	buffer := make([]byte, pipe.bufferSize)

	for {
		// If using multiplex, the manager handles reception
		if pipe.multiplexer != nil {
			time.Sleep(100 * time.Millisecond) // Avoid excessive CPU usage
			continue
		}

		// Read data from client
		n, err := conn.Read(buffer)
		if err != nil {
			if err != io.EOF {
				fmt.Fprintf(os.Stderr, "Error reading from client %s: %v\n", clientID, err)
			}
			break
		}

		if n > 0 {
			data := buffer[:n]

			// Record for the web interface, if enabled
			if pipe.config.webUI {
				content := string(data)
				RecordReceivedData(uint64(n), conn.RemoteAddr().String())
				RecordMessage(content, "in", n, conn.RemoteAddr().String(), conn.LocalAddr().String())
			}

			// Write data to standard output
			os.Stdout.Write(data)
		}
	}
}

// handleSend manages sending data to the server
func (pipe *TCPPipe) handleSend() error {
	if pipe.conn == nil {
		return fmt.Errorf("connection not established with server")
	}

	defer pipe.conn.Close()
	fmt.Fprintf(os.Stderr, "TCP: Connected to %s\n", pipe.conn.RemoteAddr())

	// If using multiplex, add the connection to the manager
	if pipe.multiplexer != nil {
		clientID := pipe.conn.RemoteAddr().String()
		pipe.multiplexer.AddConnection(clientID, pipe.conn)

		// Start listening in goroutine
		go pipe.multiplexer.StartListening(func(id string, data []byte) {
			// Process data received via multiplex
			os.Stdout.Write(data)
		})
	} else {
		// Start goroutine to receive data from the server
		go pipe.handleReceive()
	}

	// Read from standard input and send to the server
	buffer := make([]byte, pipe.bufferSize)
	for {
		n, err := os.Stdin.Read(buffer)
		if err != nil {
			if err != io.EOF {
				fmt.Fprintf(os.Stderr, "Error reading from standard input: %v\n", err)
			}
			break
		}

		if n > 0 {
			data := buffer[:n]

			// If using multiplex, send via manager
			if pipe.multiplexer != nil {
				clientID := pipe.conn.RemoteAddr().String()
				err = pipe.multiplexer.SendTo(clientID, data)
			} else {
				// Send directly
				_, err = pipe.conn.Write(data)

				// Record for the web interface, if enabled
				if err == nil && pipe.config.webUI {
					content := string(data)
					RecordSentData(uint64(n), pipe.conn.RemoteAddr().String())
					RecordMessage(content, "out", n, pipe.conn.LocalAddr().String(), pipe.conn.RemoteAddr().String())
				}
			}

			if err != nil {
				fmt.Fprintf(os.Stderr, "Error sending data: %v\n", err)
				break
			}
		}
	}

	return nil
}

// handleReceive manages receiving data from the server
func (pipe *TCPPipe) handleReceive() {
	buffer := make([]byte, pipe.bufferSize)

	for {
		n, err := pipe.conn.Read(buffer)
		if err != nil {
			if err != io.EOF {
				fmt.Fprintf(os.Stderr, "Error receiving data: %v\n", err)
			}
			break
		}

		if n > 0 {
			data := buffer[:n]

			// Record for the web interface, if enabled
			if pipe.config.webUI {
				content := string(data)
				RecordReceivedData(uint64(n), pipe.conn.RemoteAddr().String())
				RecordMessage(content, "in", n, pipe.conn.RemoteAddr().String(), pipe.conn.LocalAddr().String())
			}

			// Write data to standard output
			os.Stdout.Write(data)
		}
	}
}

// Close closes all connections
func (pipe *TCPPipe) Close() error {
	var lastErr error

	// Close the listener, if it exists
	if pipe.listener != nil {
		if err := pipe.listener.Close(); err != nil {
			lastErr = err
			fmt.Fprintf(os.Stderr, "Error closing listener: %v\n", err)
		}
	}

	// Close the main connection, if it exists
	if pipe.conn != nil {
		if err := pipe.conn.Close(); err != nil {
			lastErr = err
			fmt.Fprintf(os.Stderr, "Error closing main connection: %v\n", err)
		}
	}

	// Close all client connections
	pipe.clientsMutex.Lock()
	for id, conn := range pipe.clients {
		if err := conn.Close(); err != nil {
			lastErr = err
			fmt.Fprintf(os.Stderr, "Error closing connection with client %s: %v\n", id, err)
		}
	}
	pipe.clientsMutex.Unlock()

	// Close the discovery service, if it exists
	if pipe.discovery != nil {
		if err := pipe.discovery.Close(); err != nil {
			lastErr = err
			fmt.Fprintf(os.Stderr, "Error closing discovery service: %v\n", err)
		}
	}

	// Close the multiplexer, if it exists
	if pipe.multiplexer != nil {
		pipe.multiplexer.Close()
	}

	return lastErr
}
