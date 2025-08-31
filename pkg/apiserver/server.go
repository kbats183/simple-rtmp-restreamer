package apiserver

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/kbats183/simple-rtmp-restreamer/pkg/registry"
	"log"
	"net/http"
	"path/filepath"
	"strings"
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
	router.Mount("/debug", middleware.Profiler())

	// Serve static files from web directory
	workDir, _ := filepath.Abs(".")
	filesDir := http.Dir(filepath.Join(workDir, "web"))
	FileServer(router, "/", filesDir)

	return &webServer{
		registry: registry,
		router:   router,
	}
}

func (a *webServer) Start() {
	log.Fatal(http.ListenAndServe(":6070", a.router)) //viper.GetString("server.port")
}

// FileServer conveniently sets up a http.FileServer handler to serve
// static files from a http.FileSystem.
func FileServer(r chi.Router, path string, root http.FileSystem) {
	if path != "/" && path[len(path)-1] != '/' {
		r.Get(path, http.RedirectHandler(path+"/", 301).ServeHTTP)
		path += "/"
	}
	path += "*"

	r.Get(path, func(w http.ResponseWriter, r *http.Request) {
		rctx := chi.RouteContext(r.Context())
		pathPrefix := strings.TrimSuffix(rctx.RoutePattern(), "/*")
		fs := http.StripPrefix(pathPrefix, http.FileServer(root))
		fs.ServeHTTP(w, r)
	})
}
