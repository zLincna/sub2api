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

// LotteryChance holds granted lottery chances.
type LotteryChance struct {
	ent.Schema
}

func (LotteryChance) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "lottery_chances"},
	}
}

func (LotteryChance) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("user_id"),
		field.String("source_type").MaxLen(32),
		field.String("source_key").MaxLen(128),
		field.Int("total_count").Default(0),
		field.Int("used_count").Default(0),
		field.Float("source_amount").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}).
			Default(0),
		field.Time("grant_date").
			SchemaType(map[string]string{dialect.Postgres: "date"}),
		field.Time("expires_at").
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.String("notes").
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

func (LotteryChance) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("lottery_chances").
			Field("user_id").
			Unique().
			Required(),
		edge.To("draw_records", LotteryDrawRecord.Type),
	}
}

func (LotteryChance) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id"),
		index.Fields("source_type"),
		index.Fields("expires_at"),
		index.Fields("grant_date"),
		index.Fields("user_id", "source_type", "source_key").Unique(),
	}
}
