package infrastructure

import (
	"database/sql"
	"fmt"
	"github.com/paulmach/osm"
)

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
