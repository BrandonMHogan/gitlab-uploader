package models

import (
    "encoding/xml"
)

type Partner struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
}

type PartnersConfig struct {
    Partners []Partner `json:"partners"`
}

type UploadResponse struct {
    Success bool   `json:"success"`
    Message string `json:"message"`
    Details string `json:"details"`
}

type FileCheckResponse struct {
    Exists    bool   `json:"exists"`
    FileName  string `json:"fileName"`
    FileURL   string `json:"fileUrl"`
    UpdatedAt string `json:"updatedAt"`
}

type PomProject struct {
    XMLName    xml.Name `xml:"project"`
    GroupId    string   `xml:"groupId"`
    ArtifactId string   `xml:"artifactId"`
}