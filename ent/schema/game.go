package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type Game struct {
	ent.Schema
}

func (Game) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			Unique().
			NotEmpty(),
		field.Text("description").
			Optional(),
	}
}

func (Game) Edges() []ent.Edge {
	return []ent.Edge{
		// Defines the one-to-many relationship: one Game can have many Scores.
		edge.To("scores", Score.Type),
	}
}
