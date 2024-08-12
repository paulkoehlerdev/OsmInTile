package http

import (
	"github.com/paulkoehlerdev/OsmInTile/pkg/osmintile/application"
	"net"
	"net/http"
)

func ServeApplication(l net.Listener, application application.Application) error {
	mux := http.NewServeMux()
	WebPageRoute(mux)
	MapStyleRoute(mux, application)
	MapTileRoute(mux, application)

	return http.Serve(l, mux)
}
