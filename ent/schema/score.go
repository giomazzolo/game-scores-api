package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type Score struct {
	ent.Schema
}

func (Score) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("value").
			Positive(),
		field.Time("created_at").
			Default(time.Now),
	}
}

func (Score) Edges() []ent.Edge {
	return []ent.Edge{
		// Creates the many-to-one relationship back to User.
		edge.From("user", User.Type).
			Ref("scores").
			Unique(). // A score must belong to exactly one user.
			Required(),
		// Creates the many-to-one relationship back to Game.
		edge.From("game", Game.Type).
			Ref("scores").
			Unique(). // A score must belong to exactly one game.
			Required(),
	}
}
