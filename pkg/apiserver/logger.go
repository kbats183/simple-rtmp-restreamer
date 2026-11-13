package apiserver

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
)

func loggerMiddleware() func(next http.Handler) http.Handler {
	return middleware.RequestLogger(&middleware.DefaultLogFormatter{Logger: log.Default(), NoColor: true})
}
