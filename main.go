package main

import (
	"fmt"
	"log"
	"os/exec"
	"runtime"
	"time"
	
	"nextcloud-perf/internal/ui"
)

func openBrowser(url string) {
	var err error
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		log.Printf("Failed to open browser: %v", err)
	}
}

func main() {
	fmt.Println("Starting Nextcloud Performance Tool...")
	
	// Start UI Server
	server := ui.NewServer(3000)
	
	// Open Browser in a goroutine (wait a bit for server to start)
	go func() {
		time.Sleep(500 * time.Millisecond)
		openBrowser("http://localhost:3000")
	}()
	
	server.Listen()
}
