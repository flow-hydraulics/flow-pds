package http

import (
	"net/http"
	"net/http/pprof"

	"github.com/flow-hydraulics/flow-pds/service/app"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

func NewRouter(app *app.App) http.Handler {
	r := mux.NewRouter()

	requestLogger := log.New()

	// Catch the api version
	rv := r.PathPrefix("/{apiVersion}").Subrouter()

	rv.HandleFunc("/health/ready", HandleHealthReady()).Methods(http.MethodGet)

	rv.HandleFunc("/set-dist-cap", HandleSetDistCap(requestLogger, app)).Methods(http.MethodPost)

	rv.HandleFunc("/distributions", HandleCreateDistribution(requestLogger, app)).Methods(http.MethodPost)
	rv.HandleFunc("/distributions", HandleListDistributions(requestLogger, app)).Methods(http.MethodGet)
	rv.HandleFunc("/distributions/{id}", HandleGetDistribution(requestLogger, app)).Methods(http.MethodGet)
	rv.HandleFunc("/distributions/{id}/updatestate", HandleUpdateDistributionComplete(requestLogger, app)).Methods(http.MethodPatch)
	rv.HandleFunc("/distributions/{id}/abort", HandleAbortDistribution(requestLogger, app)).Methods(http.MethodPost)

	// Add the pprof routes
	rv.HandleFunc("/debug/pprof/", pprof.Index)
	rv.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	rv.HandleFunc("/debug/pprof/profile", pprof.Profile)
	rv.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	rv.HandleFunc("/debug/pprof/trace", pprof.Trace)

	rv.Handle("/debug/pprof/block", pprof.Handler("block"))
	rv.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
	rv.Handle("/debug/pprof/heap", pprof.Handler("heap"))
	rv.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))

	// Use middleware
	h := UseCors(r)
	h = UseLogging(requestLogger.Writer(), h)
	h = UseCompress(h)
	h = UseJson(h)

	return h
}
