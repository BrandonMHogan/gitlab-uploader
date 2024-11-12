package config

import (
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
    
    "gitlab-uploader/internal/models"
)

func LoadPartnersConfig() ([]models.Partner, error) {
    file, err := os.ReadFile("partners.json")
    if err != nil {
        return nil, fmt.Errorf("error reading partners config: %v", err)
    }

    var config models.PartnersConfig
    if err := json.Unmarshal(file, &config); err != nil {
        return nil, fmt.Errorf("error parsing partners config: %v", err)
    }

    return config.Partners, nil
}