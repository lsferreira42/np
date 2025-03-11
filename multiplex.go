package main

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"fmt"
	"io"
	"net"
	"os"
	"sync"

	"github.com/klauspost/compress/zstd"
)

// CompressionType defines the compression algorithm to use
type CompressionType int

// Supported compression types
const (
	NoCompression CompressionType = iota
	GzipCompression
	ZlibCompression
	ZstdCompression
)

// CompressionHeader contains the byte signatures that identify compressed data formats
// These are used to automatically detect the compression type of incoming data
var CompressionHeader = map[CompressionType][]byte{
	GzipCompression: []byte{0x1F, 0x8B},             // Gzip magic header
	ZlibCompression: []byte{0x78, 0x9C},             // Zlib default compression
	ZstdCompression: []byte{0x28, 0xB5, 0x2F, 0xFD}, // Zstandard frame magic
}

// ZstdReadCloser is a wrapper that implements io.ReadCloser for zstd.Decoder
// This is needed because zstd.Decoder alone doesn't properly implement the interface
type ZstdReadCloser struct {
	*zstd.Decoder
}

// Close implements io.Closer for the zstd decoder
func (z *ZstdReadCloser) Close() error {
	z.Decoder.Close()
	return nil
}

// MultiplexManager handles multiple network connections and applies compression
// It serves as an abstraction layer for sending and receiving data across all connections
type MultiplexManager struct {
	config        *Config                   // Application configuration
	connections   map[string]net.Conn       // Active connections by ID
	mutex         sync.RWMutex              // Mutex for thread-safe connection access
	compression   CompressionType           // Active compression algorithm
	compressLevel int                       // Compression level (1-9)
	encoders      map[string]io.WriteCloser // Compression encoders by connection ID
	decoders      map[string]io.ReadCloser  // Compression decoders by connection ID
}

// NewMultiplexManager creates a new multiplexing manager
func NewMultiplexManager(config *Config) *MultiplexManager {
	return &MultiplexManager{
		config:      config,
		connections: make(map[string]net.Conn),
		encoders:    make(map[string]io.WriteCloser),
		decoders:    make(map[string]io.ReadCloser),
		compression: NoCompression,
	}
}

// SetCompression configures the compression type and level to be used
func (mm *MultiplexManager) SetCompression(compType CompressionType, level int) {
	mm.compression = compType
	mm.compressLevel = level
}

// GetCompressionName returns a human-readable name for a compression type
func GetCompressionName(compType CompressionType) string {
	switch compType {
	case NoCompression:
		return "None"
	case GzipCompression:
		return "Gzip"
	case ZlibCompression:
		return "Zlib"
	case ZstdCompression:
		return "Zstandard"
	default:
		return "Unknown"
	}
}

// AddConnection registers a new connection with the multiplexer
func (mm *MultiplexManager) AddConnection(id string, conn net.Conn) {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	mm.connections[id] = conn

	// Log the new connection if web UI is enabled
	if mm.config.webUI {
		RecordMessage("Multiplexed connection added", "system", 0, conn.RemoteAddr().String(), conn.LocalAddr().String())
	}

	fmt.Fprintf(os.Stderr, "Multiplex: Added connection %s: %s -> %s\n",
		id, conn.RemoteAddr().String(), conn.LocalAddr().String())
}

// RemoveConnection removes a connection from the manager
func (mm *MultiplexManager) RemoveConnection(id string) {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	if conn, exists := mm.connections[id]; exists {
		// Close the connection
		conn.Close()

		// Close compressors/decompressors
		if encoder, ok := mm.encoders[id]; ok {
			encoder.Close()
			delete(mm.encoders, id)
		}

		if decoder, ok := mm.decoders[id]; ok {
			decoder.Close()
			delete(mm.decoders, id)
		}

		// Remove from the list
		delete(mm.connections, id)

		// Record for the web interface, if enabled
		if mm.config.webUI {
			RecordMessage("Multiplexed connection removed", "system", 0, conn.RemoteAddr().String(), conn.LocalAddr().String())
		}

		fmt.Fprintf(os.Stderr, "Multiplex: Removed connection %s\n", id)
	}
}

// GetConnection returns a connection by ID
func (mm *MultiplexManager) GetConnection(id string) (net.Conn, bool) {
	mm.mutex.RLock()
	defer mm.mutex.RUnlock()

	conn, exists := mm.connections[id]
	return conn, exists
}

// GetConnections returns all connections
func (mm *MultiplexManager) GetConnections() map[string]net.Conn {
	mm.mutex.RLock()
	defer mm.mutex.RUnlock()

	// Copy the map to avoid concurrency issues
	result := make(map[string]net.Conn)
	for id, conn := range mm.connections {
		result[id] = conn
	}

	return result
}

// NumConnections returns the number of active connections
func (mm *MultiplexManager) NumConnections() int {
	mm.mutex.RLock()
	defer mm.mutex.RUnlock()

	return len(mm.connections)
}

// SendToAll sends data to all connections
func (mm *MultiplexManager) SendToAll(data []byte) {
	mm.mutex.RLock()
	connections := make(map[string]net.Conn, len(mm.connections))
	for id, conn := range mm.connections {
		connections[id] = conn
	}
	mm.mutex.RUnlock()

	var wg sync.WaitGroup
	for id, conn := range connections {
		wg.Add(1)
		go func(connID string, c net.Conn) {
			defer wg.Done()
			mm.SendTo(connID, data)
		}(id, conn)
	}

	wg.Wait()
}

// SendTo sends data to a specific connection, with compression if configured
func (mm *MultiplexManager) SendTo(id string, data []byte) error {
	mm.mutex.Lock()
	conn, exists := mm.connections[id]
	if !exists {
		mm.mutex.Unlock()
		return fmt.Errorf("connection %s not found", id)
	}

	// If no compression, send directly
	if mm.compression == NoCompression {
		mm.mutex.Unlock()
		_, err := conn.Write(data)

		// Record for the web interface
		if err == nil && mm.config.webUI {
			remoteAddr := conn.RemoteAddr().String()
			RecordSentData(uint64(len(data)), remoteAddr)
			RecordMessage(string(data), "out", len(data), conn.LocalAddr().String(), remoteAddr)
		}

		return err
	}

	// Get or create a compressor for this connection
	encoder, ok := mm.encoders[id]
	if !ok {
		var err error
		var buf bytes.Buffer

		switch mm.compression {
		case GzipCompression:
			encoder, err = gzip.NewWriterLevel(&buf, mm.compressLevel)
		case ZlibCompression:
			encoder, err = zlib.NewWriterLevel(&buf, mm.compressLevel)
		case ZstdCompression:
			encoder, err = zstd.NewWriter(&buf)
		default:
			mm.mutex.Unlock()
			return fmt.Errorf("unsupported compression type")
		}

		if err != nil {
			mm.mutex.Unlock()
			return fmt.Errorf("error creating compressor: %v", err)
		}

		mm.encoders[id] = encoder
	}

	mm.mutex.Unlock()

	// Compress the data
	var buf bytes.Buffer
	writer, ok := encoder.(io.Writer)
	if !ok {
		return fmt.Errorf("error getting compressor writer")
	}

	_, err := writer.Write(data)
	if err != nil {
		return fmt.Errorf("error compressing data: %v", err)
	}

	// Get the compressed data
	if flusher, ok := encoder.(interface{ Flush() error }); ok {
		err = flusher.Flush()
		if err != nil {
			return fmt.Errorf("error flushing compressor: %v", err)
		}
	}

	// Send the compressed data
	_, err = conn.Write(buf.Bytes())

	// Record for the web interface
	if err == nil && mm.config.webUI {
		remoteAddr := conn.RemoteAddr().String()
		RecordSentData(uint64(buf.Len()), remoteAddr)
		recordMsg := fmt.Sprintf("[Compressed: %s] %s", GetCompressionName(mm.compression), string(data))
		RecordMessage(recordMsg, "out", buf.Len(), conn.LocalAddr().String(), remoteAddr)
	}

	return err
}

// ReceiveFrom receives data from a specific connection, decompressing if necessary
func (mm *MultiplexManager) ReceiveFrom(id string, buffer []byte) (int, error) {
	mm.mutex.Lock()
	conn, exists := mm.connections[id]
	if !exists {
		mm.mutex.Unlock()
		return 0, fmt.Errorf("connection %s not found", id)
	}

	// Read data from the connection
	n, err := conn.Read(buffer)
	if err != nil {
		mm.mutex.Unlock()
		return 0, err
	}

	// Check if the data is compressed
	compType := NoCompression
	data := buffer[:n]

	for t, header := range CompressionHeader {
		if n >= len(header) && bytes.Equal(data[:len(header)], header) {
			compType = t
			break
		}
	}

	// If not compressed, return the data as is
	if compType == NoCompression {
		mm.mutex.Unlock()

		// Record for the web interface
		if mm.config.webUI {
			remoteAddr := conn.RemoteAddr().String()
			RecordReceivedData(uint64(n), remoteAddr)
			RecordMessage(string(data), "in", n, remoteAddr, conn.LocalAddr().String())
		}

		return n, nil
	}

	// Get or create a decompressor for this connection
	decoder, ok := mm.decoders[id]
	if !ok {
		var err error
		buf := bytes.NewReader(data)

		switch compType {
		case GzipCompression:
			decoder, err = gzip.NewReader(buf)
		case ZlibCompression:
			decoder, err = zlib.NewReader(buf)
		case ZstdCompression:
			zstdDecoder, err := zstd.NewReader(buf)
			if err != nil {
				mm.mutex.Unlock()
				return 0, fmt.Errorf("error creating zstd decompressor: %v", err)
			}
			decoder = &ZstdReadCloser{zstdDecoder}
		default:
			mm.mutex.Unlock()
			return 0, fmt.Errorf("unrecognized compression format")
		}

		if err != nil {
			mm.mutex.Unlock()
			return 0, fmt.Errorf("error creating decompressor: %v", err)
		}

		mm.decoders[id] = decoder
	}

	mm.mutex.Unlock()

	// Decompress the data
	var buf bytes.Buffer
	_, err = io.Copy(&buf, decoder)
	if err != nil {
		return 0, fmt.Errorf("error decompressing data: %v", err)
	}

	// Copy the decompressed data to the buffer
	decompressed := buf.Bytes()
	if len(decompressed) > len(buffer) {
		return 0, fmt.Errorf("buffer too small for decompressed data")
	}

	copy(buffer, decompressed)

	// Record for the web interface
	if mm.config.webUI {
		remoteAddr := conn.RemoteAddr().String()
		RecordReceivedData(uint64(n), remoteAddr)
		recordMsg := fmt.Sprintf("[Decompressed: %s] %s", GetCompressionName(compType), string(decompressed))
		RecordMessage(recordMsg, "in", n, remoteAddr, conn.LocalAddr().String())
	}

	return len(decompressed), nil
}

// StartListening starts listening on all connections
func (mm *MultiplexManager) StartListening(handler func(id string, data []byte)) {
	connections := mm.GetConnections()

	for id, _ := range connections {
		go mm.listenConnection(id, handler)
	}

	fmt.Fprintf(os.Stderr, "Multiplex: Listening on %d connections\n", len(connections))
}

// listenConnection listens for data on a specific connection
func (mm *MultiplexManager) listenConnection(id string, handler func(id string, data []byte)) {
	buffer := make([]byte, BUFFER_SIZE)

	for {
		mm.mutex.RLock()
		_, exists := mm.connections[id]
		mm.mutex.RUnlock()

		if !exists {
			break
		}

		n, err := mm.ReceiveFrom(id, buffer)
		if err != nil {
			if err != io.EOF {
				fmt.Fprintf(os.Stderr, "Error receiving from %s: %v\n", id, err)
			}
			mm.RemoveConnection(id)
			break
		}

		if n > 0 {
			data := make([]byte, n)
			copy(data, buffer[:n])
			handler(id, data)
		}
	}
}

// Close closes all connections and cleans up resources
func (mm *MultiplexManager) Close() {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	for id, conn := range mm.connections {
		conn.Close()

		if encoder, ok := mm.encoders[id]; ok {
			encoder.Close()
		}

		if decoder, ok := mm.decoders[id]; ok {
			decoder.Close()
		}
	}

	mm.connections = make(map[string]net.Conn)
	mm.encoders = make(map[string]io.WriteCloser)
	mm.decoders = make(map[string]io.ReadCloser)
}
