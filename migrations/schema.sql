SELECT InitSpatialMetadata(1);
SELECT sqlite_version();
SELECT spatialite_version();

CREATE TABLE IF NOT EXISTS node
(
    node_id bigint NOT NULL PRIMARY KEY
);
SELECT AddGeometryColumn('node', 'geom', 4326, 'POINT');
CREATE INDEX IF NOT EXISTS node_node_id ON node (node_id);

CREATE TABLE IF NOT EXISTS node_tag
(
    node_id bigint NOT NULL,
    key     text   NOT NULL,
    value   text   NOT NULL
);
CREATE INDEX IF NOT EXISTS node_tag_node_id ON node_tag (node_id);

CREATE TABLE IF NOT EXISTS way
(
    way_id bigint NOT NULL PRIMARY KEY
);
CREATE INDEX IF NOT EXISTS way_way_id ON way (way_id);

CREATE TABLE IF NOT EXISTS way_tag
(
    way_id bigint NOT NULL,
    key    text   NOT NULL,
    value  text   NOT NULL
);
CREATE INDEX IF NOT EXISTS way_tag_way_id ON way_tag (way_id);

CREATE TABLE IF NOT EXISTS way_node
(
    way_id      bigint NOT NULL,
    node_id     bigint NOT NULL,
    sequence_id int    NOT NULL
);
CREATE INDEX IF NOT EXISTS way_node_way_id ON way_node (way_id);

CREATE TABLE IF NOT EXISTS relation
(
    relation_id bigint NOT NULL PRIMARY KEY
);
CREATE INDEX IF NOT EXISTS relation_relation_id ON relation (relation_id);

CREATE TABLE IF NOT EXISTS relation_tag
(
    relation_id bigint NOT NULL,
    key         text   NOT NULL,
    value       text   NOT NULL
);
CREATE INDEX IF NOT EXISTS relation_tag_relation_id ON relation_tag (relation_id);

CREATE TABLE IF NOT EXISTS relation_member
(
    relation_id bigint                                                                                 NOT NULL,
    member_type text CHECK ( member_type = 'way' OR member_type = 'node' OR member_type = 'relation' ) NOT NULL,
    member_id   bigint                                                                                 NOT NULL,
    sequence_id int                                                                                    NOT NULL
);
CREATE INDEX IF NOT EXISTS relation_member_relation_id ON relation_member (relation_id);