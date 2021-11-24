package http

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/flow-hydraulics/flow-pds/service/app"
	"github.com/flow-hydraulics/flow-pds/service/config"
	log "github.com/sirupsen/logrus"
)

type Server struct {
	Server *http.Server
	cfg    *config.Config
}

func NewServer(cfg *config.Config, app *app.App) *Server {

	r := NewRouter(app)

	// Server boilerplate
	srv := &http.Server{
		Handler:      r,
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		WriteTimeout: 15 * time.Minute,
		ReadTimeout:  15 * time.Minute,
	}

	return &Server{srv, cfg}
}

func (s *Server) ListenAndServe() {
	// Run our server in a goroutine so that it doesn't block.
	go func() {
		log.Infof("Server listening on %s:%d", s.cfg.Host, s.cfg.Port)
		log.Error(s.Server.ListenAndServe())
	}()

	// Trap interupt or sigterm and gracefully shutdown the server
	c := make(chan os.Signal, 1)
	// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C)
	// SIGKILL, SIGQUIT or SIGTERM (Ctrl+/) will not be caught.
	signal.Notify(c, os.Interrupt)

	// Block until we receive our signal.
	sig := <-c

	log.Infof("Got signal: %s. Shutting down...", sig)

	// Create a deadline to wait for.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	if err := s.Server.Shutdown(ctx); err != nil {
		log.Fatalf("Error in server shutdown: %s", err)
	}
}
