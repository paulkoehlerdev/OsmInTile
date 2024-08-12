package main

import (
	"flag"
	"github.com/paulkoehlerdev/OsmInTile/pkg/osmintile/application"
	"github.com/paulkoehlerdev/OsmInTile/pkg/osmintile/domain/service"
	"github.com/paulkoehlerdev/OsmInTile/pkg/osmintile/interface/http"
	"net"
)

func main() {
	publicUrl := flag.String("public-url", "http://localhost:8080", "Public URL of OsmInTile")
	flag.Parse()

	listener, err := net.Listen("tcp", "0.0.0.0:8080")
	if err != nil {
		panic(err)
	}

	styleSvc := service.NewMapStyleService(*publicUrl)

	app := application.New(styleSvc)

	err = http.ServeApplication(listener, app)
	if err != nil {
		panic(err)
	}
}
