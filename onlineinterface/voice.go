package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func processAudioHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	file, _, err := r.FormFile("audio")
	if err != nil {
		http.Error(w, "Failed to read audio file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Create a temporary directory in the onlineinterface folder
	tempDirName := fmt.Sprintf("temp-audio-%d", time.Now().UnixNano())
	tempDir := filepath.Join("onlineinterface", tempDirName)

	err = os.MkdirAll(tempDir, 0755)
	if err != nil {
		http.Error(w, "Failed to create temp directory", http.StatusInternalServerError)
		return
	}

	// Ensure cleanup happens at the end
	defer func() {
		os.RemoveAll(tempDir)
		fmt.Printf("Cleaned up temp directory: %s\n", tempDir)
	}()

	// Create a unique filename in the temp directory
	filename := fmt.Sprintf("audio-%d.wav", time.Now().UnixNano())
	audioPath := filepath.Join(tempDir, filename)

	audioFile, err := os.Create(audioPath)
	if err != nil {
		http.Error(w, "Failed to create audio file", http.StatusInternalServerError)
		return
	}
	defer audioFile.Close()

	_, err = io.Copy(audioFile, file)
	if err != nil {
		http.Error(w, "Failed to save audio file", http.StatusInternalServerError)
		return
	}

	// Process the audio file
	cmd := exec.Command("go", "run", "justaserver.go", audioPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Error running justaserver.go: %v\n", err)
		http.Error(w, "Failed to process audio", http.StatusInternalServerError)
		return
	}

	result := strings.TrimSpace(string(output))
	fmt.Printf("justaserver.go output: %s\n", result)

	// Return the text result
	fmt.Fprint(w, result)
}

func pingHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	fmt.Println("LAUNCH detected! Ping received.")
	fmt.Fprint(w, "Launch detected - Ping sent!")
}

func serveHTML(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "onlineinterface/voice.html")
}

func main() {
	http.HandleFunc("/", serveHTML)
	http.HandleFunc("/process-audio", processAudioHandler)
	http.HandleFunc("/ping", pingHandler)

	fmt.Println("Server is running on:")
	fmt.Println("- HTTP:  http://localhost:8080")
	fmt.Println("- HTTPS: https://localhost:8443")
	fmt.Println("")
	fmt.Println("Network access available at:")
	fmt.Println("- Find your IP address with: ip addr show | grep 'inet ' | grep -v '127.0.0.1'")
	fmt.Println("- Then access from other devices at: https://YOUR_IP:8443")
	fmt.Println("- Note: You'll need to accept the self-signed certificate warning")

	go func() {
		fmt.Println("Starting HTTPS server on port 8443...")
		if err := http.ListenAndServeTLS("0.0.0.0:8443", "cert.pem", "key.pem", nil); err != nil {
			fmt.Printf("HTTPS server error: %v\n", err)
		}
	}()

	httpMux := http.NewServeMux()

	httpMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Host == "localhost:8080" || r.Host == "127.0.0.1:8080" {
			serveHTML(w, r)
		} else {
			httpsURL := "https://" + r.Host[:len(r.Host)-4] + "8443" + r.RequestURI
			http.Redirect(w, r, httpsURL, http.StatusMovedPermanently)
		}
	})

	httpMux.HandleFunc("/process-audio", func(w http.ResponseWriter, r *http.Request) {
		if r.Host == "localhost:8080" || r.Host == "127.0.0.1:8080" {
			processAudioHandler(w, r)
		} else {
			httpsURL := "https://" + r.Host[:len(r.Host)-4] + "8443" + r.RequestURI
			http.Redirect(w, r, httpsURL, http.StatusMovedPermanently)
		}
	})

	httpMux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		if r.Host == "localhost:8080" || r.Host == "127.0.0.1:8080" {
			pingHandler(w, r)
		} else {
			httpsURL := "https://" + r.Host[:len(r.Host)-4] + "8443" + r.RequestURI
			http.Redirect(w, r, httpsURL, http.StatusMovedPermanently)
		}
	})

	fmt.Println("Starting HTTP server on port 8080...")
	http.ListenAndServe("0.0.0.0:8080", httpMux)
}
