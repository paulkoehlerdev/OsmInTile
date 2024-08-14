package http

import (
	"github.com/paulkoehlerdev/OsmInTile/pkg/osmintile/application"
	"net/http"
	"strconv"
	"strings"
)

func MapTileRoute(mux *http.ServeMux, application application.Application) {
	mux.HandleFunc("GET /tiles/{level}/{z}/{x}/{y}", func(w http.ResponseWriter, req *http.Request) {
		levelStr := req.PathValue("level")
		zStr := req.PathValue("z")
		xStr := req.PathValue("x")
		yStr := req.PathValue("y")

		level, err := strconv.Atoi(levelStr)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		z, err := strconv.Atoi(zStr)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		x, err := strconv.Atoi(xStr)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		y, err := strconv.Atoi(yStr)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		encodings := req.Header.Get("Accept-Encoding")
		acceptGzip := strings.Contains(encodings, "gzip")

		tile, err := application.GetTile(req.Context(), level, uint32(x), uint32(y), uint32(z), acceptGzip)
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
