// package main

// import (
// 	"context"
// 	"fmt"
// 	"log"
// 	"os"
// 	"os/exec"
// 	"os/signal"
// 	"path/filepath"
// 	"sync"
// 	"syscall"
// 	"time"

// 	"github.com/adityakhattri21/scout/models"
// 	"github.com/akamensky/argparse"
// 	"github.com/fsnotify/fsnotify"
// 	"gopkg.in/yaml.v3"
// )

// func GetArgs() (models.Config, error) {

// 	parser := argparse.NewParser("Scout", "A Hot Reloader in Go for Go ;)")

// 	cwd, err := os.Getwd()
// 	if err != nil {
// 		return models.Config{}, err
// 	}

// 	configFile := parser.File("c", "config", os.O_RDONLY, 0644, &argparse.Options{
// 		Help:     "Path to the configuration file",
// 		Default:  cwd + "/scout_config.yaml",
// 		Required: false,
// 	})

// 	err = parser.Parse(os.Args)
// 	if err != nil {
// 		fmt.Printf("Error occurred: %s\n", err)
// 		return models.Config{}, err
// 	}

// 	config := models.Config{File_Location: configFile.Name()}

// 	return config, nil
// }

// func loadConfig(config *models.Config) (string, error) {
// 	data, err := os.ReadFile(config.File_Location)

// 	if err != nil {
// 		return "", err
// 	}

// 	err = yaml.Unmarshal(data, config)

// 	if err != nil {
// 		return "", err
// 	}
// 	return "", nil
// }

// func clearTerminal() {
// 	fmt.Print("\033[H\033[2J")
// }

// func killProcess(proc *os.Process) error {
// 	if proc == nil {
// 		return nil
// 	}

// 	// Check if process is still alive
// 	if err := proc.Signal(syscall.Signal(0)); err != nil {
// 		log.Println("Process already finished. Skipping termination.")
// 		return nil
// 	}

// 	log.Println("Killing existing process...")
// 	err := proc.Kill()
// 	if err != nil {
// 		log.Printf("Error killing process: %v", err)
// 		return err
// 	}
// 	proc.Release()
// 	time.Sleep(2 * time.Second) // Wait for port to be released
// 	return nil
// }

// func startProcess(ctx context.Context, config models.Config, wg *sync.WaitGroup, processChan chan *os.Process) {
// 	defer wg.Done()
// 	clearTerminal()
// 	time.Sleep(2 * time.Second)
// 	cmd := exec.CommandContext(ctx, "sh", "-c", config.Start_Command)
// 	cmd.Stdout = os.Stdout
// 	cmd.Stderr = os.Stderr

// 	if err := cmd.Start(); err != nil {
// 		log.Printf("Failed to start command: %v", err)
// 		return
// 	}

// 	// Add the process to the channel
// 	processChan <- cmd.Process

// 	// Wait for the command to complete
// 	if err := cmd.Wait(); err != nil {
// 		log.Printf("Command exited with signal: %v", err)
// 	}

// 	log.Println("Command execution finished")
// }

// func watchFileChange(ctx context.Context, config models.Config, wg *sync.WaitGroup, processChan chan *os.Process) {
// 	defer wg.Done()

// 	watcher, err := fsnotify.NewWatcher()
// 	if err != nil {
// 		log.Fatalf("Error creating watcher: %s", err)
// 	}
// 	defer watcher.Close()

// 	// Add the work directory to the watcher
// 	err = watcher.Add(config.Work_dir)

// 	if err != nil {
// 		log.Fatalf("Error adding work directory to watcher: %s", err)
// 		return
// 	}
// 	log.Printf("Watching directory: %s", config.Work_dir)
// 	entries, _ := os.ReadDir(config.Work_dir)

// 	for _, entry := range entries {
// 		if entry.IsDir() {
// 			err = watcher.Add(entry.Name())
// 			if err != nil {
// 				log.Fatalf("Error adding sub directory %s to watcher: %s", entry.Name(), err)
// 				return
// 			}
// 			log.Printf("Watching sub directory: %s", entry.Name())
// 		}
// 	}

// 	for {
// 		select {
// 		case event, ok := <-watcher.Events:
// 			if !ok {
// 				return
// 			}
// 			if event.Has(fsnotify.Write) {
// 				inc_matched, _ := filepath.Match(config.File_Patterns, filepath.Base(event.Name))
// 				exc_matched, _ := filepath.Match(config.Exclude_Patterns, filepath.Base(event.Name))
// 				if inc_matched && !exc_matched {
// 					log.Println("Modified file:", event.Name)
// 					if proc := <-processChan; proc != nil {
// 						log.Println("Killing existing process...")
// 						if err := killProcess(proc); err != nil {
// 							log.Println("Error during process termination:", err)
// 							return
// 						}
// 					}
// 					wg.Add(1)
// 					go startProcess(ctx, config, wg, processChan)
// 				}
// 			}
// 		case err, ok := <-watcher.Errors:
// 			if !ok {
// 				return
// 			}
// 			log.Println("error:", err)
// 		case <-ctx.Done():
// 			log.Println("Stopping file watcher...")
// 			return
// 		}
// 	}
// }

// func main() {
// 	var wg sync.WaitGroup

// 	args, argsErr := GetArgs()
// 	if argsErr != nil {
// 		return
// 	}

// 	_, err := loadConfig(&args)

// 	if err != nil {
// 		fmt.Printf("Error occured: %s/n", err.Error())
// 		return
// 	}

// 	ctx, cancel := context.WithCancel(context.Background())

// 	signalChan := make(chan os.Signal, 1)
// 	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

// 	processChan := make(chan *os.Process, 1)

// 	wg.Add(1)
// 	go watchFileChange(ctx, args, &wg, processChan)

// 	wg.Add(1)
// 	go startProcess(ctx, args, &wg, processChan)

// 	<-signalChan
// 	log.Println("Received shutdown signal. Exiting...")

// 	cancel()
// 	wg.Wait()
// }

package main

import (
	"fmt"
	"net/http"
)

func main() {
	// Define a handler for the "/" route

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, World! This is a test route.")
	})

	http.HandleFunc("/greet", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, World! This is a greets route.")
	})

	// Start the server on port 8080
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Printf("Error starting server: %s\n", err)
		return
	}
	fmt.Println("Server is running on http://localhost:8080")
}
