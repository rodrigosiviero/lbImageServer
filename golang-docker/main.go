package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

// ServeFiles starts the HTTP server to serve files from the specified folder
func ServeFiles(port, folder string) {
	// Custom handler to log every request
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Received request: %s %s", r.Method, r.URL.Path)
		http.FileServer(http.Dir(folder)).ServeHTTP(w, r)
	})

	fmt.Println("Serving", folder, "on port", port)
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatal("Error starting server:", err)
	}
}

func main() {
	// Read config from environment variables
	port := os.Getenv("PORT")
	if port == "" {
		port = "8089" // Default port
		log.Println("PORT environment variable not set, using default port:", port)
	}

	folder := os.Getenv("IMAGE_FOLDER")
	if folder == "" {
		folder = "/images" // Default folder inside the container
		log.Println("IMAGE_FOLDER environment variable not set, using default folder:", folder)
	}

	// Check if the folder exists
	if _, err := os.Stat(folder); os.IsNotExist(err) {
		log.Fatalf("The folder %s does not exist", folder)
	}

	// Start HTTP server
	ServeFiles(port, folder)
}
