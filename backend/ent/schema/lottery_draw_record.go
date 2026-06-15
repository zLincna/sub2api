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

// LotteryDrawRecord holds user lottery draw history.
type LotteryDrawRecord struct {
	ent.Schema
}

func (LotteryDrawRecord) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "lottery_draw_records"},
	}
}

func (LotteryDrawRecord) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("user_id"),
		field.Int64("chance_id"),
		field.Int64("prize_id"),
		field.String("prize_name").MaxLen(100),
		field.Float("amount").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}).
			Default(0),
		field.Float("balance_before").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}).
			Default(0),
		field.Float("balance_after").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}).
			Default(0),
		field.String("source_type").MaxLen(32),
		field.JSON("config_snapshot", map[string]any{}).
			Optional().
			SchemaType(map[string]string{dialect.Postgres: "jsonb"}),
		field.Time("created_at").
			Immutable().
			Default(time.Now).
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (LotteryDrawRecord) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("lottery_draw_records").
			Field("user_id").
			Unique().
			Required(),
		edge.From("chance", LotteryChance.Type).
			Ref("draw_records").
			Field("chance_id").
			Unique().
			Required(),
		edge.From("prize", LotteryPrize.Type).
			Ref("draw_records").
			Field("prize_id").
			Unique().
			Required(),
	}
}

func (LotteryDrawRecord) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id"),
		index.Fields("created_at"),
		index.Fields("source_type"),
	}
}
