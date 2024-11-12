package handlers

import (
    "html/template"
    "net/http"
    
    "gitlab-uploader/internal/config"
    "gitlab-uploader/internal/models"
)

type HomeHandler struct {
    tmpl *template.Template
}

func NewHomeHandler() *HomeHandler {
    return &HomeHandler{
        tmpl: template.Must(template.ParseFiles("templates/index.html")),
    }
}

func (h *HomeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    partners, err := config.LoadPartnersConfig()
    if err != nil {
        http.Error(w, "Failed to load partners configuration", http.StatusInternalServerError)
        return
    }

    data := struct {
        Partners []models.Partner
    }{
        Partners: partners,
    }

    h.tmpl.Execute(w, data)
}