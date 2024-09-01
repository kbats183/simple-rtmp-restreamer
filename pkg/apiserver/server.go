package apiserver

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/kbats183/simple-rtmp-restreamer/pkg/registry"
	"log"
	"net/http"
)

type webServer struct {
	registry registry.Registry
	router   *chi.Mux
}

func NewWebServer(registry registry.Registry) *webServer {
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(loggerMiddleware())
	router.Use(middleware.Recoverer)

	streamRouter := newStreamsRouter(router, registry)
	streamRouter.Routes()

	return &webServer{
		registry: registry,
		router:   router,
	}
}

func (a *webServer) Start() {
	log.Fatal(http.ListenAndServe(":6070", a.router)) //viper.GetString("server.port")
}
