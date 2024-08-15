package http

import (
	"github.com/paulkoehlerdev/OsmInTile/pkg/osmintile/application"
	"net/http"
)

func MapStyleRoute(mux *http.ServeMux, application application.Application) {
	mux.HandleFunc("GET /style.json", func(w http.ResponseWriter, req *http.Request) {
		style, err := application.GetMapStyle(req.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		_, err = w.Write(style)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})
}
