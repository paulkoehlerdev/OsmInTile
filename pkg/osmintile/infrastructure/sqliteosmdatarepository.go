package infrastructure

import (
	"compress/bzip2"
	"context"
	"database/sql"
	"fmt"
	"github.com/paulkoehlerdev/OsmInTile/migrations"
	_ "github.com/paulkoehlerdev/OsmInTile/pkg/libraries/sqlitedriver"
	"github.com/paulkoehlerdev/OsmInTile/pkg/osmintile/domain/repository"
	"github.com/paulmach/osm"
	"github.com/paulmach/osm/osmpbf"
	"github.com/paulmach/osm/osmxml"
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
	conn *sql.DB
}

func (s *SqliteOsmDataRepository) init() (*SqliteOsmDataRepository, error) {
	file, err := migrations.FS.ReadFile("schema.sql")
	if err != nil {
		return nil, fmt.Errorf("failed to open schema file: %w", err)
	}

	for _, query := range strings.Split(string(file), ";") {
		if _, err := s.conn.Exec(query); err != nil {
			return nil, fmt.Errorf("failed to execute schema file at query %s: %w", query, err)
		}
	}

	return s, nil
}

func (s *SqliteOsmDataRepository) Import(ctx context.Context, path string) error {
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

type sqliteosmobjectimporter struct {
	insertNodePreparedStatement           *sql.Stmt
	insertNodeTagPreparedStatement        *sql.Stmt
	insertWayPreparedStatement            *sql.Stmt
	insertWayTagPreparedStatement         *sql.Stmt
	insertWayNodePreparedStatement        *sql.Stmt
	insertRelationPreparedStatement       *sql.Stmt
	insertRelationTagPreparedStatement    *sql.Stmt
	insertRelationMemberPreparedStatement *sql.Stmt
}

func (s *sqliteosmobjectimporter) init(tx *sql.Tx) error {
	if err := s.prepareStatements(tx); err != nil {
		return fmt.Errorf("failed to prepare statements: %w", err)
	}

	return nil
}

func (s *sqliteosmobjectimporter) importObject(obj osm.Object) error {
	if node, ok := obj.(*osm.Node); ok {
		return s.importNode(node)
	}

	if way, ok := obj.(*osm.Way); ok {
		return s.importWay(way)
	}

	if relation, ok := obj.(*osm.Relation); ok {
		return s.importRelation(relation)
	}

	return fmt.Errorf("unexpected object type %T", obj)
}

func (s *sqliteosmobjectimporter) prepareStatements(tx *sql.Tx) error {
	var err error

	s.insertNodePreparedStatement, err = tx.Prepare(
		"INSERT OR REPLACE INTO node (node_id, geom) VALUES (?, SetSRID(MakePoint(?, ?), 4326))",
	)
	if err != nil {
		return err
	}

	s.insertNodeTagPreparedStatement, err = tx.Prepare(
		"INSERT OR REPLACE INTO node_tag (node_id, key, value) VALUES (?, ?, ?)",
	)
	if err != nil {
		return err
	}

	s.insertWayPreparedStatement, err = tx.Prepare(
		"INSERT OR REPLACE INTO way (way_id) VALUES (?)",
	)
	if err != nil {
		return err
	}

	s.insertWayTagPreparedStatement, err = tx.Prepare(
		"INSERT OR REPLACE INTO way_tag (way_id, key, value) VALUES (?, ?, ?)",
	)
	if err != nil {
		return err
	}

	s.insertWayNodePreparedStatement, err = tx.Prepare(
		"INSERT OR REPLACE INTO way_node (way_id, node_id, sequence_id) VALUES (?, ?, ?)",
	)
	if err != nil {
		return err
	}

	s.insertRelationPreparedStatement, err = tx.Prepare(
		"INSERT OR REPLACE INTO relation (relation_id) VALUES (?)",
	)
	if err != nil {
		return err
	}

	s.insertRelationTagPreparedStatement, err = tx.Prepare(
		"INSERT OR REPLACE INTO relation_tag (relation_id, key, value) VALUES (?, ?, ?)",
	)
	if err != nil {
		return err
	}

	s.insertRelationMemberPreparedStatement, err = tx.Prepare(
		"INSERT OR REPLACE INTO relation_member (relation_id, member_type, member_id, sequence_id) VALUES (?, ?, ?, ?)",
	)
	if err != nil {
		return err
	}

	return nil
}

func (s *sqliteosmobjectimporter) importNode(node *osm.Node) error {
	_, err := s.insertNodePreparedStatement.Exec(node.ID, node.Lon, node.Lat)
	if err != nil {
		return fmt.Errorf("failed to insert node: %w", err)
	}

	for key, value := range node.TagMap() {
		_, err := s.insertNodeTagPreparedStatement.Exec(node.ID, key, value)
		if err != nil {
			return fmt.Errorf("failed to insert node_tag: %w", err)
		}
	}

	return nil
}

func (s *sqliteosmobjectimporter) importWay(way *osm.Way) error {
	_, err := s.insertWayPreparedStatement.Exec(way.ID)
	if err != nil {
		return fmt.Errorf("failed to insert way: %w", err)
	}

	for key, value := range way.TagMap() {
		_, err := s.insertWayTagPreparedStatement.Exec(way.ID, key, value)
		if err != nil {
			return fmt.Errorf("failed to insert way_tag: %w", err)
		}
	}

	for sequenceID, node := range way.Nodes {
		_, err := s.insertWayNodePreparedStatement.Exec(way.ID, node.ID, sequenceID)
		if err != nil {
			return fmt.Errorf("failed to insert way_node: %w", err)
		}
	}

	return nil
}

func (s *sqliteosmobjectimporter) importRelation(relation *osm.Relation) error {

	_, err := s.insertRelationPreparedStatement.Exec(relation.ID)
	if err != nil {
		return fmt.Errorf("failed to insert relation: %w", err)
	}

	for key, value := range relation.TagMap() {
		_, err := s.insertRelationTagPreparedStatement.Exec(relation.ID, key, value)
		if err != nil {
			return fmt.Errorf("failed to insert relation_tag: %w", err)
		}
	}

	for sequenceID, member := range relation.Members {
		_, err := s.insertRelationMemberPreparedStatement.Exec(relation.ID, member.Type, member.ElementID(), sequenceID)
		if err != nil {
			return fmt.Errorf("failed to insert relation_member: %w", err)
		}
	}

	return nil
}

func (s *SqliteOsmDataRepository) GetNode(ctx context.Context, id int64) (*osm.Node, error) {
	//TODO implement me
	panic("implement me")
}

func (s *SqliteOsmDataRepository) GetWay(ctx context.Context, id int64) (*osm.Way, error) {
	//TODO implement me
	panic("implement me")
}

func (s *SqliteOsmDataRepository) GetRelation(ctx context.Context, id int64) (*osm.Relation, error) {
	//TODO implement me
	panic("implement me")
}
