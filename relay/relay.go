package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"sync"
	"time"
)

// RelayConfig stores the configuration for the relay server
type RelayConfig struct {
	TCPPort        int
	HTTPPort       int
	HTTPSPort      int
	TLSCertFile    string
	TLSKeyFile     string
	EnableHTTP     bool
	EnableHTTPS    bool
	EnableTCP      bool
	DebugMode      bool
	MaxConnections int
	IdleTimeout    time.Duration
}

// RelayServer represents the relay server instance
type RelayServer struct {
	config      *RelayConfig
	sessions    map[string]*RelaySession
	sessionsMu  sync.RWMutex
	tcpListener net.Listener
}

// RelaySession represents a relay session between two clients
type RelaySession struct {
	ID        string
	CreatedAt time.Time
	LastUsed  time.Time
	Clients   [2]net.Conn
	Active    bool
	mu        sync.RWMutex
}

// NewRelayServer creates a new relay server with the given configuration
func NewRelayServer(config *RelayConfig) *RelayServer {
	return &RelayServer{
		config:   config,
		sessions: make(map[string]*RelaySession),
	}
}

// Start starts the relay server
func (rs *RelayServer) Start() error {
	// Start TCP server if enabled
	if rs.config.EnableTCP {
		go rs.startTCPServer()
	}

	// Start HTTP server if enabled
	if rs.config.EnableHTTP {
		go rs.startHTTPServer()
	}

	// Start HTTPS server if enabled
	if rs.config.EnableHTTPS {
		go rs.startHTTPSServer()
	}

	// Start session cleaner
	go rs.cleanupSessions()

	// Keep the main goroutine alive
	select {}
}

// startTCPServer starts the TCP server
func (rs *RelayServer) startTCPServer() error {
	addr := fmt.Sprintf(":%d", rs.config.TCPPort)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to start TCP server: %v", err)
	}

	rs.tcpListener = listener
	log.Printf("TCP relay server listening on %s", addr)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Error accepting connection: %v", err)
			continue
		}

		go rs.handleTCPConnection(conn)
	}
}

// handleTCPConnection handles a new TCP connection
func (rs *RelayServer) handleTCPConnection(conn net.Conn) {
	defer conn.Close()

	// Read the session ID from the connection
	buffer := make([]byte, 64)
	n, err := conn.Read(buffer)
	if err != nil {
		log.Printf("Error reading session ID: %v", err)
		return
	}

	sessionID := string(buffer[:n])

	if rs.config.DebugMode {
		log.Printf("New connection for session: %s from %s", sessionID, conn.RemoteAddr())
	}

	rs.sessionsMu.Lock()
	session, exists := rs.sessions[sessionID]

	if !exists {
		// Create a new session
		session = &RelaySession{
			ID:        sessionID,
			CreatedAt: time.Now(),
			LastUsed:  time.Now(),
			Active:    true,
		}
		session.Clients[0] = conn
		rs.sessions[sessionID] = session
		rs.sessionsMu.Unlock()

		// Wait for the second client to connect
		if rs.config.DebugMode {
			log.Printf("Created new session: %s, waiting for peer", sessionID)
		}

		// Send acknowledgment to the first client
		conn.Write([]byte("WAITING"))
		return
	}

	// If the session exists but already has two clients, reject
	if session.Clients[0] != nil && session.Clients[1] != nil {
		rs.sessionsMu.Unlock()
		conn.Write([]byte("SESSION_FULL"))
		log.Printf("Session %s is full, rejecting connection from %s", sessionID, conn.RemoteAddr())
		return
	}

	// Add the second client to the session
	session.Clients[1] = conn
	session.LastUsed = time.Now()
	rs.sessionsMu.Unlock()

	if rs.config.DebugMode {
		log.Printf("Second client connected to session %s from %s", sessionID, conn.RemoteAddr())
	}

	// Notify both clients that the session is ready
	session.Clients[0].Write([]byte("CONNECTED"))
	session.Clients[1].Write([]byte("CONNECTED"))

	// Start relaying data between the clients
	go rs.relayData(session)
}

// relayData relays data between the two clients in a session
func (rs *RelayServer) relayData(session *RelaySession) {
	var wg sync.WaitGroup
	wg.Add(2)

	// Relay from client 0 to client 1
	go func() {
		defer wg.Done()
		rs.copyData(session.Clients[0], session.Clients[1], session)
	}()

	// Relay from client 1 to client 0
	go func() {
		defer wg.Done()
		rs.copyData(session.Clients[1], session.Clients[0], session)
	}()

	// Wait for both directions to complete
	wg.Wait()

	// Close the session
	rs.closeSession(session.ID)
}

// copyData copies data from src to dst and updates the session's LastUsed time
func (rs *RelayServer) copyData(src, dst net.Conn, session *RelaySession) {
	buffer := make([]byte, 4096)

	for {
		// Set read deadline if idle timeout is configured
		if rs.config.IdleTimeout > 0 {
			src.SetReadDeadline(time.Now().Add(rs.config.IdleTimeout))
		}

		n, err := src.Read(buffer)
		if err != nil {
			if err != io.EOF {
				log.Printf("Read error: %v", err)
			}
			break
		}

		// Update last used time
		session.mu.Lock()
		session.LastUsed = time.Now()
		session.mu.Unlock()

		// Write data to destination
		_, err = dst.Write(buffer[:n])
		if err != nil {
			log.Printf("Write error: %v", err)
			break
		}

		if rs.config.DebugMode {
			log.Printf("Relayed %d bytes from %s to %s", n, src.RemoteAddr(), dst.RemoteAddr())
		}
	}
}

// closeSession closes a session and its connections
func (rs *RelayServer) closeSession(sessionID string) {
	rs.sessionsMu.Lock()
	defer rs.sessionsMu.Unlock()

	session, exists := rs.sessions[sessionID]
	if !exists {
		return
	}

	// Close connections
	if session.Clients[0] != nil {
		session.Clients[0].Close()
	}
	if session.Clients[1] != nil {
		session.Clients[1].Close()
	}

	// Remove session
	delete(rs.sessions, sessionID)

	if rs.config.DebugMode {
		log.Printf("Closed session: %s", sessionID)
	}
}

// cleanupSessions periodically removes idle sessions
func (rs *RelayServer) cleanupSessions() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rs.sessionsMu.Lock()
		now := time.Now()

		for id, session := range rs.sessions {
			session.mu.RLock()
			idle := now.Sub(session.LastUsed)
			session.mu.RUnlock()

			// Close sessions idle for more than the configured timeout
			if idle > rs.config.IdleTimeout {
				if rs.config.DebugMode {
					log.Printf("Cleaning up idle session: %s (idle for %v)", id, idle)
				}

				// Close connections
				if session.Clients[0] != nil {
					session.Clients[0].Close()
				}
				if session.Clients[1] != nil {
					session.Clients[1].Close()
				}

				// Remove session
				delete(rs.sessions, id)
			}
		}

		rs.sessionsMu.Unlock()
	}
}

// startHTTPServer starts the HTTP server
func (rs *RelayServer) startHTTPServer() error {
	addr := fmt.Sprintf(":%d", rs.config.HTTPPort)

	// Create HTTP server
	server := &http.Server{
		Addr:    addr,
		Handler: http.HandlerFunc(rs.handleHTTPRequest),
	}

	log.Printf("HTTP relay server listening on %s", addr)
	err := server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		log.Printf("HTTP server error: %v", err)
		return err
	}

	return nil
}

// startHTTPSServer starts the HTTPS server
func (rs *RelayServer) startHTTPSServer() error {
	addr := fmt.Sprintf(":%d", rs.config.HTTPSPort)

	// Check if TLS certificate and key files exist
	if rs.config.TLSCertFile == "" || rs.config.TLSKeyFile == "" {
		log.Printf("TLS certificate or key file not specified, HTTPS server not started")
		return nil
	}

	// Create TLS configuration
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	// Create HTTPS server
	server := &http.Server{
		Addr:      addr,
		Handler:   http.HandlerFunc(rs.handleHTTPRequest),
		TLSConfig: tlsConfig,
	}

	log.Printf("HTTPS relay server listening on %s", addr)
	err := server.ListenAndServeTLS(rs.config.TLSCertFile, rs.config.TLSKeyFile)
	if err != nil && err != http.ErrServerClosed {
		log.Printf("HTTPS server error: %v", err)
		return err
	}

	return nil
}

// handleHTTPRequest handles HTTP/HTTPS requests
func (rs *RelayServer) handleHTTPRequest(w http.ResponseWriter, r *http.Request) {
	// Check if it's a relay request
	if r.URL.Path == "/relay" {
		rs.handleHTTPRelay(w, r)
		return
	}

	// Serve status page for root path
	if r.URL.Path == "/" {
		rs.serveStatusPage(w, r)
		return
	}

	// 404 for other paths
	http.NotFound(w, r)
}

// handleHTTPRelay handles relay requests over HTTP
func (rs *RelayServer) handleHTTPRelay(w http.ResponseWriter, r *http.Request) {
	// Get session ID from query parameter
	sessionID := r.URL.Query().Get("session")
	if sessionID == "" {
		http.Error(w, "Missing session ID", http.StatusBadRequest)
		return
	}

	// Check if it's a WebSocket upgrade request
	// For now, we'll just use a simple HTTP connection

	// Create a connection wrapper for the HTTP connection
	conn := newHTTPConnection(w, r)

	// Handle the connection like a TCP connection
	rs.handleHTTPConnection(conn, sessionID)
}

// handleHTTPConnection handles an HTTP connection for relaying
func (rs *RelayServer) handleHTTPConnection(conn *httpConnection, sessionID string) {
	if rs.config.DebugMode {
		log.Printf("New HTTP connection for session: %s from %s", sessionID, conn.RemoteAddr())
	}

	rs.sessionsMu.Lock()
	session, exists := rs.sessions[sessionID]

	if !exists {
		// Create a new session
		session = &RelaySession{
			ID:        sessionID,
			CreatedAt: time.Now(),
			LastUsed:  time.Now(),
			Active:    true,
		}
		session.Clients[0] = conn
		rs.sessions[sessionID] = session
		rs.sessionsMu.Unlock()

		if rs.config.DebugMode {
			log.Printf("Created new HTTP session: %s, waiting for peer", sessionID)
		}

		// Send acknowledgment to the first client
		conn.Write([]byte("WAITING"))
		return
	}

	// If the session exists but already has two clients, reject
	if session.Clients[0] != nil && session.Clients[1] != nil {
		rs.sessionsMu.Unlock()
		conn.Write([]byte("SESSION_FULL"))
		log.Printf("Session %s is full, rejecting HTTP connection", sessionID)
		return
	}

	// Add the second client to the session
	session.Clients[1] = conn
	session.LastUsed = time.Now()
	rs.sessionsMu.Unlock()

	if rs.config.DebugMode {
		log.Printf("Second client connected to HTTP session %s", sessionID)
	}

	// Notify both clients that the session is ready
	session.Clients[0].Write([]byte("CONNECTED"))
	session.Clients[1].Write([]byte("CONNECTED"))

	// Start relaying data between the clients
	go rs.relayData(session)
}

// serveStatusPage serves a status page with information about the relay server
func (rs *RelayServer) serveStatusPage(w http.ResponseWriter, r *http.Request) {
	rs.sessionsMu.RLock()
	sessionCount := len(rs.sessions)
	rs.sessionsMu.RUnlock()

	fmt.Fprintf(w, "NP Relay Server\n")
	fmt.Fprintf(w, "---------------\n\n")
	fmt.Fprintf(w, "Active sessions: %d\n", sessionCount)
	fmt.Fprintf(w, "Server time: %s\n", time.Now().Format(time.RFC1123))
	fmt.Fprintf(w, "\nThis is a relay server for the NP (Network Pipe) tool.\n")
	fmt.Fprintf(w, "For more information, visit: https://github.com/lsferreira42/np\n")
}

// httpConnection implements the net.Conn interface for HTTP connections
type httpConnection struct {
	w          http.ResponseWriter
	r          *http.Request
	remoteAddr string
	localAddr  string
	readBuf    []byte
	closed     bool
}

// newHTTPConnection creates a new HTTP connection
func newHTTPConnection(w http.ResponseWriter, r *http.Request) *httpConnection {
	return &httpConnection{
		w:          w,
		r:          r,
		remoteAddr: r.RemoteAddr,
		localAddr:  r.Host,
		readBuf:    make([]byte, 0),
	}
}

// Read reads data from the HTTP connection
func (c *httpConnection) Read(b []byte) (n int, err error) {
	if c.closed {
		return 0, io.EOF
	}

	// If we have data in the buffer, return it
	if len(c.readBuf) > 0 {
		n = copy(b, c.readBuf)
		c.readBuf = c.readBuf[n:]
		return n, nil
	}

	// Otherwise, read from the request body
	return c.r.Body.Read(b)
}

// Write writes data to the HTTP connection
func (c *httpConnection) Write(b []byte) (n int, err error) {
	if c.closed {
		return 0, io.ErrClosedPipe
	}

	// Write to the response
	n, err = c.w.Write(b)
	if f, ok := c.w.(http.Flusher); ok {
		f.Flush()
	}
	return n, err
}

// Close closes the HTTP connection
func (c *httpConnection) Close() error {
	c.closed = true
	return nil
}

// LocalAddr returns the local network address
func (c *httpConnection) LocalAddr() net.Addr {
	return &addr{c.localAddr}
}

// RemoteAddr returns the remote network address
func (c *httpConnection) RemoteAddr() net.Addr {
	return &addr{c.remoteAddr}
}

// SetDeadline sets the read and write deadlines
func (c *httpConnection) SetDeadline(t time.Time) error {
	return nil // Not implemented for HTTP
}

// SetReadDeadline sets the read deadline
func (c *httpConnection) SetReadDeadline(t time.Time) error {
	return nil // Not implemented for HTTP
}

// SetWriteDeadline sets the write deadline
func (c *httpConnection) SetWriteDeadline(t time.Time) error {
	return nil // Not implemented for HTTP
}

// addr implements the net.Addr interface
type addr struct {
	address string
}

func (a *addr) Network() string {
	return "http"
}

func (a *addr) String() string {
	return a.address
}

func main() {
	// Parse command line flags
	tcpPort := flag.Int("tcp-port", 42421, "TCP port to listen on")
	httpPort := flag.Int("http-port", 80, "HTTP port to listen on")
	httpsPort := flag.Int("https-port", 443, "HTTPS port to listen on")
	tlsCert := flag.String("tls-cert", "", "TLS certificate file")
	tlsKey := flag.String("tls-key", "", "TLS key file")
	enableHTTP := flag.Bool("http", true, "Enable HTTP server")
	enableHTTPS := flag.Bool("https", false, "Enable HTTPS server")
	enableTCP := flag.Bool("tcp", true, "Enable TCP server")
	debugMode := flag.Bool("debug", false, "Enable debug mode")
	maxConn := flag.Int("max-connections", 1000, "Maximum number of concurrent connections")
	idleTimeout := flag.Duration("idle-timeout", 30*time.Minute, "Idle timeout for connections")

	flag.Parse()

	// Create server configuration
	config := &RelayConfig{
		TCPPort:        *tcpPort,
		HTTPPort:       *httpPort,
		HTTPSPort:      *httpsPort,
		TLSCertFile:    *tlsCert,
		TLSKeyFile:     *tlsKey,
		EnableHTTP:     *enableHTTP,
		EnableHTTPS:    *enableHTTPS,
		EnableTCP:      *enableTCP,
		DebugMode:      *debugMode,
		MaxConnections: *maxConn,
		IdleTimeout:    *idleTimeout,
	}

	// Create and start the relay server
	server := NewRelayServer(config)

	log.Printf("Starting NP Relay Server")
	log.Printf("TCP: %v (port %d)", config.EnableTCP, config.TCPPort)
	log.Printf("HTTP: %v (port %d)", config.EnableHTTP, config.HTTPPort)
	log.Printf("HTTPS: %v (port %d)", config.EnableHTTPS, config.HTTPSPort)

	err := server.Start()
	if err != nil {
		log.Fatalf("Failed to start relay server: %v", err)
	}
}
