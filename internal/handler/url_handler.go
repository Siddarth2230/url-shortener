package handler

import (
	"net/http"

	"github.com/Siddarth2230/url-shortener/internal/service"
)

type URLHandler struct {
	service *service.URLService
}

func NewURLHandler(service *service.URLService) *URLHandler {
	return &URLHandler{service: service}
}

func (h *URLHandler) ShortenURL(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement POST /shorten
	// 1. Decode JSON request
	// 2. Call service.ShortenURL
	// 3. Return JSON response (201 Created)
	// 4. Handle errors (400 Bad Request, 500 Internal Server Error)
}

func (h *URLHandler) RedirectURL(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement GET /:shortCode
	// 1. Extract shortCode from URL path
	// 2. Call service.GetLongURL
	// 3. Return 301/302 redirect
	// 4. Handle not found (404)
}
