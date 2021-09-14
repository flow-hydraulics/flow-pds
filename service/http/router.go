package http

import (
	"log"
	"net/http"
	"os"

	"github.com/flow-hydraulics/flow-pds/service/app"
	"github.com/gorilla/mux"
)

func NewRouter(logger *log.Logger, app *app.App) http.Handler {
	r := mux.NewRouter()

	// Catch the api version
	rv := r.PathPrefix("/{apiVersion}").Subrouter()

	rv.HandleFunc("/distributions", HandleCreateDistribution(logger, app)).Methods(http.MethodPost)
	rv.HandleFunc("/distributions", HandleListDistributions(logger, app)).Methods(http.MethodGet)
	rv.HandleFunc("/distributions/{id}", HandleGetDistribution(logger, app)).Methods(http.MethodGet)
	rv.HandleFunc("/distributions/{id}", HandleCancelDistribution(logger, app)).Methods(http.MethodDelete)

	// Use middleware
	h := UseCors(r)
	h = UseLogging(os.Stdout, h)
	h = UseCompress(h)
	h = UseJson(h)

	return h
}
