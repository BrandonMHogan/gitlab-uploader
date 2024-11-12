package handlers

import (
    "encoding/json"
    "net/http"
    
    "gitlab-uploader/internal/gitlab"
)

type UploadHandler struct {
    gitlabClient *gitlab.Client
}

func NewUploadHandler() *UploadHandler {
    return &UploadHandler{
        gitlabClient: gitlab.NewClient(),
    }
}

func (h *UploadHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    if r.Method == "POST" {
        h.handleUpload(w, r)
    } else if r.Method == "HEAD" {
        h.handleFileCheck(w, r)
    } else {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
    }
}

func (h *UploadHandler) handleFileCheck(w http.ResponseWriter, r *http.Request) {
    url := r.URL.Query().Get("url")
    deployToken := r.URL.Query().Get("token")

    if url == "" || deployToken == "" {
        http.Error(w, "Missing required parameters", http.StatusBadRequest)
        return
    }

    result, err := h.gitlabClient.CheckFileExists(url, deployToken)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    json.NewEncoder(w).Encode(result)
}

func (h *UploadHandler) handleUpload(w http.ResponseWriter, r *http.Request) {
    // ... rest of your existing upload handling code ...
    // Make sure to use the gitlabClient for uploads
}

// Helper functions like parsePomFile remain the same