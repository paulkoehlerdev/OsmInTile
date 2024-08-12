package http

import (
	"net"
	"net/http"
)

func ServeApplication(l net.Listener) error {
	mux := http.NewServeMux()
	WebPageRoute(mux)

	return http.Serve(l, mux)
}
