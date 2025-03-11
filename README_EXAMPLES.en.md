# NP Usage Examples

This document contains detailed usage examples of NP (Network Pipe) for various scenarios. For installation instructions and general documentation, please refer to the [main README](README.en.md).

## Basic Usage

### Simple Example

```bash
# On the receiver machine (server)
np --receiver

# On the sender machine (client)
echo "Hello, world!" | np --sender -H 192.168.1.100
```

### Advanced Options

```bash
# Using long flags
np --sender --host 192.168.1.100 --port 5000

# With web interface enabled
np --sender -H 192.168.1.100 --web-ui

# Using TCP instead of UDP
np --sender --tcp

# Using HTTP (useful for firewall-restricted environments)
np --sender --http

# With mDNS discovery to automatically find servers
np --sender --mdns

# With support for multiple connections
np --sender --multi

# With data compression
np --sender --compression gzip --compress-level 9
```

## Working with Logs and Streams

### Docker Logs

Capturing logs from a Docker container and sending to another machine:

```bash
# On the receiver machine
np --receiver --tcp > docker_logs.txt

# On the sender machine
docker logs -f my_container | np --sender -H 192.168.1.100 --tcp
```

Capturing logs from multiple Docker containers:

```bash
# On the receiver machine
np --receiver --tcp --multi | grep ERROR > docker_errors.log

# On the sender machine (run for each container of interest)
docker logs -f container1 | np --sender -H 192.168.1.100 --tcp
docker logs -f container2 | np --sender -H 192.168.1.100 --tcp
```

### Kubernetes Logs

Monitoring logs from pods in a Kubernetes cluster:

```bash
# On the receiver machine
np --receiver --tcp --compression zstd > k8s_logs.txt

# On the sender machine
kubectl logs -f deployment/my-application | np --sender -H 192.168.1.100 --tcp --compression zstd
```

Monitoring logs from all pods with a certain label:

```bash
# On the receiver machine
np --receiver --tcp --multi > logs_by_environment.txt

# On the sender machine
kubectl logs -f -l app=backend | np --sender -H 192.168.1.100 --tcp
kubectl logs -f -l app=frontend | np --sender -H 192.168.1.100 --tcp
```

### Systemd Logs

Monitoring systemd services in real-time:

```bash
# On the receiver machine
np --receiver --tcp > systemd_logs.txt

# On the sender machine
journalctl -fu nginx | np --sender -H 192.168.1.100 --tcp
```

Monitoring multiple systemd services:

```bash
# On the receiver machine
np --receiver --tcp --multi | tee complete_logs.txt | grep ERROR > errors_only.txt

# On the sender machine
journalctl -fu nginx -fu postgresql -fu redis | np --sender -H 192.168.1.100 --tcp
```

### Log Files

Monitoring log files in real-time:

```bash
# On the receiver machine
np --receiver --tcp > application_logs.txt

# On the sender machine
tail -f /var/log/apache2/error.log | np --sender -H 192.168.1.100 --tcp
```

Monitoring multiple log files:

```bash
# On the receiver machine
np --receiver --tcp --compression zstd | tee -a all_logs.txt

# On the sender machine
tail -f /var/log/nginx/*.log | np --sender -H 192.168.1.100 --tcp --compression zstd
```

Concatenating large files and sending with compression:

```bash
# On the receiver machine
np --receiver --tcp --compression zstd > concatenated_logs.txt

# On the sender machine
cat file1.log file2.log | np --sender -H 192.168.1.100 --tcp --compression zstd
```

## Advanced Use Cases

### Automatic Discovery via mDNS

Use mDNS to automatically find NP servers on your local network:

```bash
# On the receiver machine (server)
np --receiver --mdns --tcp

# On the sender machine (client), without needing to specify the address
np --sender --mdns --tcp
```

### File Transfer with Compression

Send files with real-time compression for better performance:

```bash
# On the receiver machine
np --receiver --tcp --compression zstd > received_file.txt

# On the sender machine
cat large_file.txt | np --sender -H 192.168.1.100 --tcp --compression zstd
```

### Multiple Simultaneous Connections

Accept and manage multiple client connections at the same time:

```bash
# On the receiver machine (server)
np --receiver --tcp --multi --web-ui | ./process_data.sh

# On client machines
cat data1.txt | np --sender -H 192.168.1.100 --tcp
cat data2.txt | np --sender -H 192.168.1.100 --tcp
```

### Using HTTP for Firewall-Restricted Environments

Use HTTP mode when firewall or proxy blocks direct UDP/TCP connections:

```bash
# On the receiver machine (server)
np --receiver --http

# On the sender machine (client)
cat data.txt | np --sender -H 192.168.1.100 --http
```

### Using the Relay Server to Traverse NATs

Use the relay server to establish connections through NATs and firewalls:

```bash
# On the receiver machine (behind NAT)
np --receiver --relay relay.apisbr.dev --session my-session

# On the sender machine (behind another NAT)
cat data.txt | np --sender --relay relay.apisbr.dev --session my-session
```

The relay server at `relay.apisbr.dev` is available for all NP users.

### Combining All Features

For more complex applications, combine all features:

```bash
# Complete server with all features
np --receiver --tcp --mdns --multi --compression zstd --web-ui

# Client using automatic discovery and compression
np --sender --tcp --mdns --compression zstd --web-ui
```

## Real-Time Data Processing

### Processing Pipeline

Creating a processing pipeline between machines:

```bash
# On the receiver machine
np --receiver --tcp | grep "ERROR" | sort | uniq -c > error_report.txt

# On the sender machine
cat logs*.txt | np --sender -H 192.168.1.100 --tcp
```

### Distributed Processing

Splitting data for parallel processing on multiple machines:

```bash
# On the control machine
cat large_data.csv | 
  tee >(head -n 1000 | np --sender -H worker1.local --tcp) \
      >(tail -n +1001 | head -n 1000 | np --sender -H worker2.local --tcp) \
      >(tail -n +2001 | np --sender -H worker3.local --tcp) > /dev/null

# On worker machines
np --receiver --tcp | ./process_batch.sh
```

### Real-Time Production Monitoring

Monitoring production logs in real-time with alerts:

```bash
# On the monitoring machine
np --receiver --tcp --multi | 
  tee >(grep -i error | mail -s "Error Alert" admin@example.com) \
      complete_logs.txt

# On production machines
journalctl -fu application | np --sender -H monitor.local --tcp
``` 