package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type User struct {
	ent.Schema
}

func (User) Fields() []ent.Field {
	return []ent.Field{
		field.String("username").
			Unique().
			NotEmpty(),
		field.String("email").
			Unique().
			NotEmpty(),
		field.String("password_hash").
			NotEmpty().
			Sensitive(), // Prevents it from being exposed in logs
		field.Enum("role").
			Values("player", "admin").
			Default("player"),
	}
}

func (User) Edges() []ent.Edge {
	return []ent.Edge{
		// Defines the one-to-many relationship: one User can have many Scores.
		edge.To("scores", Score.Type),
	}
}
