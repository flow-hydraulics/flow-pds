package http

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	gorilla "github.com/gorilla/handlers"
)

func UseCors(h http.Handler) http.Handler {
	return gorilla.CORS(gorilla.AllowedOrigins([]string{"*"}))(h)
}

func UseLogging(out io.Writer, h http.Handler) http.Handler {
	return gorilla.CombinedLoggingHandler(out, h)
}

func UseCompress(h http.Handler) http.Handler {
	return gorilla.CompressHandler(h)
}

func UseJson(h http.Handler) http.Handler {
	// Only PUT, POST, and PATCH requests are considered.
	return gorilla.ContentTypeHandler(h, "application/json")
}

// handleError is a helper function for unified HTTP error handling.
func handleError(rw http.ResponseWriter, logger *log.Logger, err error) {
	if logger != nil {
		logger.Printf("Error: %v\n", err)
	}

	// Check for "record not found" database error
	if strings.Contains(err.Error(), "record not found") {
		http.Error(rw, "record not found", http.StatusNotFound)
		return
	}

	http.Error(rw, err.Error(), http.StatusBadRequest)
}

// handleJsonResponse is a helper function for unified JSON response handling.
func handleJsonResponse(rw http.ResponseWriter, status int, res interface{}) {
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(status)
	json.NewEncoder(rw).Encode(res)
}

func checkNonEmptyBody(r *http.Request) error {
	if r.Body == nil || r.Body == http.NoBody {
		return fmt.Errorf("empty body")
	}
	return nil
}
