package callback

import (
	"context"
	"net/http"

	"github.com/go-chi/chi"
	"go.uber.org/zap"
)

// port is the default port where the server listens.
const port string = ":8080"

// Server represents a server instance.
type Server struct {
	codeChan   chan string
	httpServer *http.Server
}

// NewServer creates a new Server instance.
func NewServer(logger *zap.Logger, router chi.Router, codeCh chan string) *Server {
	router.Get("/auth", authCallbackHandler(codeCh))

	return &Server{
		codeChan: codeCh,
		httpServer: &http.Server{
			Addr:    port,
			Handler: router,
		},
	}
}

// Start starts the Server instance.
func (s *Server) Start() error {
	return s.httpServer.ListenAndServe()
}

// Stop stops the Server instance.
func (s *Server) Stop(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

// authCallbackHandler handles the callback from the authentication provider.
func authCallbackHandler(codeChan chan string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		codeChan <- r.URL.Query().Get("code")
		w.WriteHeader(http.StatusOK)
	}
}
