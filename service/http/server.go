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
	logger *log.Logger
}

func NewServer(cfg *config.Config, logger *log.Logger, app *app.App) *Server {
	if logger == nil {
		panic("no logger")
	}

	r := NewRouter(logger, app)

	// Server boilerplate
	srv := &http.Server{
		Handler:      r,
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		WriteTimeout: 15 * time.Minute,
		ReadTimeout:  15 * time.Minute,
	}

	return &Server{srv, cfg, logger}
}

func (s *Server) ListenAndServe() {
	// Run our server in a goroutine so that it doesn't block.
	go func() {
		s.logger.Printf("Server listening on %s:%d\n", s.cfg.Host, s.cfg.Port)
		s.logger.Print(s.Server.ListenAndServe())
	}()

	// Trap interupt or sigterm and gracefully shutdown the server
	c := make(chan os.Signal, 1)
	// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C)
	// SIGKILL, SIGQUIT or SIGTERM (Ctrl+/) will not be caught.
	signal.Notify(c, os.Interrupt)

	// Block until we receive our signal.
	sig := <-c

	s.logger.Printf("Got signal: %s. Shutting down..\n", sig)

	// Create a deadline to wait for.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	if err := s.Server.Shutdown(ctx); err != nil {
		s.logger.Fatal("Error in server shutdown; ", err)
	}
}
