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

    err := r.ParseMultipartForm(10 << 20)
    if err != nil {
        sendJSONResponse(w, false, "Failed to parse form: "+err.Error())
        return
    }

    baseURL := r.FormValue("gitlab_url")
    deployToken := r.FormValue("deploy_token")
    artifactName := r.FormValue("artifact_name")
    version := r.FormValue("version")

    if baseURL == "" || deployToken == "" || artifactName == "" || version == "" {
        sendJSONResponse(w, false, "All fields are required")
        return
    }

    files := r.MultipartForm.File["files"]
    if len(files) == 0 {
        sendJSONResponse(w, false, "No files selected")
        return
    }

    if len(files) > 2 {
        sendJSONResponse(w, false, "Maximum 2 files allowed")
        return
    }

    // Ensure the base URL doesn't end with a slash
    baseURL = strings.TrimSuffix(baseURL, "/")
    
    // Construct the full URL with artifact name and version
    uploadBaseURL := fmt.Sprintf("%s/%s/%s", baseURL, artifactName, version)

    var errors []string
    for _, fileHeader := range files {
        file, err := fileHeader.Open()
        if err != nil {
            errors = append(errors, fmt.Sprintf("Error opening %s: %v", fileHeader.Filename, err))
            continue
        }
        defer file.Close()

        // Construct the correct filename
        ext := filepath.Ext(fileHeader.Filename)
        newFilename := fmt.Sprintf("%s-%s%s", artifactName, version, ext)
        
        // Create the full URL for this specific file
        fullURL := fmt.Sprintf("%s/%s", uploadBaseURL, newFilename)

        // Create a temporary file
        tempFile, err := os.CreateTemp("", "upload-*"+ext)
        if err != nil {
            errors = append(errors, fmt.Sprintf("Error creating temp file for %s: %v", newFilename, err))
            continue
        }
        defer os.Remove(tempFile.Name())
        defer tempFile.Close()

        _, err = io.Copy(tempFile, file)
        if err != nil {
            errors = append(errors, fmt.Sprintf("Error copying %s: %v", newFilename, err))
            continue
        }

        // Rewind the temp file for reading
        _, err = tempFile.Seek(0, 0)
        if err != nil {
            errors = append(errors, fmt.Sprintf("Error preparing %s for upload: %v", newFilename, err))
            continue
        }

        err = uploadToGitLab(fullURL, deployToken, tempFile.Name())
        if err != nil {
            errors = append(errors, fmt.Sprintf("Error uploading %s: %v", newFilename, err))
            continue
        }
    }

    if len(errors) > 0 {
        sendJSONResponse(w, false, "Errors occurred: "+strings.Join(errors, "; "))
    } else {
        implementationPath := fmt.Sprintf("implementation '%s:%s:%s'", 
            strings.ReplaceAll(baseURL[strings.LastIndex(baseURL, "maven/")+6:], "/", "."),
            artifactName,
            version)
        sendJSONResponse(w, true, fmt.Sprintf("All files uploaded successfully. Use: %s", implementationPath))
    }
}

func uploadToGitLab(gitlabURL, deployToken, filePath string) error {
    file, err := os.Open(filePath)
    if err != nil {
        return fmt.Errorf("error opening file: %v", err)
    }
    defer file.Close()

    request, err := http.NewRequest("PUT", gitlabURL, file)
    if err != nil {
        return fmt.Errorf("error creating request: %v", err)
    }

    request.Header.Set("Deploy-Token", deployToken)

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