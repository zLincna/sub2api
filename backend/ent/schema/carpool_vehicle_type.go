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

// CarpoolVehicleType stores front-facing carpool queue configuration.
type CarpoolVehicleType struct {
	ent.Schema
}

func (CarpoolVehicleType) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "carpool_vehicle_types"},
	}
}

func (CarpoolVehicleType) Fields() []ent.Field {
	return []ent.Field{
		field.String("product").MaxLen(32).Default("openai"),
		field.String("plan_tier").MaxLen(32).Default("pro"),
		field.String("multiplier").MaxLen(32).Default("20x"),
		field.String("name").MaxLen(100).NotEmpty(),
		field.Int("seat_count").Default(2),
		field.Float("total_price").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,2)"}).
			Default(0),
		field.Float("unit_price").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,2)"}).
			Default(0),
		field.Int("service_days").Default(30),
		field.Int("refund_wait_hours").Default(2),
		field.Int("completed_base_count").Default(0),
		field.Bool("enabled").Default(false),
		field.Bool("support_revenue_pool").Default(false),
		field.Bool("require_static_ip").Default(true),
		field.JSON("wait_duration_options", []int{2, 6, 12, 24}).
			Optional().
			SchemaType(map[string]string{dialect.Postgres: "jsonb"}),
		field.JSON("refund_methods", []string{"balance", "gateway"}).
			Optional().
			SchemaType(map[string]string{dialect.Postgres: "jsonb"}),
		field.String("description").
			SchemaType(map[string]string{dialect.Postgres: "text"}).
			Default(""),
		field.Int("sort_order").Default(0),
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

func (CarpoolVehicleType) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("sessions", CarpoolSession.Type),
		edge.To("participants", CarpoolParticipant.Type),
	}
}

func (CarpoolVehicleType) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("enabled"),
		index.Fields("sort_order"),
		index.Fields("seat_count"),
	}
}
