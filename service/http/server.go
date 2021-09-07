package http

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/flow-hydraulics/flow-pds/service/app"
	"github.com/flow-hydraulics/flow-pds/service/config"
)

type Server struct {
	cfg    *config.Config
	srv    *http.Server
	logger *log.Logger
}

func NewServer(cfg *config.Config, logger *log.Logger, app *app.App) *Server {
	if logger == nil {
		logger = log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile)
	}

	r := NewRouter(logger, app)

	// Server boilerplate
	srv := &http.Server{
		Handler:      r,
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	return &Server{cfg, srv, logger}
}

func (s *Server) ListenAndServe() {
	// Run our server in a goroutine so that it doesn't block.
	go func() {
		s.logger.Printf("Server listening on %s:%d\n", s.cfg.Host, s.cfg.Port)
		s.logger.Print(s.srv.ListenAndServe())
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

	if err := s.srv.Shutdown(ctx); err != nil {
		s.logger.Fatal("Error in server shutdown; ", err)
	}
}
