package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	gorilla "github.com/gorilla/handlers"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
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
		logger.Error(err)
	}

	// Check for "record not found" database error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		http.Error(rw, err.Error(), http.StatusNotFound)
		return
	}

	http.Error(rw, err.Error(), http.StatusBadRequest)
}

// handleJsonResponse is a helper function for unified JSON response handling.
func handleJsonResponse(rw http.ResponseWriter, status int, res interface{}) {
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(status)
	if err := json.NewEncoder(rw).Encode(res); err != nil {
		log.WithFields(log.Fields{"error": err}).Warn("error while encoding response to JSON")
	}
}

func checkNonEmptyBody(r *http.Request) error {
	if r.Body == nil || r.Body == http.NoBody {
		return fmt.Errorf("empty body")
	}
	return nil
}
