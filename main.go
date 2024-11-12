package main

import (
    "fmt"
    "net/http"
    
    "gitlab-uploader/internal/handlers"
)

func main() {
    // Serve static files
    fs := http.FileServer(http.Dir("static"))
    http.Handle("/static/", http.StripPrefix("/static/", fs))

    // Register handlers
    http.Handle("/", handlers.NewHomeHandler())
    http.Handle("/upload", handlers.NewUploadHandler())

    fmt.Println("Server starting on http://localhost:8080")
    http.ListenAndServe(":8080", nil)
}