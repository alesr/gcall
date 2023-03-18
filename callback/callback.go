package callback

import (
	"net/http"

	"github.com/go-chi/chi"
	"go.uber.org/zap"
)

const port string = ":8080"

type Server struct {
	codeChan   chan string
	httpServer *http.Server
}

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

func (s *Server) Start() error {
	return s.httpServer.ListenAndServe()
}

func (s *Server) Stop() error {
	return s.httpServer.Close()
}

func authCallbackHandler(codeChan chan string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		codeChan <- r.URL.Query().Get("code")
		w.WriteHeader(http.StatusOK)
	}
}
