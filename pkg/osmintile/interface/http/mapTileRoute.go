package http

import (
	"github.com/paulkoehlerdev/OsmInTile/pkg/osmintile/application"
	"net/http"
	"strconv"
	"strings"
)

func MapTileRoute(mux *http.ServeMux, application application.Application) {
	mux.HandleFunc("GET /tiles/{z}/{x}/{y}", func(w http.ResponseWriter, req *http.Request) {
		zStr := req.PathValue("z")
		xStr := req.PathValue("x")
		yStr := req.PathValue("y")

		z, err := strconv.Atoi(zStr)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}

		x, err := strconv.Atoi(xStr)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}

		y, err := strconv.Atoi(yStr)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}

		encodings := req.Header.Get("Accept-Encoding")
		acceptGzip := strings.Contains(encodings, "gzip")

		tile, err := application.GetTile(req.Context(), uint32(x), uint32(y), uint32(z), acceptGzip)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}

		w.Header().Set("Content-Type", "application/octet-stream")
		if acceptGzip {
			w.Header().Set("Content-Encoding", "gzip")
		}

		_, err = w.Write(tile)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})
}
