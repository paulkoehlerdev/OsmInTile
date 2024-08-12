package main

import (
	"context"
	"flag"
	"github.com/paulkoehlerdev/OsmInTile/pkg/osmintile/application"
	"github.com/paulkoehlerdev/OsmInTile/pkg/osmintile/domain/service"
	"github.com/paulkoehlerdev/OsmInTile/pkg/osmintile/infrastructure"
	"github.com/paulkoehlerdev/OsmInTile/pkg/osmintile/interface/http"
	"log"
	"net"
)

func main() {
	publicUrl := flag.String("public-url", "http://localhost:8080", "Public URL of OsmInTile")
	databasePath := flag.String("database", ":memory:", "Database file path")
	osmFile := flag.String("osm-file", "", "Import OSM file")
	flag.Parse()

	listener, err := net.Listen("tcp", "0.0.0.0:8080")
	if err != nil {
		panic(err)
	}

	osmDataRepo, err := infrastructure.NewSqliteOsmDataRepository(*databasePath)
	if err != nil {
		panic(err)
	}

	if *osmFile != "" {
		log.Println("Loading osm file", *osmFile)
		err := osmDataRepo.Import(context.Background(), *osmFile)
		if err != nil {
			panic(err)
		}
	}

	styleSvc := service.NewMapStyleService(*publicUrl)

	tilesSvc := service.NewMapTilesService(osmDataRepo)

	app := application.New(styleSvc, tilesSvc)

	log.Println("Starting OsmInTile server")
	err = http.ServeApplication(listener, app)
	if err != nil {
		panic(err)
	}
}
