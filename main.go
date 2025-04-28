package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Metrics holds resource usage data
type Metrics struct {
	CPU  float64 `json:"cpu"`
	RAM  float64 `json:"ram"`
	Disk float64 `json:"disk"`
}

// Snap represents a connected client
type Snap struct {
	ID       string
	Conn     net.Conn
	LastSeen time.Time
	Alive    bool
	Metrics  Metrics
	LogCh    chan string
}

var (
	snaps    = make(map[string]*Snap)
	snapsMu  sync.Mutex
	upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
)

func main() {
	go startTCPServer()
	startWebServer()
}

func startTCPServer() {
	ln, err := net.Listen("tcp", "0.0.0.0:8081")
	if err != nil {
		log.Fatal("Error starting TCP server:", err)
	}
	defer ln.Close()
	log.Println("Master TCP server started on port 8081")
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("Failed to accept connection:", err)
			continue
		}
		go handleSnap(conn)
	}
}

func handleSnap(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)

	// First line is the Snap ID
	snapID, err := reader.ReadString('\n')
	if err != nil {
		log.Println("Failed to read Snap ID:", err)
		return
	}
	snapID = strings.TrimSpace(snapID)

	s := &Snap{
		ID:       snapID,
		Conn:     conn,
		LastSeen: time.Now(),
		Alive:    true,
		LogCh:    make(chan string, 100),
	}
	snapsMu.Lock()
	snaps[snapID] = s
	snapsMu.Unlock()
	log.Printf("Snap %s connected", snapID)

	// Heartbeat: ping every 10s
	go func(s *Snap) {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			s.Conn.Write([]byte("ping\n"))
			if time.Since(s.LastSeen) > 30*time.Second {
				s.Alive = false
			}
		}
	}(s)

	// Process incoming messages
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("Snap %s disconnected", snapID)
			snapsMu.Lock()
			delete(snaps, snapID)
			snapsMu.Unlock()
			return
		}
		line = strings.TrimSpace(line)
		switch {
		case line == "pong":
			s.LastSeen = time.Now()
			s.Alive = true

		case strings.HasPrefix(line, "metrics "):
			var m Metrics
			if err := json.Unmarshal([]byte(line[8:]), &m); err == nil {
				s.Metrics = m
			}

		case strings.HasPrefix(line, "log "):
			msg := line[4:]
			select {
			case s.LogCh <- msg:
			default:
			}
		}
	}
}

func startWebServer() {
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/shutdown", shutdownHandler)
	http.HandleFunc("/setbackground", setBackgroundHandler)
	http.HandleFunc("/ws/logs", logsWSHandler)

	addr := ":8080"
	url := "http://localhost" + addr
	log.Println("Dashboard available at", url)
	go openBrowser(url)

	log.Fatal(http.ListenAndServe(addr, nil))
}

// openBrowser opens the URL in the default browser
func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		return
	}
	_ = cmd.Start()
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	snapsMu.Lock()
	list := make([]*Snap, 0, len(snaps))
	for _, s := range snaps {
		list = append(list, s)
	}
	snapsMu.Unlock()

	const tmpl = `<!DOCTYPE html>
	<html lang="en">
	<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<title>Master Control Panel</title>
	<link href="https://cdn.jsdelivr.net/npm/bootstrap@5.3.0/dist/css/bootstrap.min.css" rel="stylesheet">
	</head>
	<body class="bg-light">
	<div class="container py-5">
	<h1 class="mb-4 text-center">Connected Snaps</h1>
	<div class="row">
	{{if .}}
	{{range .}}
	<div class="col-md-4">
	<div class="card shadow-sm mb-4">
	<div class="card-body">
	<h5 class="card-title">{{.ID}}</h5>
	<p>Last Seen: {{.LastSeen.Format "15:04:05 on 2006-01-02"}}</p>
	<p>Status: {{if .Alive}}<span class="text-success">Alive</span>{{else}}<span class="text-danger">Down</span>{{end}}</p>
	<p>CPU: {{printf "%.1f" .Metrics.CPU}}%</p>
	<p>RAM: {{printf "%.1f" .Metrics.RAM}}%</p>
	<p>Disk: {{printf "%.1f" .Metrics.Disk}}%</p>
	<form method="POST" action="/shutdown">
	<input type="hidden" name="id" value="{{.ID}}">
	<button class="btn btn-danger">Shutdown</button>
	</form>
	<hr>
	<form method="POST" action="/setbackground">
	<input type="hidden" name="id" value="{{.ID}}">
	<input type="text" name="bgurl" placeholder="Image URL" class="form-control mb-2">
	<button class="btn btn-primary">Set Background</button>
	</form>
	<button class="btn btn-secondary mt-2" onclick="openLogs('{{.ID}}')">View Logs</button>
	<pre id="logs-{{.ID}}" style="height:150px; overflow:auto; background:#f8f9fa;"></pre>
	</div>
	</div>
	</div>
	{{end}}
	{{else}}
	<p>No Snaps connected.</p>
	{{end}}
	</div>
	</div>
	<script>
		function openLogs(id) {
		var uri = "ws://" + location.host + "/ws/logs?id=" + id;
		var ws = new WebSocket(uri);
		var pre = document.getElementById("logs-" + id);
		ws.onmessage = function(evt) {
			pre.textContent += evt.data + "\n";
			pre.scrollTop = pre.scrollHeight;
		};
	}
	</script>
	</body>
	</html>`

	t := template.Must(template.New("webpage").Parse(tmpl))
	t.Execute(w, list)
}

func shutdownHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
		return
	}
	snapID := r.FormValue("id")
	snapsMu.Lock()
	s, ok := snaps[snapID]
	snapsMu.Unlock()
	if !ok {
		http.Error(w, "Snap not found", http.StatusNotFound)
		return
	}
	fmt.Fprintln(s.Conn, "shutdown")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func setBackgroundHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
		return
	}
	id := r.FormValue("id")
	url := r.FormValue("bgurl")
	snapsMu.Lock()
	s, ok := snaps[id]
	snapsMu.Unlock()
	if !ok {
		http.Error(w, "Snap not found", http.StatusNotFound)
		return
	}
	// send set background command
	fmt.Fprintf(s.Conn, "setbg %s\n", url)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func logsWSHandler(w http.ResponseWriter, r *http.Request) {
	snapID := r.URL.Query().Get("id")
	snapsMu.Lock()
	s, ok := snaps[snapID]
	snapsMu.Unlock()
	if !ok {
		http.Error(w, "Unknown snap", http.StatusNotFound)
		return
	}
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer ws.Close()
	for msg := range s.LogCh {
		if err := ws.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
			return
		}
	}
}
