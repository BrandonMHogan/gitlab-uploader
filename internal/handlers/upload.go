package handlers

import (
    "encoding/json"
    "encoding/xml"
    "fmt"
    "io"
    "log"
    "net/http"
    "os"
    "path/filepath"
    "strings"
    "mime/multipart"
    
    "gitlab-uploader/internal/gitlab"
    "gitlab-uploader/internal/models"
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
    } else {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
    }
}

func (h *UploadHandler) handleUpload(w http.ResponseWriter, r *http.Request) {
    log.Println("Starting file upload...")
    
    // Parse multipart form
    err := r.ParseMultipartForm(32 << 20) // 32MB max memory
    if err != nil {
        log.Printf("Error parsing multipart form: %v", err)
        sendJSONResponse(w, false, "Failed to parse upload form", err.Error())
        return
    }

    // Get form values
    projectID := r.FormValue("project_id")
    version := r.FormValue("version")
    deployToken := r.FormValue("deploy_token")

    log.Printf("Upload request - Project ID: %s, Version: %s", projectID, version)

    if projectID == "" || version == "" || deployToken == "" {
        sendJSONResponse(w, false, "Missing required fields", "Project ID, version, and deploy token are required")
        return
    }

    // Get uploaded files
    files := r.MultipartForm.File["files"]
    if len(files) == 0 {
        sendJSONResponse(w, false, "No files uploaded", "Please select files to upload")
        return
    }

    // Find and parse POM file first
    var pomFile *multipart.FileHeader
    var aarFile *multipart.FileHeader
    for _, file := range files {
        if strings.HasSuffix(file.Filename, ".pom") {
            pomFile = file
        } else if strings.HasSuffix(file.Filename, ".aar") {
            aarFile = file
        }
    }

    if pomFile == nil {
        sendJSONResponse(w, false, "Missing POM file", "Please provide a POM file")
        return
    }

    if aarFile == nil {
        sendJSONResponse(w, false, "Missing AAR file", "Please provide an AAR file")
        return
    }

    // Parse POM file to get groupId
    pomData, err := h.parsePomFile(pomFile)
    if err != nil {
        log.Printf("Error parsing POM file: %v", err)
        sendJSONResponse(w, false, "Failed to parse POM file", err.Error())
        return
    }

    log.Printf("Parsed POM - GroupId: %s, ArtifactId: %s", pomData.GroupId, pomData.ArtifactId)

    // Construct base URL
    baseURL := fmt.Sprintf("https://code.q2developer.com/api/v4/projects/%s/packages/maven/%s/%s",
        projectID,
        strings.ReplaceAll(pomData.GroupId, ".", "/"),
        pomData.ArtifactId)

    log.Printf("Base URL constructed: %s", baseURL)

    // Upload both files
    for _, fileHeader := range []*multipart.FileHeader{pomFile, aarFile} {
        log.Printf("Processing file: %s", fileHeader.Filename)
        
        file, err := fileHeader.Open()
        if err != nil {
            log.Printf("Error opening file %s: %v", fileHeader.Filename, err)
            sendJSONResponse(w, false, "Failed to process file", err.Error())
            return
        }
        defer file.Close()

        // Create temporary file
        tempFile, err := os.CreateTemp("", "upload-*"+filepath.Ext(fileHeader.Filename))
        if err != nil {
            log.Printf("Error creating temp file for %s: %v", fileHeader.Filename, err)
            sendJSONResponse(w, false, "Failed to process file", err.Error())
            return
        }
        tempPath := tempFile.Name()
        defer os.Remove(tempPath)
        defer tempFile.Close()

        // Copy file to temp
        _, err = io.Copy(tempFile, file)
        if err != nil {
            log.Printf("Error copying file %s: %v", fileHeader.Filename, err)
            sendJSONResponse(w, false, "Failed to process file", err.Error())
            return
        }

        // Construct full URL for this file
        fullURL := fmt.Sprintf("%s/%s/%s", baseURL, version, fileHeader.Filename)
        log.Printf("Uploading to URL: %s", fullURL)

        // Upload file
        err = h.gitlabClient.UploadFile(fullURL, deployToken, tempPath)
        if err != nil {
            log.Printf("Error uploading file %s: %v", fileHeader.Filename, err)
            sendJSONResponse(w, false, "Upload failed", err.Error())
            return
        }

        log.Printf("Successfully uploaded: %s", fileHeader.Filename)
    }

    // Success response
    implementationPath := fmt.Sprintf("implementation '%s:%s:%s'", 
        pomData.GroupId,
        pomData.ArtifactId,
        version)
    
    sendJSONResponse(w, true, "Files uploaded successfully", implementationPath)
}

func (h *UploadHandler) parsePomFile(fileHeader *multipart.FileHeader) (*models.PomProject, error) {
    file, err := fileHeader.Open()
    if err != nil {
        return nil, fmt.Errorf("error opening POM file: %v", err)
    }
    defer file.Close()

    var project models.PomProject
    decoder := xml.NewDecoder(file)
    if err := decoder.Decode(&project); err != nil {
        return nil, fmt.Errorf("error parsing POM XML: %v", err)
    }

    if project.GroupId == "" {
        return nil, fmt.Errorf("GroupId not found in POM file")
    }

    if project.ArtifactId == "" {
        return nil, fmt.Errorf("ArtifactId not found in POM file")
    }

    return &project, nil
}

func sendJSONResponse(w http.ResponseWriter, success bool, message, details string) {
    w.Header().Set("Content-Type", "application/json")
    response := models.UploadResponse{
        Success: success,
        Message: message,
        Details: details,
    }
    json.NewEncoder(w).Encode(response)
}