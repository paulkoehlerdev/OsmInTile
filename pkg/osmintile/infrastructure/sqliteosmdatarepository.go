package infrastructure

import (
	"compress/bzip2"
	"context"
	"database/sql"
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
	"strconv"
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
	conn                     *sql.DB
	getBasePreparedStatement *sql.Stmt
}

func (s *SqliteOsmDataRepository) init() (*SqliteOsmDataRepository, error) {
	err := s.prepareDatabase()
	if err != nil {
		return nil, fmt.Errorf("failed to prepare database: %w", err)
	}

	s.getBasePreparedStatement, err = s.conn.Prepare(`
		SELECT ST_AsBinary((
			SELECT BuildArea(MakeLine(geom))
			FROM node
			JOIN main.way_node wn on node.node_id = wn.node_id
			WHERE wn.way_id = way.way_id
		)) as geom
		FROM way
		WHERE way.way_id IN (SELECT way_tag.way_id FROM way_tag WHERE way_tag.key = 'indoor')
		  AND way.way_id IN (SELECT way_tag.way_id FROM way_tag WHERE way_tag.key = 'level' AND way_tag.value = ?)
		  AND (SELECT n.node_id FROM (SELECT way_node.node_id as node_id, MAX(way_node.sequence_id) FROM way_node WHERE way_node.way_id = way.way_id) as n) =
			  (SELECT n.node_id FROM (SELECT way_node.node_id as node_id, MIN(way_node.sequence_id) FROM way_node WHERE way_node.way_id = way.way_id) as n)
		  AND way.way_id IN (SELECT DISTINCT way_node.way_id FROM way_node JOIN node on way_node.node_id = node.node_id WHERE ST_Intersects(node.geom, ST_GeomFromWKB(?)));
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

func (s *SqliteOsmDataRepository) GetBase(ctx context.Context, level int, bound orb.Bound) (*geojson.FeatureCollection, error) {
	boundStr, err := wkb.MarshalToHex(bound.ToPolygon())
	if err != nil {
		return nil, fmt.Errorf("failed to marshal bound: %w", err)
	}

	rows, err := s.getBasePreparedStatement.QueryContext(ctx, strconv.Itoa(level), boundStr)
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

	includedObjects := make(map[osm.FeatureID]struct{})

	scanPasses := []func(osm.Scanner, map[osm.FeatureID]struct{}) error{
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

		// can be cast, as ObjectID == ElementID for OSM Elements
		elemID := osm.ElementID(obj.ObjectID()).FeatureID()

		if _, ok := includedObjects[elemID]; !ok {
			continue
		}

		delete(includedObjects, elemID)
		err := sqlimporter.importObject(obj)
		if err != nil {
			return fmt.Errorf("failed to import osm database object: %w", err)
		}

		count++
	}

	log.Printf("Imported %d objects. %d Objects not found", count, len(includedObjects))

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
func (s *SqliteOsmDataRepository) relationImportPass(scanner osm.Scanner, includedObjects map[osm.FeatureID]struct{}) error {
	includeRelation := func(relation *osm.Relation) {
		includedObjects[relation.FeatureID()] = struct{}{}
		for _, member := range relation.Members {
			includedObjects[member.FeatureID()] = struct{}{}
		}
	}

	for scanner.Scan() {
		obj := scanner.Object()
		relation, ok := obj.(*osm.Relation)
		if !ok {
			continue
		}

		// early return for passthrough
		if _, ok := includedObjects[relation.FeatureID()]; ok {
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
func (s *SqliteOsmDataRepository) wayImportPass(scanner osm.Scanner, includedObjects map[osm.FeatureID]struct{}) error {
	includeWay := func(way *osm.Way) {
		includedObjects[way.FeatureID()] = struct{}{}
		for _, node := range way.Nodes {
			includedObjects[node.FeatureID()] = struct{}{}
		}
	}

	for scanner.Scan() {
		obj := scanner.Object()

		way, ok := obj.(*osm.Way)
		if !ok {
			continue
		}

		// early return for passthrough
		if _, ok := includedObjects[way.FeatureID()]; ok {
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
func (s *SqliteOsmDataRepository) nodeImportPass(scanner osm.Scanner, includedObjects map[osm.FeatureID]struct{}) error {
	for scanner.Scan() {
		obj := scanner.Object()

		node, ok := obj.(*osm.Node)
		if !ok {
			continue
		}

		tags := node.TagMap()

		if _, ok := includedObjects[node.FeatureID()]; ok {
			includedObjects[node.FeatureID()] = struct{}{}
			continue
		}

		// node[~"amenity|shop|railway|highway|door|entrance"~"."]
		contains := false
		for _, v := range []string{"amenity", "shop", "railway", "highway", "door", "entrance"} {
			if _, ok := tags[v]; ok {
				contains = true
				break
			}
		}
		if contains {
			includedObjects[node.FeatureID()] = struct{}{}
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
