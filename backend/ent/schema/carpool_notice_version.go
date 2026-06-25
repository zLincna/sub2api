package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// CarpoolNoticeVersion stores versioned Markdown notices for carpool checkout.
type CarpoolNoticeVersion struct {
	ent.Schema
}

func (CarpoolNoticeVersion) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "carpool_notice_versions"},
	}
}

func (CarpoolNoticeVersion) Fields() []ent.Field {
	return []ent.Field{
		field.String("title").MaxLen(120).NotEmpty(),
		field.String("content_md").
			SchemaType(map[string]string{dialect.Postgres: "text"}).
			Default(""),
		field.Int("version").Default(1),
		field.Bool("active").Default(false),
		field.Time("published_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("created_at").
			Immutable().
			Default(time.Now).
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now).
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (CarpoolNoticeVersion) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("participants", CarpoolParticipant.Type),
	}
}

func (CarpoolNoticeVersion) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("active"),
		index.Fields("version"),
	}
}
