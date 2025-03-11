# NP (Network Pipe)

NP is a command-line tool for creating bidirectional network pipes between machines. It works as a modern and intuitive alternative to netcat, with built-in support for service detection and a simple authentication protocol.

## What is NP?

NP (Network Pipe) allows you to easily connect the standard input and output (stdin/stdout) of two processes on different machines through the network using the UDP or TCP protocol. This tool is ideal for:

- Fast file transfers between machines
- Real-time communication (like a simple chat)
- Piping data between commands on different machines
- Network connection debugging
- Streaming logs or remote command outputs

## Features

- **Bidirectional Communication**: Support for UDP and TCP
- **Intuitive Interface**: Interactive mode for easy configuration
- **Basic Security**: Authentication verification between peers
- **Shell Integration**: Seamless integration with stdin/stdout for use with Unix pipes
- **Two Modes**: Receiver (server) and sender (client)
- **Monitoring**: Web interface for real-time visualization
- **Automatic Discovery**: Service location via mDNS (Bonjour/Avahi)
- **Multiple Connections**: Support for multiplex mode for simultaneous connections
- **Compression**: Real-time compression algorithms (gzip, zlib, zstd)
- **Portability**: Lightweight code written in Go, compatible with multiple platforms

## Installation

### Via Go Install
```bash
go install github.com/lsferreira42/np@latest
```

### Building from Source
```bash
git clone https://github.com/lsferreira42/np.git
cd np
go build
```

### Pre-compiled Binaries

You can find pre-compiled binaries for various platforms on the [releases page](https://github.com/lsferreira42/np/releases).

## Usage

NP operates in two main modes:

```bash
# Receiver mode (server): listens for incoming connections
np --receiver

# Sender mode (client): connects to a receiver
np --sender -H 192.168.1.100
```

For a complete list of detailed examples, including specific scenarios with Docker logs, Kubernetes, systemd, and log files, see the [Examples Guide](README_EXAMPLES.en.md).

## Web Interface

NP includes an integrated web interface that allows you to monitor connections, view traffic, and access real-time statistics.

### Enabling the Web Interface

```bash
# Enable with default settings (port 8080)
np --receiver --web-ui

# Specify custom port for the web interface
np --receiver --web-ui --web-port 9000
```

The interface is accessible through any modern web browser and updates data in real-time.

## Options

### Global Options
- `-p, --port`: Port for connection (default: 4242)
- `--web-ui`: Enables the monitoring web interface
- `--web-port`: Port for the web interface (default: 8080)
- `--web-bind`: Address to bind the web interface to (default: 0.0.0.0)
- `--tcp`: Uses TCP instead of UDP for communication
- `--http`: Uses HTTP for communication (useful for firewall-restricted environments)
- `--mdns`: Enables discovery/advertisement via mDNS
- `--multi`: Enables support for multiple simultaneous connections
- `--compression`: Compression algorithm (none, gzip, zlib, zstd)
- `--compress-level`: Compression level (1-9, default: 6)
- `--relay`: Address of the relay server (default: relay.apisbr.dev)
- `--session`: Session ID for relay connection

### Receiver Options
- `-b, --bind`: Address to bind to (default: 0.0.0.0)

### Sender Options
- `-H, --host`: Host to connect to (default: 127.0.0.1)

## Protocol

NP uses a simple protocol for authentication:

1. Client sends "ISNP"
2. Server responds with "OK" if it's a valid NP instance
3. Normal communication can begin after this authentication

This protocol ensures that NP only communicates with other NP instances, avoiding confusion with other network services.

## Current Limitations

- No native encryption (use SSH tunneling for secure communications)
- Buffer size limited to 4096 bytes per packet

## Future Features

- [x] TCP support for guaranteed delivery
- [x] HTTP support for firewall-restricted environments
- [x] Automatic discovery via mDNS
- [ ] End-to-end encryption
- [x] Relay mode for NAT traversal
- [x] Web interface for monitoring
- [x] Multiplex mode for multiple simultaneous connections
- [x] Real-time data compression
- [x] Support for multiple compression algorithms (gzip, zlib, zstd)
- [x] Configurable compression levels

## Troubleshooting

### Port in Use
If NP shows an error indicating that the port is already in use, try:
1. Check if another NP instance is running
2. Check if another application is using the port
3. Choose a different port with the `-p` parameter

### Connection Problems
If the sender cannot connect to the receiver:
1. Check if the receiver is running
2. Check if there are firewalls blocking the connection
3. Try using the `-b 0.0.0.0` option on the receiver to listen on all interfaces

### Web Interface Inaccessible
If the web interface is not accessible:
1. Check if you specified the `--web-ui` option
2. Make sure the web interface port is not blocked by a firewall
3. Check if the binding address allows access from other machines
4. Use `--web-bind 0.0.0.0` to allow access from any address

## Contributing

Contributions are welcome! To contribute:

1. Fork the repository
2. Create a branch for your feature (`git checkout -b feature/new-feature`)
3. Commit your changes (`git commit -am 'Add new feature'`)
4. Push to the branch (`git push origin feature/new-feature`)
5. Create a new Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Author

Leandro Ferreira (@lsferreira42)

## Acknowledgments

- Inspired by netcat and other classic network tools
- Special thanks to the Go community for feedback and contributions 