package http

import (
	"github.com/paulkoehlerdev/OsmInTile/static"
	"net/http"
	"regexp"
)

var allowedFilesRegex = regexp.MustCompile("^.+\\.(js|css|html)$")

func WebPageRoute(mux *http.ServeMux) {
	fileServ := http.FileServerFS(static.FS)

	mux.HandleFunc("GET /", func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/" && !allowedFilesRegex.MatchString(req.URL.Path) {
			http.Error(w, "403 Forbidden", http.StatusForbidden)
			return
		}

		fileServ.ServeHTTP(w, req)
	})
}
