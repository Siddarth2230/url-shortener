package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/Siddarth2230/url-shortener/internal/models"
	"github.com/Siddarth2230/url-shortener/internal/service"
)

type URLHandler struct {
	service *service.URLService
}

func NewURLHandler(svc *service.URLService) *URLHandler {
	return &URLHandler{service: svc}
}

// POST /shorten
func (h *URLHandler) ShortenURL(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// decode request
	var req models.ShortenRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request payload")
		return
	}

	// call service
	resp, err := h.service.ShortenURL(ctx, req)
	if err != nil {
		// map service errors to HTTP responses
		switch err {
		case service.ErrInvalidURL:
			writeError(w, http.StatusBadRequest, err.Error())
			return
		case service.ErrCustomCodeTaken:
			writeError(w, http.StatusConflict, err.Error()) // 409 Conflict
			return
		default:
			// unknown/internal error
			log.Printf("ShortenURL error: %v", err)
			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}
	}

	// success: return 201 Created with JSON body
	writeJSON(w, http.StatusCreated, resp)
}

// GET /{shortCode} - redirect to long URL
func (h *URLHandler) RedirectURL(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	vars := mux.Vars(r)
	shortCode, ok := vars["shortCode"]
	if !ok || shortCode == "" {
		writeError(w, http.StatusBadRequest, "missing short code")
		return
	}

	longURL, err := h.service.GetLongURL(ctx, shortCode)
	if err != nil {
		switch err {
		case service.ErrNotFound:
			writeError(w, http.StatusNotFound, "short code not found")
			return
		case service.ErrExpired:
			writeError(w, http.StatusGone, "short URL expired")
			return
		default:
			log.Printf("RedirectURL error: %v", err)
			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}
	}

	// Redirect (302 Found). Use 302 so browsers use it as a temporary redirect by default.
	http.Redirect(w, r, longURL, http.StatusFound)
}

// helper: write JSON response
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(v); err != nil {
		// Log encoding error (can't write response now)
		log.Printf("writeJSON encode error: %v", err)
	}
}

// helper: write an error message in JSON form { "error": "msg" }
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
