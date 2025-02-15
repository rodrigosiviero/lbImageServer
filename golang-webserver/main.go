package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/eventlog"
	"golang.org/x/sys/windows/svc/mgr"
)

// Config holds the settings for the server
type Config struct {
	Port   string `json:"port"`
	Folder string `json:"folder"`
}

// Service structure with embedded dependencies
type Service struct {
	server     *http.Server
	elog       *eventlog.Log
	config     *Config
	isRunning  bool
	runningMux sync.Mutex
}

// LoadConfig reads the configuration file from the executable's directory
func LoadConfig(filename string) (*Config, error) {
	exePath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("failed to get executable path: %w", err)
	}

	configPath := filepath.Join(filepath.Dir(exePath), filename)
	file, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file %s: %w", configPath, err)
	}
	defer file.Close()

	var config Config
	if err := json.NewDecoder(file).Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode config: %w", err)
	}

	// Validate config
	if config.Port == "" {
		return nil, fmt.Errorf("port cannot be empty")
	}
	if config.Folder == "" {
		return nil, fmt.Errorf("folder cannot be empty")
	}

	// Verify folder exists
	if _, err := os.Stat(config.Folder); os.IsNotExist(err) {
		return nil, fmt.Errorf("folder does not exist: %s", config.Folder)
	}

	return &config, nil
}

func (s *Service) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (bool, uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown
	changes <- svc.Status{State: svc.StartPending, Accepts: cmdsAccepted, WaitHint: 10000}

	s.elog.Info(1, "Service Execute started")

	// Initialize context for shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start server in goroutine
	errChan := make(chan error, 1)
	go func() {
		s.elog.Info(1, fmt.Sprintf("Starting HTTP server on port %s serving folder %s", s.config.Port, s.config.Folder))

		s.runningMux.Lock()
		s.isRunning = true
		s.runningMux.Unlock()

		if err := s.server.ListenAndServe(); err != http.ErrServerClosed {
			s.elog.Error(1, fmt.Sprintf("HTTP server error: %v", err))
			errChan <- err
		}
	}()

	// Wait a moment to ensure server starts
	time.Sleep(1 * time.Second)

	// Update status to running
	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
	s.elog.Info(1, "Service status set to running")

	// Service loop
	for {
		select {
		case err := <-errChan:
			s.elog.Error(1, fmt.Sprintf("Server error: %v", err))
			return false, 1
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				s.elog.Info(1, "Service interrogate received")
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				s.elog.Info(1, "Service stop/shutdown received")
				changes <- svc.Status{State: svc.StopPending, Accepts: cmdsAccepted, WaitHint: 10000}

				// Graceful shutdown
				shutdownCtx, cancelShutdown := context.WithTimeout(ctx, 5*time.Second)
				defer cancelShutdown()

				s.runningMux.Lock()
				s.isRunning = false
				s.runningMux.Unlock()

				if err := s.server.Shutdown(shutdownCtx); err != nil {
					s.elog.Error(1, fmt.Sprintf("Error during shutdown: %v", err))
				}

				s.elog.Info(1, "Service stopped successfully")
				return false, 0
			default:
				s.elog.Error(1, fmt.Sprintf("Unexpected control request: %d", c))
			}
		}
	}
}

func createServer(config *Config) *http.Server {
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir(config.Folder)))

	return &http.Server{
		Addr:         ":" + config.Port,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "install":
			if err := installService(); err != nil {
				log.Fatal(err)
			}
			fmt.Println("Service installed successfully")
			return
		case "remove":
			if err := removeService(); err != nil {
				log.Fatal(err)
			}
			fmt.Println("Service removed successfully")
			return
		case "debug":
			// Run in debug mode with console logging
			config, err := LoadConfig("config.json")
			if err != nil {
				log.Fatal(err)
			}
			server := createServer(config)
			log.Printf("Debug mode: Serving %s on port %s\n", config.Folder, config.Port)
			if err := server.ListenAndServe(); err != nil {
				log.Fatal(err)
			}
			return
		}
	}

	// Running as service
	isService, err := svc.IsWindowsService()
	if err != nil {
		log.Fatalf("Failed to determine if running as service: %v", err)
	}
	if !isService {
		log.Fatal("This program can only be run as a Windows service or with the debug flag")
	}

	// Initialize event logger
	elog, err := eventlog.Open("ImageServer")
	if err != nil {
		log.Fatal("Failed to open event log:", err)
	}
	defer elog.Close()

	elog.Info(1, "Service starting...")

	// Load configuration
	config, err := LoadConfig("config.json")
	if err != nil {
		elog.Error(1, fmt.Sprintf("Failed to load config: %v", err))
		log.Fatal(err)
	}

	// Create service instance
	srv := &Service{
		server: createServer(config),
		elog:   elog,
		config: config,
	}

	// Run service
	err = svc.Run("ImageServer", srv)
	if err != nil {
		elog.Error(1, fmt.Sprintf("Service failed: %v", err))
		log.Fatal(err)
	}
}

// Add these functions after the main() function:

func installService() error {
	// First try to remove any existing event logger
	eventlog.Remove("ImageServer") // Ignore error as it might not exist

	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer m.Disconnect()

	// Check if service already exists
	s, err := m.OpenService("ImageServer")
	if err == nil {
		s.Close()
		return fmt.Errorf("service already exists - please remove it first")
	}

	s, err = m.CreateService(
		"ImageServer",
		exePath,
		mgr.Config{
			StartType:        mgr.StartAutomatic,
			DisplayName:      "Image Server",
			Description:      "A simple image-serving web server.",
			ServiceStartName: "LocalSystem",
		},
		"service",
	)
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}
	defer s.Close()

	// Install event logger
	err = eventlog.InstallAsEventCreate("ImageServer", eventlog.Error|eventlog.Warning|eventlog.Info)
	if err != nil {
		// If it fails because it already exists, that's okay
		if !strings.Contains(err.Error(), "registry key already exists") {
			s.Delete()
			return fmt.Errorf("failed to install event logger: %w", err)
		}
	}

	return nil
}

func removeService() error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService("ImageServer")
	if err != nil {
		return fmt.Errorf("failed to open service: %w", err)
	}
	defer s.Close()

	// First stop the service if it's running
	status, err := s.Query()
	if err == nil && status.State != svc.Stopped {
		_, err = s.Control(svc.Stop)
		if err != nil {
			return fmt.Errorf("failed to stop service: %w", err)
		}
		// Wait for the service to stop
		for status.State != svc.Stopped {
			time.Sleep(time.Second)
			status, err = s.Query()
			if err != nil {
				break
			}
		}
	}

	err = s.Delete()
	if err != nil {
		return fmt.Errorf("failed to delete service: %w", err)
	}

	err = eventlog.Remove("ImageServer")
	if err != nil {
		// Ignore error if event logger doesn't exist
		if !strings.Contains(err.Error(), "registry key does not exist") {
			return fmt.Errorf("failed to remove event logger: %w", err)
		}
	}

	return nil
}
