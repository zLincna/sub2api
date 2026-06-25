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

// CarpoolParticipant stores a user's paid seat in a carpool session.
type CarpoolParticipant struct {
	ent.Schema
}

func (CarpoolParticipant) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "carpool_participants"},
	}
}

func (CarpoolParticipant) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("session_id").
			Optional().
			Nillable(),
		field.Int64("vehicle_type_id"),
		field.Int64("user_id"),
		field.Int64("payment_order_id").
			Optional().
			Nillable(),
		field.String("status").MaxLen(32).Default("pending_payment"),
		field.Float("amount").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,2)"}).
			Default(0),
		field.Time("wait_until").
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.String("refund_method").MaxLen(32).Default("balance"),
		field.Int64("notice_version_id").
			Optional().
			Nillable(),
		field.Time("notice_accepted_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.String("notice_accept_ip").MaxLen(64).Default(""),
		field.Time("joined_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("paid_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("refunded_at").
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

func (CarpoolParticipant) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("session", CarpoolSession.Type).
			Ref("participants").
			Field("session_id").
			Unique(),
		edge.From("vehicle_type", CarpoolVehicleType.Type).
			Ref("participants").
			Field("vehicle_type_id").
			Unique().
			Required(),
		edge.From("user", User.Type).
			Ref("carpool_participants").
			Field("user_id").
			Unique().
			Required(),
		edge.From("payment_order", PaymentOrder.Type).
			Ref("carpool_participants").
			Field("payment_order_id").
			Unique(),
		edge.From("notice_version", CarpoolNoticeVersion.Type).
			Ref("participants").
			Field("notice_version_id").
			Unique(),
	}
}

func (CarpoolParticipant) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("session_id"),
		index.Fields("vehicle_type_id"),
		index.Fields("user_id"),
		index.Fields("payment_order_id"),
		index.Fields("status"),
		index.Fields("wait_until"),
	}
}
