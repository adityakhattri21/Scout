package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

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
		fmt.Printf("Error occurred: %s\n", err)
		return models.Config{}, err
	}

	config := models.Config{File_Location: configFile.Name()}

	return config, nil
}

func loadConfig(config *models.Config) (string, error) {
	data, err := os.ReadFile(config.File_Location)

	if err != nil {
		return "", err
	}

	err = yaml.Unmarshal(data, config)

	if err != nil {
		return "", err
	}
	return "", nil
}

func watchFileChange(_ context.Context, config models.Config) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalf("Error creating watcher: %s", err)
	}
	defer watcher.Close()

	// Add the work directory to the watcher

	entries, _ := os.ReadDir(config.Work_dir)

	for _, entry := range entries {
		fmt.Println(entry)
	}
	err = watcher.Add(config.Work_dir)
	if err != nil {
		log.Fatalf("Error adding directory to watcher: %s", err)
	}

	log.Printf("Watching directory: %s", config.Work_dir)

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			log.Println("event:", event)
			if event.Has(fsnotify.Write) {
				log.Println("modified file:", event.Name)
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Println("error:", err)
		}
	}

	// // Start watching for events
	// go func() {
	// 	for {
	// 		select {
	// 		case event, ok := <-watcher.Events:
	// 			if !ok {
	// 				return
	// 			}

	// 			// Match file patterns
	// 			for _, pattern := range config.File_patterns {
	// 				matched, _ := filepath.Match(pattern, filepath.Base(event.Name))
	// 				if matched {
	// 					log.Printf("File change detected: %s, Event: %s", event.Name, event.Op)
	// 					// Handle the detected event (e.g., reload or trigger actions)
	// 				}
	// 			}

	// 		case err, ok := <-watcher.Errors:
	// 			if !ok {
	// 				return
	// 			}
	// 			log.Printf("Error watching files: %s", err)

	// 		case <-ctx.Done():
	// 			log.Println("Stopping file watcher...")
	// 			return
	// 		}
	// 	}
	// }()
}

func main() {
	args, argsErr := GetArgs()
	if argsErr != nil {
		return
	}

	_, err := loadConfig(&args)

	if err != nil {
		fmt.Printf("Error occured: %s/n", err.Error())
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	go watchFileChange(ctx, args)

	<-signalChan
	log.Println("Received shutdown signal. Exiting...")
}
