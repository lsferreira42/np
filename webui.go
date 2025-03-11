package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sync"
	"time"
)

// WebUIConfig stores the web interface configuration
type WebUIConfig struct {
	Address string // IP address to bind the web UI to
	Port    int    // Port to serve the web UI on
	Enabled bool   // Whether the web UI is enabled
}

// Statistics maintains connection statistics and metrics for the application
type Statistics struct {
	BytesSent     uint64           // Total bytes sent across all connections
	BytesReceived uint64           // Total bytes received across all connections
	StartTime     time.Time        // Time when the application started
	Connections   []ConnectionInfo // Information about active connections
	mu            sync.RWMutex     // Mutex for thread-safe access
}

// ConnectionInfo stores detailed information about a single connection
type ConnectionInfo struct {
	RemoteAddr  string    `json:"remoteAddr"`  // Remote address (IP:port)
	LocalAddr   string    `json:"localAddr"`   // Local address (IP:port)
	ConnectedAt time.Time `json:"connectedAt"` // When the connection was established
	BytesIn     uint64    `json:"bytesIn"`     // Bytes received from this connection
	BytesOut    uint64    `json:"bytesOut"`    // Bytes sent to this connection
	LastActive  time.Time `json:"lastActive"`  // When the connection was last active
	IsActive    bool      `json:"isActive"`    // Whether the connection is currently active
}

// MessageBuffer stores recent messages for display in the web UI
type MessageBuffer struct {
	Messages []Message    // Circular buffer of messages
	Size     int          // Maximum number of messages to store
	mu       sync.RWMutex // Mutex for thread-safe access
}

// Message represents a single sent or received message
type Message struct {
	Content   string    `json:"content"`   // Content of the message (may be truncated)
	Direction string    `json:"direction"` // "in", "out", or "system"
	Timestamp time.Time `json:"timestamp"` // When the message was sent/received
	Size      int       `json:"size"`      // Original size in bytes
	From      string    `json:"from"`      // Source address
	To        string    `json:"to"`        // Destination address
}

var (
	stats         Statistics
	messageBuffer MessageBuffer
)

// StartWebUI initializes and starts the web user interface
// This runs in a separate goroutine so it doesn't block the main application
func StartWebUI(config *WebUIConfig, parentConfig *Config) {
	if !config.Enabled {
		return
	}

	// Initialize statistics tracking
	stats = Statistics{
		StartTime:   time.Now(),
		Connections: make([]ConnectionInfo, 0),
	}

	// Initialize message history buffer
	messageBuffer = MessageBuffer{
		Messages: make([]Message, 0),
		Size:     100, // Store the last 100 messages
	}

	// Setup HTTP routes
	http.HandleFunc("/", handleRoot)
	http.HandleFunc("/api/stats", handleStats)
	http.HandleFunc("/api/messages", handleMessages)
	http.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		handleConfig(w, r, parentConfig)
	})

	// Start the HTTP server in a separate goroutine
	addr := fmt.Sprintf("%s:%d", config.Address, config.Port)
	go func() {
		fmt.Printf("Web interface started at http://%s\n", addr)
		if err := http.ListenAndServe(addr, nil); err != nil {
			log.Fatalf("Error starting web server: %v", err)
		}
	}()
}

// handleRoot serves the main HTML page of the web interface
func handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	tmpl := template.Must(template.New("index").Parse(indexHTML))
	tmpl.Execute(w, nil)
}

// handleStats returns current statistics in JSON format
func handleStats(w http.ResponseWriter, r *http.Request) {
	stats.mu.RLock()
	defer stats.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"bytesSent":     stats.BytesSent,
		"bytesReceived": stats.BytesReceived,
		"uptime":        time.Since(stats.StartTime).String(),
		"connections":   stats.Connections,
	})
}

// handleMessages returns the message history buffer in JSON format
func handleMessages(w http.ResponseWriter, r *http.Request) {
	messageBuffer.mu.RLock()
	defer messageBuffer.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messageBuffer.Messages)
}

// handleConfig returns the current application configuration in JSON format
func handleConfig(w http.ResponseWriter, r *http.Request, config *Config) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"mode":     config.mode,
		"port":     config.port,
		"host":     config.host,
		"bindAddr": config.bindAddr,
	})
}

// RecordSentData updates statistics when data is sent
func RecordSentData(bytes uint64, to string) {
	stats.mu.Lock()
	defer stats.mu.Unlock()
	stats.BytesSent += bytes

	// Update the corresponding connection
	for i := range stats.Connections {
		if stats.Connections[i].RemoteAddr == to {
			stats.Connections[i].BytesOut += bytes
			stats.Connections[i].LastActive = time.Now()
			stats.Connections[i].IsActive = true
			break
		}
	}
}

// RecordReceivedData updates statistics when data is received
func RecordReceivedData(bytes uint64, from string) {
	stats.mu.Lock()
	defer stats.mu.Unlock()
	stats.BytesReceived += bytes

	// Check if the connection already exists
	var found bool
	for i := range stats.Connections {
		if stats.Connections[i].RemoteAddr == from {
			stats.Connections[i].BytesIn += bytes
			stats.Connections[i].LastActive = time.Now()
			stats.Connections[i].IsActive = true
			found = true
			break
		}
	}

	// If not found, add a new connection
	if !found {
		stats.Connections = append(stats.Connections, ConnectionInfo{
			RemoteAddr:  from,
			ConnectedAt: time.Now(),
			BytesIn:     bytes,
			LastActive:  time.Now(),
			IsActive:    true,
		})
	}
}

// RecordMessage adds a message to the history buffer
func RecordMessage(content string, direction string, size int, from, to string) {
	if len(content) > 100 {
		// Truncate very long messages for display
		content = content[:100] + "..."
	}

	msg := Message{
		Content:   content,
		Direction: direction,
		Timestamp: time.Now(),
		Size:      size,
		From:      from,
		To:        to,
	}

	messageBuffer.mu.Lock()
	defer messageBuffer.mu.Unlock()

	// Adds at the beginning so the most recent appear first
	messageBuffer.Messages = append([]Message{msg}, messageBuffer.Messages...)

	// Limits the buffer size
	if len(messageBuffer.Messages) > messageBuffer.Size {
		messageBuffer.Messages = messageBuffer.Messages[:messageBuffer.Size]
	}
}

// HTML template for the web interface with escaped $ characters
const indexHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>NP - Network Pipe Monitor</title>
    <style>
        body {
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            padding: 20px;
            background-color: #f5f5f5;
            color: #333;
            line-height: 1.6;
            max-width: 1200px;
            margin: 0 auto;
        }
        header {
            background-color: #2c3e50;
            color: white;
            padding: 1em;
            border-radius: 5px;
            margin-bottom: 20px;
            display: flex;
            justify-content: space-between;
            align-items: center;
        }
        h1, h2, h3 {
            margin-top: 0;
        }
        .card {
            background-color: white;
            border-radius: 5px;
            box-shadow: 0 2px 5px rgba(0,0,0,0.1);
            padding: 20px;
            margin-bottom: 20px;
        }
        .stats-container {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
            gap: 15px;
            margin-bottom: 20px;
        }
        .stat-card {
            background-color: #ecf0f1;
            border-radius: 5px;
            padding: 15px;
            text-align: center;
        }
        .stat-value {
            font-size: 1.8em;
            font-weight: bold;
            color: #2980b9;
        }
        .stat-label {
            color: #7f8c8d;
            font-size: 0.9em;
        }
        table {
            width: 100%;
            border-collapse: collapse;
            margin-top: 10px;
        }
        th, td {
            text-align: left;
            padding: 10px;
            border-bottom: 1px solid #ddd;
        }
        th {
            background-color: #f2f2f2;
        }
        tr:hover {
            background-color: #f5f5f5;
        }
        .message-item {
            border-left: 4px solid #3498db;
            padding: 10px;
            margin-bottom: 10px;
            background-color: #ecf0f1;
        }
        .message-item.outgoing {
            border-left-color: #e74c3c;
        }
        .message-content {
            font-family: monospace;
            white-space: pre-wrap;
            word-break: break-all;
            background-color: #f8f9fa;
            padding: 8px;
            border-radius: 3px;
        }
        .message-meta {
            display: flex;
            justify-content: space-between;
            color: #7f8c8d;
            font-size: 0.8em;
            margin-top: 5px;
        }
        .tabs {
            margin-bottom: 20px;
        }
        .tab-button {
            background-color: #f8f9fa;
            border: none;
            padding: 10px 20px;
            cursor: pointer;
            border-radius: 5px 5px 0 0;
            font-size: 1em;
        }
        .tab-button.active {
            background-color: white;
            border-bottom: 3px solid #3498db;
        }
        .tab-content {
            display: none;
        }
        .tab-content.active {
            display: block;
        }
        .status-indicator {
            width: 10px;
            height: 10px;
            border-radius: 50%;
            display: inline-block;
            margin-right: 5px;
        }
        .status-active {
            background-color: #2ecc71;
        }
        .status-inactive {
            background-color: #e74c3c;
        }
        .refresh-control {
            margin-bottom: 20px;
            display: flex;
            align-items: center;
        }
        .refresh-button {
            background-color: #3498db;
            color: white;
            border: none;
            padding: 8px 15px;
            border-radius: 5px;
            cursor: pointer;
            margin-right: 10px;
        }
        .refresh-button:hover {
            background-color: #2980b9;
        }
        .footer {
            text-align: center;
            margin-top: 30px;
            padding-top: 10px;
            border-top: 1px solid #ddd;
            color: #7f8c8d;
            font-size: 0.9em;
        }
        @media (max-width: 768px) {
            .stats-container {
                grid-template-columns: 1fr;
            }
        }
    </style>
</head>
<body>
    <header>
        <h1>NP - Network Pipe Monitor</h1>
        <div>
            <span id="mode-badge"></span>
        </div>
    </header>

    <div class="tabs">
        <button class="tab-button active" data-tab="dashboard">Dashboard</button>
        <button class="tab-button" data-tab="connections">Connections</button>
        <button class="tab-button" data-tab="messages">Messages</button>
        <button class="tab-button" data-tab="configuration">Configuration</button>
    </div>

    <div class="refresh-control">
        <button class="refresh-button" id="refresh-button">Refresh Data</button>
        <label for="auto-refresh">
            <input type="checkbox" id="auto-refresh" checked> Auto-refresh (5s)
        </label>
    </div>

    <div class="tab-content active" id="dashboard-tab">
        <div class="stats-container">
            <div class="stat-card">
                <div class="stat-value" id="bytes-sent">0</div>
                <div class="stat-label">Bytes Sent</div>
            </div>
            <div class="stat-card">
                <div class="stat-value" id="bytes-received">0</div>
                <div class="stat-label">Bytes Received</div>
            </div>
            <div class="stat-card">
                <div class="stat-value" id="active-connections">0</div>
                <div class="stat-label">Active Connections</div>
            </div>
            <div class="stat-card">
                <div class="stat-value" id="uptime">0</div>
                <div class="stat-label">Uptime</div>
            </div>
        </div>

        <div class="card">
            <h2>Latest Activity</h2>
            <div id="activity-feed"></div>
        </div>
    </div>

    <div class="tab-content" id="connections-tab">
        <div class="card">
            <h2>Connection Details</h2>
            <table id="connections-table">
                <thead>
                    <tr>
                        <th>Status</th>
                        <th>Remote Address</th>
                        <th>Connected At</th>
                        <th>Last Active</th>
                        <th>Bytes In</th>
                        <th>Bytes Out</th>
                    </tr>
                </thead>
                <tbody id="connections-body">
                    <!-- Connections will be listed here -->
                </tbody>
            </table>
        </div>
    </div>

    <div class="tab-content" id="messages-tab">
        <div class="card">
            <h2>Message Log</h2>
            <div id="message-log">
                <!-- Messages will be listed here -->
            </div>
        </div>
    </div>

    <div class="tab-content" id="configuration-tab">
        <div class="card">
            <h2>NP Configuration</h2>
            <table>
                <tr>
                    <td><strong>Mode:</strong></td>
                    <td id="config-mode"></td>
                </tr>
                <tr>
                    <td><strong>Port:</strong></td>
                    <td id="config-port"></td>
                </tr>
                <tr>
                    <td><strong>Host:</strong></td>
                    <td id="config-host"></td>
                </tr>
                <tr>
                    <td><strong>Bind Address:</strong></td>
                    <td id="config-bind"></td>
                </tr>
            </table>
        </div>
    </div>

    <div class="footer">
        <p>NP - Network Pipe | GitHub: <a href="https://github.com/lsferreira42/np" target="_blank">lsferreira42/np</a></p>
    </div>

    <script>
        document.addEventListener('DOMContentLoaded', function() {
            // Initialize tabs
            const tabs = document.querySelectorAll('.tab-button');
            tabs.forEach(tab => {
                tab.addEventListener('click', function() {
                    // Remove active from all tabs
                    tabs.forEach(t => t.classList.remove('active'));
                    document.querySelectorAll('.tab-content').forEach(
                        content => content.classList.remove('active')
                    );
                    
                    // Activate the clicked tab
                    this.classList.add('active');
                    document.getElementById(this.dataset.tab + '-tab').classList.add('active');
                });
            });

            // Function to format bytes
            function formatBytes(bytes) {
                if (bytes === 0) return '0 Bytes';
                const k = 1024;
                const sizes = ['Bytes', 'KB', 'MB', 'GB'];
                const i = Math.floor(Math.log(bytes) / Math.log(k));
                return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
            }

            // Function to format date
            function formatDate(dateString) {
                const date = new Date(dateString);
                return date.toLocaleString();
            }

            // Function to calculate time elapsed
            function timeAgo(dateString) {
                const date = new Date(dateString);
                const seconds = Math.floor((new Date() - date) / 1000);
                
                let interval = seconds / 31536000;
                if (interval > 1) return Math.floor(interval) + " years ago";
                
                interval = seconds / 2592000;
                if (interval > 1) return Math.floor(interval) + " months ago";
                
                interval = seconds / 86400;
                if (interval > 1) return Math.floor(interval) + " days ago";
                
                interval = seconds / 3600;
                if (interval > 1) return Math.floor(interval) + " hours ago";
                
                interval = seconds / 60;
                if (interval > 1) return Math.floor(interval) + " minutes ago";
                
                return Math.floor(seconds) + " seconds ago";
            }

            // Functions to load data
            async function fetchStats() {
                try {
                    const response = await fetch('/api/stats');
                    return await response.json();
                } catch (error) {
                    console.error('Error fetching stats:', error);
                    return null;
                }
            }

            async function fetchMessages() {
                try {
                    const response = await fetch('/api/messages');
                    return await response.json();
                } catch (error) {
                    console.error('Error fetching messages:', error);
                    return [];
                }
            }

            async function fetchConfig() {
                try {
                    const response = await fetch('/api/config');
                    return await response.json();
                } catch (error) {
                    console.error('Error fetching config:', error);
                    return {};
                }
            }

            // Function to update the dashboard
            async function updateDashboard() {
                const stats = await fetchStats();
                if (!stats) return;

                document.getElementById('bytes-sent').textContent = formatBytes(stats.bytesSent);
                document.getElementById('bytes-received').textContent = formatBytes(stats.bytesReceived);
                
                const activeConnections = stats.connections.filter(c => c.isActive).length;
                document.getElementById('active-connections').textContent = activeConnections;
                document.getElementById('uptime').textContent = stats.uptime;

                // Update the connections table
                const connectionsBody = document.getElementById('connections-body');
                connectionsBody.innerHTML = '';
                
                stats.connections.forEach(conn => {
                    const row = document.createElement('tr');
                    // JavaScript string template - We use normal strings with concatenation here to avoid issues with the Go compiler
                    row.innerHTML = '<td><span class="status-indicator ' + (conn.isActive ? 'status-active' : 'status-inactive') + '"></span> ' + (conn.isActive ? 'Active' : 'Inactive') + '</td>' +
                        '<td>' + conn.remoteAddr + '</td>' +
                        '<td>' + formatDate(conn.connectedAt) + '</td>' +
                        '<td>' + formatDate(conn.lastActive) + ' (' + timeAgo(conn.lastActive) + ')</td>' +
                        '<td>' + formatBytes(conn.bytesIn) + '</td>' +
                        '<td>' + formatBytes(conn.bytesOut) + '</td>';
                    connectionsBody.appendChild(row);
                });

                // Update the activity feed
                const messages = await fetchMessages();
                const activityFeed = document.getElementById('activity-feed');
                activityFeed.innerHTML = '';
                
                const recentMessages = messages.slice(0, 5);
                recentMessages.forEach(msg => {
                    const div = document.createElement('div');
                    div.className = 'message-item ' + (msg.direction === 'out' ? 'outgoing' : '');
                    // JavaScript string template - We use normal strings with concatenation here
                    div.innerHTML = '<div class="message-content">' + msg.content + '</div>' +
                        '<div class="message-meta">' +
                            '<span>' + (msg.direction === 'out' ? 'Sent to' : 'Received from') + ' ' + (msg.direction === 'out' ? msg.to : msg.from) + '</span>' +
                            '<span>' + formatBytes(msg.size) + ' | ' + timeAgo(msg.timestamp) + '</span>' +
                        '</div>';
                    activityFeed.appendChild(div);
                });
            }

            // Function to update the messages tab
            async function updateMessagesTab() {
                const messages = await fetchMessages();
                const messageLog = document.getElementById('message-log');
                messageLog.innerHTML = '';
                
                messages.forEach(msg => {
                    const div = document.createElement('div');
                    div.className = 'message-item ' + (msg.direction === 'out' ? 'outgoing' : '');
                    // JavaScript string template - We use normal strings here
                    div.innerHTML = '<div class="message-content">' + msg.content + '</div>' +
                        '<div class="message-meta">' +
                            '<span>' + (msg.direction === 'out' ? 'Sent to' : 'Received from') + ' ' + (msg.direction === 'out' ? msg.to : msg.from) + '</span>' +
                            '<span>' + formatBytes(msg.size) + ' | ' + formatDate(msg.timestamp) + '</span>' +
                        '</div>';
                    messageLog.appendChild(div);
                });
            }

            // Function to update the configuration tab
            async function updateConfigTab() {
                const config = await fetchConfig();
                
                document.getElementById('config-mode').textContent = config.mode;
                document.getElementById('config-port').textContent = config.port;
                document.getElementById('config-host').textContent = config.host || 'N/A';
                document.getElementById('config-bind').textContent = config.bindAddr || 'N/A';
                
                // Update the mode badge in the header
                const modeBadge = document.getElementById('mode-badge');
                modeBadge.textContent = config.mode === 'receiver' ? 'RECEIVER MODE' : 'SENDER MODE';
                modeBadge.style.backgroundColor = config.mode === 'receiver' ? '#27ae60' : '#e67e22';
                modeBadge.style.padding = '5px 10px';
                modeBadge.style.borderRadius = '3px';
                modeBadge.style.color = 'white';
                modeBadge.style.fontWeight = 'bold';
            }

            // Function to update all data
            async function updateAllData() {
                await updateDashboard();
                await updateMessagesTab();
                await updateConfigTab();
            }

            // Refresh configuration
            let refreshInterval;
            
            function setupAutoRefresh() {
                const autoRefreshCheckbox = document.getElementById('auto-refresh');
                
                if (autoRefreshCheckbox.checked) {
                    refreshInterval = setInterval(updateAllData, 5000);
                } else {
                    clearInterval(refreshInterval);
                }
            }
            
            document.getElementById('auto-refresh').addEventListener('change', setupAutoRefresh);
            document.getElementById('refresh-button').addEventListener('click', updateAllData);
            
            // Load initial data
            updateAllData();
            setupAutoRefresh();
        });
    </script>
</body>
</html>`
