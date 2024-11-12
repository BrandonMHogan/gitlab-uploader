package main

import (
    "fmt"
    "html/template"
    "io"
    "net/http"
    "os"
    "path/filepath"
    "strings"
)

type UploadResponse struct {
    Success bool
    Message string
}

func main() {
    http.HandleFunc("/", handleHome)
    http.HandleFunc("/upload", handleUpload)
    
    fmt.Println("Server starting on http://localhost:8080")
    http.ListenAndServe(":8080", nil)
}

func handleHome(w http.ResponseWriter, r *http.Request) {
    tmpl := template.Must(template.ParseFiles("index.html"))
    tmpl.Execute(w, nil)
}

func handleUpload(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    // Parse the multipart form
    err := r.ParseMultipartForm(10 << 20) // 10 MB max memory
    if err != nil {
        sendJSONResponse(w, false, "Failed to parse form: "+err.Error())
        return
    }

    baseURL := r.FormValue("gitlab_url")
    deployToken := r.FormValue("deploy_token")

    if baseURL == "" || deployToken == "" {
        sendJSONResponse(w, false, "GitLab URL and Deploy Token are required")
        return
    }

    // Get the files from form
    files := r.MultipartForm.File["files"]
    if len(files) == 0 {
        sendJSONResponse(w, false, "No files selected")
        return
    }

    if len(files) > 2 {
        sendJSONResponse(w, false, "Maximum 2 files allowed")
        return
    }

    var errors []string
    for _, fileHeader := range files {
        file, err := fileHeader.Open()
        if err != nil {
            errors = append(errors, fmt.Sprintf("Error opening %s: %v", fileHeader.Filename, err))
            continue
        }
        defer file.Close()

        // Create a temporary file
        tempFile, err := os.CreateTemp("", "upload-*"+filepath.Ext(fileHeader.Filename))
        if err != nil {
            errors = append(errors, fmt.Sprintf("Error creating temp file for %s: %v", fileHeader.Filename, err))
            continue
        }
        defer os.Remove(tempFile.Name())
        defer tempFile.Close()

        // Copy the uploaded file to the temporary file
        _, err = io.Copy(tempFile, file)
        if err != nil {
            errors = append(errors, fmt.Sprintf("Error copying %s: %v", fileHeader.Filename, err))
            continue
        }

        // Construct the full URL for this file
        fullURL := baseURL + "/" + fileHeader.Filename

        // Upload to GitLab
        err = uploadToGitLab(fullURL, deployToken, tempFile.Name())
        if err != nil {
            errors = append(errors, fmt.Sprintf("Error uploading %s: %v", fileHeader.Filename, err))
            continue
        }
    }

    if len(errors) > 0 {
        sendJSONResponse(w, false, "Errors occurred: "+strings.Join(errors, "; "))
    } else {
        sendJSONResponse(w, true, "All files uploaded successfully")
    }
}

func uploadToGitLab(gitlabURL, deployToken, filePath string) error {
    file, err := os.Open(filePath)
    if err != nil {
        return fmt.Errorf("error opening file: %v", err)
    }
    defer file.Close()

    // Create the request
    request, err := http.NewRequest("PUT", gitlabURL, file)
    if err != nil {
        return fmt.Errorf("error creating request: %v", err)
    }

    // Set headers
    request.Header.Set("Deploy-Token", deployToken)

    // Make the request
    client := &http.Client{}
    response, err := client.Do(request)
    if err != nil {
        return fmt.Errorf("error making request: %v", err)
    }
    defer response.Body.Close()

    if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusCreated {
        return fmt.Errorf("upload failed with status: %s", response.Status)
    }

    return nil
}

func sendJSONResponse(w http.ResponseWriter, success bool, message string) {
    w.Header().Set("Content-Type", "application/json")
    response := UploadResponse{
        Success: success,
        Message: message,
    }
    fmt.Fprintf(w, `{"success":%t,"message":"%s"}`, response.Success, response.Message)
}