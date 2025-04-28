# SnapMaster Control Panel

A lightweight Go-based master–snap system for remote monitoring and control.

## Overview

This repository contains two Go applications:

- **Master (Server)**: Listens for Snap clients, displays a real-time dashboard (HTTP + WebSockets), sends commands (shutdown, set background), and collects metrics (CPU, RAM, Disk).
- **Snap (Client)**: Connects to the Master, reports resource metrics periodically, handles commands (`ping`, `shutdown`, `setbg <url|path>`), and applies system-level actions.

## Features

- **Heartbeat & Status**: `ping`/`pong` mechanism to track connectivity and `LastSeen` timestamp.
- **Resource Metrics**: Periodic reporting of CPU, RAM, and Disk usage.
- **Dashboard**: Web interface showing connected Snaps, metrics, and live logs via WebSocket.
- **Commands**:
  - `shutdown`: Initiates OS shutdown on the Snap.
  - `setbg <url|path>`: Changes the Snap’s desktop background to a remote image URL or local file.
- **Auto-launch**: Master automatically opens the dashboard in the default browser.

## Prerequisites

- Go 1.18 or later installed: https://golang.org/dl/
- On **Linux**: `gsettings` (for GNOME background change).
- On **Windows**: PowerShell (built-in).
- On **macOS**: `osascript` (built-in).

## Setup

1. **Clone the repository**
   ```bash
   git clone https://github.com/yourusername/yourrepo.git
   cd yourrepo
   ```

2. **Initialize Go Modules and install dependencies**
   ```bash
   go mod init github.com/yourusername/yourrepo
   go get github.com/gorilla/websocket@latest \
          github.com/shirou/gopsutil/cpu@latest \
          github.com/shirou/gopsutil/mem@latest \
          github.com/shirou/gopsutil/disk@latest
   go mod tidy
   ```

## Running the Master (Server)

1. **Build** (optional)
   ```bash
   go build -o master master_control_panel.go
   ```

2. **Run**
   ```bash
   # If built:
   ./master
   # Or directly:
   go run master_control_panel.go
   ```

3. **Usage**
   - The Master starts listening on TCP port `8081` and HTTP port `8080` by default.
   - On startup, it will automatically open `http://localhost:8080` in your default browser.
   - Use the web UI to view connected Snaps, metrics, send commands, and live logs.

## Running the Snap (Client)

1. **Build** (optional)
   ```bash
   go build -o snap snap_client.go
   ```

2. **Run**
   ```bash
   go run snap_client.go
   ```

3. **Prompts**
   - **Master IP**: Enter the IP or hostname of the Master (e.g., `192.168.1.100`).
   - **Master Port**: Enter `8081` (must match Master’s TCP port).
   - **Snap ID**: Provide a unique identifier for this client (e.g., `snap1`).

4. **Behavior**
   - Connects to Master and reports metrics every 5 seconds.
   - Listens for commands (`ping`, `shutdown`, `setbg <url|path>`).
   - Applies system-level actions as instructed.

## Directory Structure

```
├── master_control_panel.go   # Master server code
├── snap_client.go            # Snap client code
├── go.mod                    # Go module definition
├── go.sum                    # Dependency checksums
└── README.md                 # This file
```

## Contributing

Feel free to open issues or submit pull requests for new features, bug fixes, or improvements.

## License

This project is licensed under the MIT License. See [LICENSE](LICENSE) for details.

