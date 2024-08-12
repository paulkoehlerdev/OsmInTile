package infrastructure

import (
	"compress/bzip2"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/paulkoehlerdev/OsmInTile/migrations"
	_ "github.com/paulkoehlerdev/OsmInTile/pkg/libraries/sqlitedriver"
	"github.com/paulkoehlerdev/OsmInTile/pkg/osmintile/domain/repository"
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/encoding/wkb"
	"github.com/paulmach/orb/geojson"
	"github.com/paulmach/osm"
	"github.com/paulmach/osm/osmpbf"
	"github.com/paulmach/osm/osmxml"
	"log"
	"os"
	"runtime"
	"strings"
)

var _ repository.OsmDataRepository = (*SqliteOsmDataRepository)(nil)

func NewSqliteOsmDataRepository(sqliteConnString string) (*SqliteOsmDataRepository, error) {
	sqlConn, err := sql.Open("sqlite3_custom", sqliteConnString)
	if err != nil {
		return nil, fmt.Errorf("failed to open osm database connection: %w", err)
	}

	return (&SqliteOsmDataRepository{
		conn: sqlConn,
	}).init()
}

type SqliteOsmDataRepository struct {
	conn                          *sql.DB
	getbuildingsPreparedStatement *sql.Stmt
}

func (s *SqliteOsmDataRepository) init() (*SqliteOsmDataRepository, error) {
	var err error

	s.getbuildingsPreparedStatement, err = s.conn.Prepare(`
		SELECT ST_AsBinary(ST_MakePolygon(ST_Collect(geom))) 
		FROM node 
		    JOIN way_node ON node.node_id = way_node.node_id
			JOIN way ON way.way_id = way_node.way_id
			JOIN way_tag ON way_tag.way_id = way_node.way_id
		WHERE way_tag.key = 'building' AND way_tag.value = 'yes'
		AND ST_Intersects(node.geom, ST_GeomFromWKB(?))
		GROUP BY way.way_id
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare statement: %w", err)
	}

	return s, nil
}

func (s *SqliteOsmDataRepository) GetBuildings(ctx context.Context, bound orb.Bound) (*geojson.FeatureCollection, error) {
	boundStr, err := wkb.MarshalToHex(bound.ToPolygon())
	if err != nil {
		return nil, fmt.Errorf("failed to marshal bound: %w", err)
	}

	rows, err := s.getbuildingsPreparedStatement.QueryContext(ctx, boundStr)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	return s.loadWBKRowsIntoGeojson(rows)
}

func (s *SqliteOsmDataRepository) loadWBKRowsIntoGeojson(rows *sql.Rows) (*geojson.FeatureCollection, error) {
	out := geojson.NewFeatureCollection()

	for rows.Next() {
		var wbkBytes []byte
		if err := rows.Scan(&wbkBytes); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		geom, err := wkb.Unmarshal(wbkBytes)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal geom: %w", err)
		}

		out.Append(geojson.NewFeature(geom))
	}

	d, _ := json.Marshal(out)
	log.Println(string(d))

	return out, nil
}

func (s *SqliteOsmDataRepository) Import(ctx context.Context, path string) error {
	file, err := migrations.FS.ReadFile("schema.sql")
	if err != nil {
		return fmt.Errorf("failed to open schema file: %w", err)
	}

	for _, query := range strings.Split(string(file), ";") {
		if _, err := s.conn.Exec(query); err != nil {
			return fmt.Errorf("failed to execute schema file at query %s: %w", query, err)
		}
	}

	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open osm dump file: %w", err)
	}
	defer f.Close()

	var scanner osm.Scanner
	if strings.HasSuffix(path, ".osm.pbf") {
		scanner = osmpbf.New(ctx, f, runtime.GOMAXPROCS(-1))
	} else if strings.HasSuffix(path, ".osm.bz2") {
		compressedReader := bzip2.NewReader(f)
		scanner = osmxml.New(ctx, compressedReader)
	} else if strings.HasSuffix(path, ".osm") {
		scanner = osmxml.New(ctx, f)
	} else {
		return fmt.Errorf("osm dump file must either be a '.osm'-XML, a '.osm.bz2'-compressed-XML or a '.osm.pbf'-protobuf file")
	}

	tx, err := s.conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to start osm database transaction: %w", err)
	}
	defer tx.Rollback()

	importer := sqliteosmobjectimporter{}
	err = importer.init(tx)
	if err != nil {
		return fmt.Errorf("failed to create sqliteimporter: %w", err)
	}

	for scanner.Scan() {
		err := importer.importObject(scanner.Object())
		if err != nil {
			return fmt.Errorf("failed to import osm database object: %w", err)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to import osm dump file: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit osm database transaction: %w", err)
	}

	return nil
}
