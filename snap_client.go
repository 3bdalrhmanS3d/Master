// package main

// import (
// 	"bufio"
// 	"encoding/json"
// 	"fmt"
// 	"io"
// 	"log"
// 	"net"
// 	"net/http"
// 	"os"
// 	"os/exec"
// 	"path/filepath"
// 	"runtime"
// 	"strings"
// 	"time"

// 	"github.com/shirou/gopsutil/cpu"
// 	"github.com/shirou/gopsutil/disk"
// 	"github.com/shirou/gopsutil/mem"
// )

// var (
// 	masterAddress string
// 	snapID        string
// )

// // MetricsSnap holds resource usage data reported to master
// //
// type MetricsSnap struct {
// 	CPU  float64 `json:"cpu"`
// 	RAM  float64 `json:"ram"`
// 	Disk float64 `json:"disk"`
// }

// func main() {
// 	reader := bufio.NewReader(os.Stdin)

// 	fmt.Print("Enter master IP (e.g. 192.168.1.100): ")
// 	ip, err := reader.ReadString('\n')
// 	if err != nil {
// 		log.Fatalf("Failed to read master IP: %v", err)
// 	}
// 	ip = strings.TrimSpace(ip)

// 	fmt.Print("Enter master port (e.g. 8081): ")
// 	port, err := reader.ReadString('\n')
// 	if err != nil {
// 		log.Fatalf("Failed to read master port: %v", err)
// 	}
// 	port = strings.TrimSpace(port)

// 	masterAddress = fmt.Sprintf("%s:%s", ip, port)

// 	fmt.Print("Enter Snap ID: ")
// 	id, err := reader.ReadString('\n')
// 	if err != nil {
// 		log.Fatalf("Failed to read Snap ID: %v", err)
// 	}
// 	snapID = strings.TrimSpace(id)

// 	for {
// 		if err := connectToMaster(); err != nil {
// 			log.Printf("Connection error: %v. Retrying in 5s...", err)
// 			time.Sleep(5 * time.Second)
// 		}
// 	}
// }

// func connectToMaster() error {
// 	conn, err := net.Dial("tcp", masterAddress)
// 	if err != nil {
// 		return err
// 	}
// 	defer conn.Close()

// 	log.Printf("Connected to master at %s", masterAddress)
// 	fmt.Fprintln(conn, snapID)

// 	go metricsLoop(conn)

// 	reader := bufio.NewReader(conn)
// 	for {
// 		line, err := reader.ReadString('\n')
// 		if err != nil {
// 			return err
// 		}
// 		cmd := strings.TrimSpace(line)
// 		log.Printf("Received command: %s", cmd)

// 		switch {
// 		case cmd == "ping":
// 			fmt.Fprintln(conn, "pong")

// 		case cmd == "shutdown":
// 			log.Println("Shutdown command received")
// 			shutdownSystem()
// 			return nil

// 		case strings.HasPrefix(cmd, "setbg "):
// 			arg := strings.TrimSpace(cmd[len("setbg "):])
// 			log.Printf("Setting background: %s", arg)
// 			if err := setBackground(arg); err != nil {
// 				log.Printf("Error setting background: %v", err)
// 			}

// 		default:
// 			log.Printf("Unknown command: %s", cmd)
// 		}
// 	}
// }

// func metricsLoop(conn net.Conn) {
// 	t := time.NewTicker(5 * time.Second)
// 	defer t.Stop()

// 	for range t.C {
// 		cpuPerc, err := cpu.Percent(0, false)
// 		if err != nil || len(cpuPerc) == 0 {
// 			continue
// 		}

// 		vm, err := mem.VirtualMemory()
// 		if err != nil {
// 			continue
// 		}

// 		du, err := disk.Usage("/")
// 		if err != nil {
// 			continue
// 		}

// 		m := MetricsSnap{CPU: cpuPerc[0], RAM: vm.UsedPercent, Disk: du.UsedPercent}
// 		b, err := json.Marshal(m)
// 		if err != nil {
// 			continue
// 		}

// 		fmt.Fprintf(conn, "metrics %s\n", b)
// 	}
// }

// func shutdownSystem() {
// 	var cmd *exec.Cmd
// 	if runtime.GOOS == "windows" {
// 		cmd = exec.Command("shutdown", "/s", "/t", "0")
// 	} else {
// 		cmd = exec.Command("sudo", "shutdown", "now")
// 	}
// 	if err := cmd.Run(); err != nil {
// 		log.Printf("Shutdown error: %v", err)
// 	}
// }

// // setBackground handles both URLs and local file paths
// func setBackground(src string) error {
// 	var bgPath string

// 	// local file?
// 	if abs, err := filepath.Abs(src); err == nil {
// 		if _, err := os.Stat(abs); err == nil {
// 			bgPath = abs
// 		}
// 	}

// 	// remote URL?
// 	if bgPath == "" && (strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://")) {
// 		resp, err := http.Get(src)
// 		if err != nil {
// 			return err
// 		}
// 		defer resp.Body.Close()

// 		ext := filepath.Ext(src)
// 		if ext == "" {
// 			ext = ".jpg"
// 		}
// 		tmp := filepath.Join(os.TempDir(), "snapbg"+ext)
// 		f, err := os.Create(tmp)
// 		if err != nil {
// 			return err
// 		}
// 		defer f.Close()
// 		if _, err := io.Copy(f, resp.Body); err != nil {
// 			return err
// 		}
// 		bgPath = tmp
// 	}

// 	if bgPath == "" {
// 		return fmt.Errorf("invalid path or URL: %s", src)
// 	}

// 	switch runtime.GOOS {
// 	case "linux":
// 		exec.Command("gsettings", "set", "org.gnome.desktop.background", "picture-uri", "file://"+bgPath).Run()
// 	case "windows":
// 		exec.Command("powershell", "-Command",
// 			fmt.Sprintf("Set-ItemProperty -Path 'HKCU:\\Control Panel\\Desktop' -Name Wallpaper -Value '%s'", bgPath),
// 		).Run()
// 		exec.Command("RUNDLL32.EXE", "user32.dll,UpdatePerUserSystemParameters").Run()
// 	case "darwin":
// 		exec.Command("osascript", "-e",
// 			fmt.Sprintf("tell application \"Finder\" to set desktop picture to POSIX file \"%s\"", bgPath),
// 		).Run()
// 	}

// 	return nil
// }
