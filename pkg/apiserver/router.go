package apiserver

import (
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"github.com/kbats183/simple-rtmp-restreamer/pkg/registry"
	"log"
	"net/http"
	"net/url"
	"strings"
)

type streamRouter struct {
	r        chi.Router
	registry registry.Registry
}

func newStreamsRouter(router chi.Router, registry registry.Registry) *streamRouter {
	return &streamRouter{
		r:        router,
		registry: registry,
	}
}

func (router *streamRouter) Routes() {
	router.r.Route("/api/streams", func(r chi.Router) {
		r.Use(ContentTypeJson)
		r.Get("/", router.getStreams())
		r.Get("/{id}", router.getStreamById())
		r.Post("/", router.createStream())
		r.Delete("/{id}", router.deleteBankByID())
		r.Get("/{id}/status", router.getStreamStatusById())
		r.Post("/{id}/targets", router.addStreamTargetByStreamId())
	})
}

func (router *streamRouter) getStreams() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		streams, err := router.registry.GetStreams()
		if err != nil {
			handleErrors(w, err)
			return
		}
		objects := make([]*Stream, len(streams))
		for i, stream := range streams {
			objects[i] = streamFromRegistryObject(stream)
		}
		if err := json.NewEncoder(w).Encode(objects); err != nil {
			handleErrors(w, err)
			return
		}
	}

}

func (router *streamRouter) getStreamById() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		stream, err := router.registry.GetStream(id)
		if err != nil {
			handleErrors(w, err)
			return
		}
		if err := json.NewEncoder(w).Encode(streamFromRegistryObject(stream)); err != nil {
			handleErrors(w, err)
			return
		}
	}

}

func (router *streamRouter) createStream() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var stream Stream
		if err := json.NewDecoder(r.Body).Decode(&stream); err != nil {
			handleErrors(w, err)
			return
		}
		st, err := stream.toRegistryObject()
		if err != nil {
			handleErrors(w, err)
			return
		}

		err = router.registry.Update(st)
		if err != nil {
			handleErrors(w, err)
			return
		}
	}

}

func (router *streamRouter) deleteBankByID() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := router.registry.DeleteStream(chi.URLParam(r, "id"))
		if err != nil {
			handleErrors(w, err)
			return
		}
	}

}

func (router *streamRouter) getStreamStatusById() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status, err := router.registry.GetStatus(chi.URLParam(r, "id"))
		if err != nil {
			handleErrors(w, err)
			return
		}
		if err := json.NewEncoder(w).Encode(status); err != nil {
			handleErrors(w, err)
			return
		}
	}
}

func (router *streamRouter) addStreamTargetByStreamId() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var targetInfo AddTargetInfo
		if err := json.NewDecoder(r.Body).Decode(&targetInfo); err != nil {
			handleErrors(w, err)
			return
		}
		target, err := url.Parse(targetInfo.Target)
		if err != nil {
			handleErrors(w, err)
			return
		}

		err = router.registry.AddStreamTarget(chi.URLParam(r, "id"), (*registry.PushTargetUrl)(target))
		if err != nil {
			handleErrors(w, err)
			return
		}
	}
}

// ErrorResponse represents json error structure
type ErrorResponse struct {
	Error string `json:"error"`
}

func JSONError(w http.ResponseWriter, error string, code int) {
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(ErrorResponse{error}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func handleErrors(w http.ResponseWriter, err error) {
	const logFormat = "fatal: %+v\n"
	if strings.Contains(err.Error(), "connection refused") {
		log.Printf(logFormat, err)
		JSONError(w, "DB_CONNECTION_FAIL", http.StatusServiceUnavailable)
		return
	}
	if err.Error() == http.StatusText(400) {
		log.Printf(logFormat, err)
		JSONError(w, err.Error(), http.StatusBadRequest)
		return
	}
	switch err.(type) {
	case registry.StreamNotFound:
		JSONError(w, err.Error(), http.StatusNotFound)
	default:
		log.Printf(logFormat, err)
		JSONError(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
	return
}
