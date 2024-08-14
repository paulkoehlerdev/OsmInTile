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
	"io"
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
	err := s.prepareDatabase()
	if err != nil {
		return nil, fmt.Errorf("failed to prepare database: %w", err)
	}

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

func (s *SqliteOsmDataRepository) prepareDatabase() error {
	file, err := migrations.FS.ReadFile("schema.sql")
	if err != nil {
		return fmt.Errorf("failed to open schema file: %w", err)
	}

	for _, query := range strings.Split(string(file), ";") {
		if _, err := s.conn.Exec(query); err != nil {
			return fmt.Errorf("failed to execute schema file at query %s: %w", query, err)
		}
	}

	return nil
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

// Import imports the osm data file and filters unneeded content.
// Filters in Overpass notation: (inferred from https://openlevelup.net/ api requests)
// relation["indoor"]["indoor"!="yes"]
// relation["buildingpart"~"room|verticalpassage|corridor"]
// relation[~"amenity|shop|railway|highway|building:levels"~"."]
// way["indoor"]["indoor"!="yes"]
// way["buildingpart"~"room|verticalpassage|corridor"]
// way[~"amenity|shop|railway|highway|building:levels"~"."]
// node[~"amenity|shop|railway|highway|door|entrance"~"."]
func (s *SqliteOsmDataRepository) Import(ctx context.Context, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open osm dump file: %w", err)
	}
	defer f.Close()

	tx, err := s.conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to start osm database transaction: %w", err)
	}
	defer tx.Rollback()

	sqlimporter := sqliteosmobjectimporter{}
	err = sqlimporter.init(tx)
	if err != nil {
		return fmt.Errorf("failed to create sqliteimporter: %w", err)
	}

	includedObjects := make(map[osm.ObjectID]struct{})

	scanPasses := []func(osm.Scanner, map[osm.ObjectID]struct{}) error{
		s.relationImportPass,
		s.wayImportPass,
		s.nodeImportPass,
	}

	// apply filter passes
	for i, pass := range scanPasses {
		scanner, err := s.createImportScanner(ctx, f, path)
		if err != nil {
			return fmt.Errorf("failed to create import scanner: %w", err)
		}

		log.Printf("running pass (%d/%d)", i+1, len(scanPasses))

		err = pass(scanner, includedObjects)
		if err != nil {
			return fmt.Errorf("failed to import objects: %w", err)
		}

		log.Printf("finished import with %d objects (%d/%d)", len(includedObjects), i+1, len(scanPasses))
	}

	scanner, err := s.createImportScanner(ctx, f, path)
	if err != nil {
		return fmt.Errorf("failed to create import scanner: %w", err)
	}

	count := 0

	// apply insert pass
	for scanner.Scan() {
		obj := scanner.Object()
		if _, ok := includedObjects[obj.ObjectID()]; !ok {
			continue
		}

		err := sqlimporter.importObject(obj)
		if err != nil {
			return fmt.Errorf("failed to import osm database object: %w", err)
		}

		count++
	}

	log.Printf("Imported %d objects", count)

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to import osm dump file: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit osm database transaction: %w", err)
	}

	return nil
}

// relationImportPass puts the neccessary objectids for the following filters into the list of imports
// Filters in Overpass notation: (inferred from https://openlevelup.net/ api requests)
// relation["indoor"]["indoor"!="yes"]
// relation["buildingpart"~"room|verticalpassage|corridor"]
// relation[~"amenity|shop|railway|highway|building:levels"~"."]
func (s *SqliteOsmDataRepository) relationImportPass(scanner osm.Scanner, includedObjects map[osm.ObjectID]struct{}) error {
	includeRelation := func(relation *osm.Relation) {
		includedObjects[relation.ObjectID()] = struct{}{}
		for _, member := range relation.Members {
			includedObjects[member.ElementID().ObjectID()] = struct{}{}
		}
	}

	for scanner.Scan() {
		obj := scanner.Object()
		relation, ok := obj.(*osm.Relation)
		if !ok {
			continue
		}

		// early return for passthrough
		if _, ok := includedObjects[relation.ObjectID()]; ok {
			includeRelation(relation)
			continue
		}

		tags := relation.TagMap()

		// relation["indoor"]["indoor"!="yes"]
		if value, ok := tags["indoor"]; ok && value != "yes" {
			includeRelation(relation)
			continue
		}

		// relation["buildingpart"~"room|verticalpassage|corridor"]
		if value, ok := tags["buildingpart"]; ok &&
			(value == "room" || value == "verticalpassage" || value == "corridor") {
			includeRelation(relation)
			continue
		}

		// relation[~"amenity|shop|railway|highway|building:levels"~"."]
		contains := false
		for _, v := range []string{"amenity", "shop", "railway", "highway", "building:levels"} {
			if _, ok := tags[v]; ok {
				contains = true
				break
			}
		}
		if contains {
			includeRelation(relation)
			continue
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to import osm dump file: %w", err)
	}

	return nil
}

// wayImportPass puts the neccessary objectids for the following filters into the list of imports
// Filters in Overpass notation: (inferred from https://openlevelup.net/ api requests)
// way["indoor"]["indoor"!="yes"]
// way["buildingpart"~"room|verticalpassage|corridor"]
// way[~"amenity|shop|railway|highway|building:levels"~"."]
func (s *SqliteOsmDataRepository) wayImportPass(scanner osm.Scanner, includedObjects map[osm.ObjectID]struct{}) error {
	includeWay := func(way *osm.Way) {
		includedObjects[way.ObjectID()] = struct{}{}
		for _, node := range way.Nodes {
			includedObjects[node.ElementID().ObjectID()] = struct{}{}
		}
	}

	for scanner.Scan() {
		obj := scanner.Object()

		way, ok := obj.(*osm.Way)
		if !ok {
			continue
		}

		// early return for passthrough
		if _, ok := includedObjects[way.ObjectID()]; ok {
			includeWay(way)
			continue
		}

		tags := way.TagMap()

		// way["indoor"]["indoor"!="yes"]
		if value, ok := tags["indoor"]; ok && value != "yes" {
			includeWay(way)
			continue
		}

		// way["buildingpart"~"room|verticalpassage|corridor"]
		if value, ok := tags["buildingpart"]; ok &&
			(value == "room" || value == "verticalpassage" || value == "corridor") {
			includeWay(way)
			continue
		}

		// way[~"amenity|shop|railway|highway|building:levels"~"."]
		contains := false
		for _, v := range []string{"amenity", "shop", "railway", "highway", "building:levels"} {
			if _, ok := tags[v]; ok {
				contains = true
				break
			}
		}
		if contains {
			includeWay(way)
			continue
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to import osm dump file: %w", err)
	}

	return nil
}

// nodeImportPass puts the neccessary objectids for the following filters into the list of imports
// Filters in Overpass notation: (inferred from https://openlevelup.net/ api requests)
// node[~"amenity|shop|railway|highway|door|entrance"~"."]
func (s *SqliteOsmDataRepository) nodeImportPass(scanner osm.Scanner, includedObjects map[osm.ObjectID]struct{}) error {
	for scanner.Scan() {
		obj := scanner.Object()

		node, ok := obj.(*osm.Node)
		if !ok {
			continue
		}

		tags := node.TagMap()

		// node[~"amenity|shop|railway|highway|door|entrance"~"."]
		contains := false
		for _, v := range []string{"amenity", "shop", "railway", "highway", "door", "entrance"} {
			if _, ok := tags[v]; ok {
				contains = true
				break
			}
		}
		if contains {
			includedObjects[node.ObjectID()] = struct{}{}
			continue
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to import osm dump file: %w", err)
	}

	return nil
}

func (s *SqliteOsmDataRepository) createImportScanner(ctx context.Context, r io.ReadSeeker, path string) (osm.Scanner, error) {
	_, err := r.Seek(0, io.SeekStart)
	if err != nil {
		return nil, fmt.Errorf("failed to reset reader: %w", err)
	}

	var scanner osm.Scanner
	if strings.HasSuffix(path, ".osm.pbf") {
		scanner = osmpbf.New(ctx, r, runtime.GOMAXPROCS(-1))
	} else if strings.HasSuffix(path, ".osm.bz2") {
		compressedReader := bzip2.NewReader(r)
		scanner = osmxml.New(ctx, compressedReader)
	} else if strings.HasSuffix(path, ".osm") {
		scanner = osmxml.New(ctx, r)
	} else {
		return nil, fmt.Errorf("osm dump file must either be a '.osm'-XML, a '.osm.bz2'-compressed-XML or a '.osm.pbf'-protobuf file")
	}
	return scanner, nil
}
