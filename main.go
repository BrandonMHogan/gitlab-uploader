package main

import (
    "fmt"
    "log"
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
    fmt.Println("Press Ctrl+C to stop the server")

    // Add middleware for logging requests
    handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        log.Printf("%s %s", r.Method, r.URL.Path)
        http.DefaultServeMux.ServeHTTP(w, r)
    })

    if err := http.ListenAndServe(":8080", handler); err != nil {
        log.Fatal("Server error:", err)
    }
}