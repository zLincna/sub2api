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

// LotteryPrize holds the schema definition for lottery prizes.
type LotteryPrize struct {
	ent.Schema
}

func (LotteryPrize) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "lottery_prizes"},
	}
}

func (LotteryPrize) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").MaxLen(100).NotEmpty(),
		field.Float("amount").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}).
			Default(0),
		field.Float("probability").
			SchemaType(map[string]string{dialect.Postgres: "decimal(10,6)"}).
			Default(0),
		field.Int("daily_stock").Default(0),
		field.Int("daily_used").Default(0),
		field.Int("total_stock").Default(0),
		field.Int("total_used").Default(0),
		field.Bool("enabled").Default(true),
		field.String("color").MaxLen(32).Default("#f59e0b"),
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

func (LotteryPrize) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("draw_records", LotteryDrawRecord.Type),
	}
}

func (LotteryPrize) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("enabled"),
		index.Fields("sort_order"),
	}
}
