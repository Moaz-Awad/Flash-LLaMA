package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"source-code-review/config"  // For ASCII art and colored logging
	"source-code-review/internal/markdown"
	"source-code-review/internal/scanner"
)

// Response structure for the LLaMA API
type LLaMAResponse struct {
	Response string `json:"response"`
}

func main() {
	// Show ASCII art at startup
	config.ShowASCII()

	// Handle graceful shutdown signals
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// Command-line arguments
	filePtr := flag.String("file", "", "File to scan for vulnerabilities")
	dirPtr := flag.String("dir", "", "Directory to scan for vulnerabilities")
	saveDir := flag.String("save", ".", "Directory to save the results (default is current directory)")
	llamaAPI := flag.String("llama-api", "http://localhost:8000/generate", "LLaMA API endpoint")
	flag.Parse()

	// Ensure save directory exists
	if _, err := os.Stat(*saveDir); os.IsNotExist(err) {
		config.Warn(fmt.Sprintf("Save directory does not exist: %s", *saveDir))
		os.Exit(1)
	}

	// Signal handling routine
	go func() {
		sig := <-sigs
		config.Info(fmt.Sprintf("Received signal: %s. Attempting graceful shutdown.", sig))
		os.Exit(0)
	}()

	// Collect files to scan
	var filesToScan []string
	if *filePtr != "" {
		filesToScan = append(filesToScan, *filePtr)
	} else if *dirPtr != "" {
		files, err := scanner.ScanDirectory(*dirPtr)
		if err != nil {
			config.Warn(fmt.Sprintf("Failed to scan directory: %v", err))
			os.Exit(1)
		}
		filesToScan = files
	} else {
		config.Warn("No file or directory specified")
		os.Exit(1)
	}

	// Process each file
	for _, file := range filesToScan {
		config.Info(fmt.Sprintf("Scanning %s using LLaMA", file))

		// Read the file content
		content, err := scanner.ScanFile(file)
		if err != nil {
			config.Warn(fmt.Sprintf("Failed to read file: %v", err))
			os.Exit(1)
		}

		// Get the AI response from LLaMA
		result, err := getLLaMAResponse(*llamaAPI, content)
		if err != nil {
			config.Warn(fmt.Sprintf("Failed to get AI response for %s: %v", file, err))
			os.Exit(1)
		}

		// Save the result to a markdown file
		config.Info(fmt.Sprintf("Saving results for %s", file))
		baseName := filepath.Base(file)
		resultFile := filepath.Join(*saveDir, baseName+".md")
		err = markdown.SaveMarkdown(resultFile, result)
		if err != nil {
			config.Warn(fmt.Sprintf("Failed to save markdown for %s: %v", file, err))
			os.Exit(1)
		}
	}

	config.Info("Scan complete.")
}

// getLLaMAResponse sends a prompt to the LLaMA API and retrieves the response
func getLLaMAResponse(apiEndpoint, prompt string) (string, error) {
	payload := map[string]string{"prompt": prompt}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to encode request payload: %v", err)
	}

	resp, err := http.Post(apiEndpoint, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return "", fmt.Errorf("API error: %s", string(body))
	}

	var llamaResp LLaMAResponse
	if err := json.NewDecoder(resp.Body).Decode(&llamaResp); err != nil {
		return "", fmt.Errorf("failed to decode API response: %v", err)
	}

	return llamaResp.Response, nil
}