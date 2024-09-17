package apiserver

import (
	"context"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/kbats183/simple-rtmp-restreamer/pkg/registry"
	"log"
	"net/http"
)

type webServer struct {
	registry registry.Registry
	router   *chi.Mux
	server   *http.Server
}

func NewWebServer(registry registry.Registry) *webServer {
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(loggerMiddleware())
	router.Use(middleware.Recoverer)

	streamRouter := newStreamsRouter(router, registry)
	streamRouter.Routes()
	router.Mount("/debug", middleware.Profiler())

	return &webServer{
		registry: registry,
		router:   router,
		server:   &http.Server{Addr: ":6070", Handler: router},
	}
}

func (a *webServer) Start(ctx context.Context) error {
	go func() {
		<-ctx.Done()
		if err := a.Stop(); err != nil {
			log.Printf("Error stopping web server: %v", err)
		}
	}()

	log.Println("Starting web server on :6070")
	if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (a *webServer) Stop() error {
	log.Println("Stopping web server")
	return a.server.Shutdown(context.Background())
}
