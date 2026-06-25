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

// CarpoolVoucher stores admin-provided purchase/provisioning proof.
type CarpoolVoucher struct {
	ent.Schema
}

func (CarpoolVoucher) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "carpool_vouchers"},
	}
}

func (CarpoolVoucher) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("session_id"),
		field.String("file_url").
			SchemaType(map[string]string{dialect.Postgres: "text"}).
			Default(""),
		field.String("file_name").MaxLen(255).Default(""),
		field.String("description").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "text"}),
		field.Int64("uploaded_by").Default(0),
		field.Time("created_at").
			Immutable().
			Default(time.Now).
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (CarpoolVoucher) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("session", CarpoolSession.Type).
			Ref("vouchers").
			Field("session_id").
			Unique().
			Required(),
	}
}

func (CarpoolVoucher) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("session_id"),
		index.Fields("created_at"),
	}
}
