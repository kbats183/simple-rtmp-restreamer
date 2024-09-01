package apiserver

import (
	"github.com/go-chi/chi/v5/middleware"
	"log"
	"net/http"
)

func loggerMiddleware() func(next http.Handler) http.Handler {
	return middleware.RequestLogger(&middleware.DefaultLogFormatter{Logger: log.Default(), NoColor: true})
}
