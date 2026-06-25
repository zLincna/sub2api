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

// CarpoolSession stores one concrete carpool round for a vehicle type.
type CarpoolSession struct {
	ent.Schema
}

func (CarpoolSession) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "carpool_sessions"},
	}
}

func (CarpoolSession) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("vehicle_type_id"),
		field.String("session_no").MaxLen(64).Default(""),
		field.String("status").MaxLen(32).Default("recruiting"),
		field.Int("seat_count").Default(2),
		field.Int("paid_count").Default(0),
		field.Time("started_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("filled_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("provisioned_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("service_started_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("service_ended_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.JSON("account_info", map[string]any{}).
			Optional().
			SchemaType(map[string]string{dialect.Postgres: "jsonb"}),
		field.JSON("proxy_info", map[string]any{}).
			Optional().
			SchemaType(map[string]string{dialect.Postgres: "jsonb"}),
		field.JSON("communication", map[string]any{}).
			Optional().
			SchemaType(map[string]string{dialect.Postgres: "jsonb"}),
		field.String("admin_notes").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "text"}),
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

func (CarpoolSession) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("vehicle_type", CarpoolVehicleType.Type).
			Ref("sessions").
			Field("vehicle_type_id").
			Unique().
			Required(),
		edge.To("participants", CarpoolParticipant.Type),
		edge.To("vouchers", CarpoolVoucher.Type),
	}
}

func (CarpoolSession) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("vehicle_type_id"),
		index.Fields("status"),
		index.Fields("created_at"),
		index.Fields("vehicle_type_id", "status"),
	}
}
