package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
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

func watchFileChange(ctx context.Context, config models.Config, wg *sync.WaitGroup) {
	defer wg.Done()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalf("Error creating watcher: %s", err)
	}
	defer watcher.Close()

	// Add the work directory to the watcher
	err = watcher.Add(config.Work_dir)

	if err != nil {
		log.Fatalf("Error adding work directory to watcher: %s", err)
		return
	}
	log.Printf("Watching directory: %s", config.Work_dir)
	entries, _ := os.ReadDir(config.Work_dir)

	for _, entry := range entries {
		if entry.IsDir() {
			err = watcher.Add(entry.Name())
			if err != nil {
				log.Fatalf("Error adding sub directory %s to watcher: %s", entry.Name(), err)
				return
			}
			log.Printf("Watching sub directory: %s", entry.Name())
		}
	}

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Has(fsnotify.Write) {
				inc_matched, _ := filepath.Match(config.File_Patterns, filepath.Base(event.Name))
				exc_matched, _ := filepath.Match(config.Exclude_Patterns, filepath.Base(event.Name))
				if inc_matched && !exc_matched {
					log.Println("Modified file:", event.Name, inc_matched)
				}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Println("error:", err)
		case <-ctx.Done():
			log.Println("Stopping file watcher...")
			return
		}
	}
}

func main() {
	var wg sync.WaitGroup

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

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	wg.Add(1)
	go watchFileChange(ctx, args, &wg)

	<-signalChan
	log.Println("Received shutdown signal. Exiting...")
	cancel()
	wg.Wait()
}
