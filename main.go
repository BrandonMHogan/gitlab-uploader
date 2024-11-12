package main

import (
    "encoding/xml"
    "fmt"
    "html/template"
    "io"
    "mime/multipart"
    "net/http"
    "os"
    "path/filepath"
    "strings"
)
type UploadResponse struct {
    Success bool
    Message string
    Details string
}

type PomProject struct {
    XMLName    xml.Name `xml:"project"`
    GroupId    string   `xml:"groupId"`
    ArtifactId string   `xml:"artifactId"`
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

func parsePomFile(file io.Reader) (*PomProject, error) {
    var project PomProject
    decoder := xml.NewDecoder(file)
    err := decoder.Decode(&project)
    if err != nil {
        return nil, fmt.Errorf("failed to parse POM file: %v", err)
    }
    
    if project.GroupId == "" {
        return nil, fmt.Errorf("GroupId not found in POM file")
    }
    if project.ArtifactId == "" {
        return nil, fmt.Errorf("ArtifactId not found in POM file")
    }
    
    return &project, nil
}

func handleUpload(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    err := r.ParseMultipartForm(10 << 20)
    if err != nil {
        sendJSONResponse(w, false, "Failed to parse form", err.Error())
        return
    }

    projectID := r.FormValue("project_id")
    deployToken := r.FormValue("deploy_token")
    version := r.FormValue("version")

    if projectID == "" || deployToken == "" || version == "" {
        sendJSONResponse(w, false, "All fields are required", "Missing required fields")
        return
    }

    // Get the files from form
    files := r.MultipartForm.File["files"]
    if len(files) != 2 {
        sendJSONResponse(w, false, "Exactly two files (AAR and POM) are required", "Invalid number of files")
        return
    }

    // Find the POM file
    var pomFile *multipart.FileHeader
    for _, file := range files {
        if strings.HasSuffix(file.Filename, ".pom") {
            pomFile = file
            break
        }
    }

    if pomFile == nil {
        sendJSONResponse(w, false, "POM file is required", "No POM file found")
        return
    }

    // Parse POM file
    pFile, err := pomFile.Open()
    if err != nil {
        sendJSONResponse(w, false, "Failed to open POM file", err.Error())
        return
    }
    defer pFile.Close()

    pomData, err := parsePomFile(pFile)
    if err != nil {
        sendJSONResponse(w, false, "Failed to parse POM file", err.Error())
        return
    }

    fmt.Printf("Parsed POM file - GroupId: %s, ArtifactId: %s\n", pomData.GroupId, pomData.ArtifactId)

    // Construct base URL
    baseURL := fmt.Sprintf("https://code.q2developer.com/api/v4/projects/%s/packages/maven/%s/%s",
        projectID,
        strings.ReplaceAll(pomData.GroupId, ".", "/"),
        pomData.ArtifactId)

    fmt.Printf("Constructed base URL: %s\n", baseURL)

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

        _, err = io.Copy(tempFile, file)
        if err != nil {
            errors = append(errors, fmt.Sprintf("Error copying %s: %v", fileHeader.Filename, err))
            continue
        }

        _, err = tempFile.Seek(0, 0)
        if err != nil {
            errors = append(errors, fmt.Sprintf("Error preparing %s for upload: %v", fileHeader.Filename, err))
            continue
        }

        // Construct full URL for this file
        fullURL := fmt.Sprintf("%s/%s/%s", baseURL, version, fileHeader.Filename)
        fmt.Printf("Uploading %s to: %s\n", fileHeader.Filename, fullURL)

        err = uploadToGitLab(fullURL, deployToken, tempFile.Name())
        if err != nil {
            errors = append(errors, fmt.Sprintf("Error uploading %s: %v", fileHeader.Filename, err))
            continue
        }
    }

    if len(errors) > 0 {
        sendJSONResponse(w, false, "Errors occurred during upload", strings.Join(errors, "; "))
    } else {
        implementationPath := fmt.Sprintf("implementation '%s:%s:%s'", 
            pomData.GroupId,
            pomData.ArtifactId,
            version)
        sendJSONResponse(w, true, fmt.Sprintf("All files uploaded successfully. Use: %s", implementationPath), "")
    }
}

func uploadToGitLab(gitlabURL, deployToken, filePath string) error {
    file, err := os.Open(filePath)
    if err != nil {
        return fmt.Errorf("error opening file: %v", err)
    }
    defer file.Close()

    stat, err := file.Stat()
    if err != nil {
        return fmt.Errorf("error getting file stats: %v", err)
    }

    fmt.Printf("Uploading to URL: %s\n", gitlabURL)

    request, err := http.NewRequest("PUT", gitlabURL, file)
    if err != nil {
        return fmt.Errorf("error creating request: %v", err)
    }

    request.Header.Set("Deploy-Token", deployToken)
    request.Header.Set("Content-Type", "application/octet-stream")
    request.ContentLength = stat.Size()

    client := &http.Client{
        CheckRedirect: func(req *http.Request, via []*http.Request) error {
            for key, value := range via[0].Header {
                req.Header[key] = value
            }
            return nil
        },
    }

    response, err := client.Do(request)
    if err != nil {
        return fmt.Errorf("error making request: %v", err)
    }
    defer response.Body.Close()

    bodyBytes, err := io.ReadAll(response.Body)
    if err != nil {
        return fmt.Errorf("error reading response body: %v", err)
    }
    bodyString := string(bodyBytes)

    fmt.Printf("Response Status: %s\n", response.Status)
    fmt.Printf("Response Body: %s\n", bodyString)

    if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusCreated {
        // Map common status codes to user-friendly messages
        var errorMessage string
        switch response.StatusCode {
        case http.StatusUnauthorized:
            errorMessage = "Unauthorized: Please check your deploy token"
        case http.StatusForbidden:
            errorMessage = "Forbidden: You don't have permission to upload to this project"
        case http.StatusNotFound:
            errorMessage = "Not Found: The project ID may be incorrect"
        default:
            errorMessage = fmt.Sprintf("Upload failed with status %d", response.StatusCode)
        }
        return fmt.Errorf("%s\nResponse: %s", errorMessage, bodyString)
    }

    return nil
}

func sendJSONResponse(w http.ResponseWriter, success bool, message, details string) {
    w.Header().Set("Content-Type", "application/json")
    response := UploadResponse{
        Success: success,
        Message: message,
        Details: details,
    }
    fmt.Fprintf(w, `{"success":%t,"message":"%s","details":"%s"}`, 
        response.Success, response.Message, response.Details)
}