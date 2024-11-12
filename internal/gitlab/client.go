package gitlab

import (
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "os"
    "time"
    
    "gitlab-uploader/internal/models"
)

type Client struct {
    baseURL string
    client  *http.Client
}

func NewClient() *Client {
    return &Client{
        baseURL: "https://code.q2developer.com/api/v4",
        client: &http.Client{
            Timeout: time.Second * 30,
            CheckRedirect: func(req *http.Request, via []*http.Request) error {
                for key, value := range via[0].Header {
                    req.Header[key] = value
                }
                return nil
            },
        },
    }
}

func (c *Client) CheckFileExists(url, deployToken string) (*models.FileCheckResponse, error) {
    req, err := http.NewRequest("HEAD", url, nil)
    if err != nil {
        return nil, err
    }

    req.Header.Set("Deploy-Token", deployToken)
    
    resp, err := c.client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    exists := resp.StatusCode == http.StatusOK
    
    if exists {
        return &models.FileCheckResponse{
            Exists:    true,
            FileName:  filepath.Base(url),
            FileURL:   url,
            UpdatedAt: resp.Header.Get("Last-Modified"),
        }, nil
    }

    return &models.FileCheckResponse{Exists: false}, nil
}

func (c *Client) UploadFile(url, deployToken, filePath string) error {
    file, err := os.Open(filePath)
    if err != nil {
        return fmt.Errorf("error opening file: %v", err)
    }
    defer file.Close()

    stat, err := file.Stat()
    if err != nil {
        return fmt.Errorf("error getting file stats: %v", err)
    }

    req, err := http.NewRequest("PUT", url, file)
    if err != nil {
        return fmt.Errorf("error creating request: %v", err)
    }

    req.Header.Set("Deploy-Token", deployToken)
    req.Header.Set("Content-Type", "application/octet-stream")
    req.ContentLength = stat.Size()

    resp, err := c.client.Do(req)
    if err != nil {
        return fmt.Errorf("error making request: %v", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
        body, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(body))
    }

    return nil
}