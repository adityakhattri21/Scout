package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/adityakhattri21/scout/models"
	"github.com/akamensky/argparse"
	"github.com/fsnotify/fsnotify"
	"gopkg.in/yaml.v3"
)

func GetArgs() (models.Config, error) {
	parser := argparse.NewParser("Scout", "A Hot Reloader in Go for Go ;)")
	cwd, err := os.Getwd()
	if err != nil {
		return models.Config{}, err
	}

	configFile := parser.File("c", "config", os.O_RDONLY, 0644, &argparse.Options{
		Help:     "Path to the configuration file",
		Default:  cwd + "/scout_config.yaml",
		Required: false,
	})

	err = parser.Parse(os.Args)
	if err != nil {
		return models.Config{}, fmt.Errorf("error parsing args: %v", err)
	}

	return models.Config{File_Location: configFile.Name()}, nil
}

func loadConfig(config *models.Config) error {
	data, err := os.ReadFile(config.File_Location)
	if err != nil {
		return fmt.Errorf("error reading config: %v", err)
	}

	if err := yaml.Unmarshal(data, config); err != nil {
		return fmt.Errorf("error parsing config: %v", err)
	}

	fmt.Printf("Scout initialized with config from: %s\n", config.File_Location)
	return nil
}

func killProcess(proc *os.Process, config models.Config) error {
	if proc == nil {
		return nil
	}

	// Get the process group ID
	pgid, err := syscall.Getpgid(proc.Pid)
	if err != nil {
		log.Printf("Error getting process group ID: %v", err)
		// Fall back to killing just the main process if we can't get the group
		return killSingleProcess(proc)
	}

	// Kill the entire process group
	if err := syscall.Kill(-pgid, syscall.SIGTERM); err != nil {
		log.Printf("Error sending SIGTERM to process group: %v", err)
		// Try SIGKILL if SIGTERM fails
		if err := syscall.Kill(-pgid, syscall.SIGKILL); err != nil {
			log.Printf("Error sending SIGKILL to process group: %v", err)
			return err
		}
	}

	// Wait for the main process to exit
	_, err = proc.Wait()
	if err != nil {
		log.Printf("Error waiting for process to terminate: %v", err)
	}

	// Additional cleanup: check if port is still in use
	timeout := time.After(2 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout waiting for port to be released")
		case <-ticker.C:
			port_int, _ := strconv.Atoi((config.Port)) //Ignoring err for now
			if !isPortInUse(port_int) {                // You might want to make this port configurable
				return nil
			}
		}
	}
}

func killSingleProcess(proc *os.Process) error {
	if err := proc.Signal(syscall.SIGTERM); err != nil {
		log.Printf("Error sending SIGTERM: %v", err)
		if err := proc.Kill(); err != nil {
			log.Printf("Error sending SIGKILL: %v", err)
			return err
		}
	}
	_, err := proc.Wait()
	return err
}

func isPortInUse(port int) bool {
	addr := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return true
	}
	listener.Close()
	return false
}

func startProcess(config models.Config) (*os.Process, error) {
	cmd := exec.Command("sh", "-c", config.Start_Command)

	// Create a new process group
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	maxRetries := 3
	var lastErr error

	for i := 0; i < maxRetries; i++ {
		// Ensure port is free before starting
		if isPortInUse(8080) { // Make port configurable
			time.Sleep(500 * time.Millisecond)
			continue
		}

		if err := cmd.Start(); err != nil {
			lastErr = fmt.Errorf("failed to start command (attempt %d/%d): %v", i+1, maxRetries, err)
			time.Sleep(500 * time.Millisecond)
			continue
		}

		log.Printf("Started process with PID: %d", cmd.Process.Pid)
		return cmd.Process, nil
	}

	return nil, lastErr
}

func watchFileChange(ctx context.Context, config models.Config, processChan chan *os.Process) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalf("Error creating watcher: %v", err)
	}
	defer watcher.Close()

	if err := watcher.Add(config.Work_dir); err != nil {
		log.Fatalf("Error watching directory: %v", err)
	}

	entries, _ := os.ReadDir(config.Work_dir)
	for _, entry := range entries {
		if entry.IsDir() {
			dirPath := filepath.Join(config.Work_dir, entry.Name())
			_ = watcher.Add(dirPath)
		}
	}

	// Debounce map
	modifiedFiles := make(map[string]time.Time)
	interval := 2 * time.Second

	fmt.Println("Scout is watching for changes...")

	for {
		select {
		case event := <-watcher.Events:
			if event.Has(fsnotify.Write) {
				incMatched, _ := filepath.Match(config.File_Patterns, filepath.Base(event.Name))
				excMatched, _ := filepath.Match(config.Exclude_Patterns, filepath.Base(event.Name))

				if incMatched && !excMatched {
					now := time.Now()
					if lastModTime, exists := modifiedFiles[event.Name]; exists && now.Sub(lastModTime) < interval {
						continue
					}

					modifiedFiles[event.Name] = now
					fmt.Printf("\nRestarting due to changes in: %s\n", filepath.Base(event.Name))

					if proc := <-processChan; proc != nil {
						if err := killProcess(proc, config); err != nil {
							log.Printf("Error stopping process: %v", err)
						}
					}

					newProc, err := startProcess(config)
					if err != nil {
						log.Printf("Error starting process: %v", err)
					} else {
						processChan <- newProc
					}
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

func main() {
	args, err := GetArgs()
	if err != nil {
		log.Fatalf("Initialization error: %v", err)
	}

	if err := loadConfig(&args); err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	processChan := make(chan *os.Process, 1)

	go watchFileChange(ctx, args, processChan)

	initialProc, err := startProcess(args)
	if err != nil {
		log.Fatalf("Failed to start process: %v", err)
	}
	processChan <- initialProc

	<-signalChan
	fmt.Println("\nShutting down...")

	if proc := <-processChan; proc != nil {
		_ = killProcess(proc, args)
	}
}
