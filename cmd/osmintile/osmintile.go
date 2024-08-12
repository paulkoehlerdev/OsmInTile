package main

import (
	"github.com/paulkoehlerdev/OsmInTile/pkg/osmintile/interface/http"
	"net"
)

func main() {
	listener, err := net.Listen("tcp", "0.0.0.0:8080")
	if err != nil {
		panic(err)
	}

	err = http.ServeApplication(listener)
	if err != nil {
		panic(err)
	}
}
