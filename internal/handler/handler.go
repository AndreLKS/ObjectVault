package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/andrelks/objectvault/internal/metadata"
	"github.com/andrelks/objectvault/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type APIHandler struct {
	bucketService service.BucketService
	objectService service.ObjectService
}

func NewAPIHandler(bs service.BucketService, os service.ObjectService) *APIHandler {
	return &APIHandler{
		bucketService: bs,
		objectService: os,
	}
}

// Router sets up middleware and routes.
func (h *APIHandler) Router() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Buckets
	r.Post("/buckets", h.CreateBucket)
	r.Get("/buckets", h.ListBuckets)

	// Objects inside buckets
	r.Get("/buckets/{bucketName}/objects", h.ListObjects)
	r.Put("/buckets/{bucketName}/objects/*", h.UploadObject)
	r.Get("/buckets/{bucketName}/objects/*", h.DownloadObject)

	return r
}

type CreateBucketRequest struct {
	Name string `json:"name"`
}

func (h *APIHandler) CreateBucket(w http.ResponseWriter, r *http.Request) {
	var req CreateBucketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	bucket, err := h.bucketService.CreateBucket(r.Context(), req.Name)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidBucketName):
			h.writeError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, metadata.ErrBucketExists):
			h.writeError(w, http.StatusConflict, err.Error())
		default:
			h.writeError(w, http.StatusInternalServerError, "failed to create bucket")
		}
		return
	}

	h.writeJSON(w, http.StatusCreated, bucket)
}

func (h *APIHandler) ListBuckets(w http.ResponseWriter, r *http.Request) {
	buckets, err := h.bucketService.ListBuckets(r.Context())
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to list buckets")
		return
	}

	h.writeJSON(w, http.StatusOK, buckets)
}

func (h *APIHandler) UploadObject(w http.ResponseWriter, r *http.Request) {
	bucketName := chi.URLParam(r, "bucketName")
	objectKey := chi.URLParam(r, "*")

	if objectKey == "" {
		h.writeError(w, http.StatusBadRequest, "object key is required")
		return
	}

	contentType := r.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	defer r.Body.Close()
	obj, err := h.objectService.UploadObject(r.Context(), bucketName, objectKey, r.Body, contentType)
	if err != nil {
		switch {
		case errors.Is(err, metadata.ErrBucketNotFound):
			h.writeError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, service.ErrEmptyObjectKey):
			h.writeError(w, http.StatusBadRequest, err.Error())
		default:
			h.writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	h.writeJSON(w, http.StatusOK, obj)
}

func (h *APIHandler) DownloadObject(w http.ResponseWriter, r *http.Request) {
	bucketName := chi.URLParam(r, "bucketName")
	objectKey := chi.URLParam(r, "*")

	if objectKey == "" {
		h.writeError(w, http.StatusBadRequest, "object key is required")
		return
	}

	obj, reader, err := h.objectService.DownloadObject(r.Context(), bucketName, objectKey)
	if err != nil {
		switch {
		case errors.Is(err, metadata.ErrObjectNotFound):
			h.writeError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, service.ErrEmptyObjectKey):
			h.writeError(w, http.StatusBadRequest, err.Error())
		default:
			h.writeError(w, http.StatusInternalServerError, "failed to retrieve object")
		}
		return
	}
	defer reader.Close()

	w.Header().Set("Content-Type", obj.ContentType)
	w.Header().Set("Content-Length", strconv.FormatInt(obj.SizeBytes, 10))
	w.WriteHeader(http.StatusOK)

	// Copy body stream
	// Note: We ignore copy errors at this stage because the headers have already been sent
	_, _ = io.Copy(w, reader)
}

func (h *APIHandler) ListObjects(w http.ResponseWriter, r *http.Request) {
	bucketName := chi.URLParam(r, "bucketName")

	objects, err := h.objectService.ListObjects(r.Context(), bucketName)
	if err != nil {
		switch {
		case errors.Is(err, metadata.ErrBucketNotFound):
			h.writeError(w, http.StatusNotFound, err.Error())
		default:
			h.writeError(w, http.StatusInternalServerError, "failed to list objects")
		}
		return
	}

	h.writeJSON(w, http.StatusOK, objects)
}

func (h *APIHandler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func (h *APIHandler) writeError(w http.ResponseWriter, status int, message string) {
	h.writeJSON(w, status, map[string]string{"error": message})
}
