package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

type User struct {
	ent.Schema
}

func (User) Fields() []ent.Field {
	return []ent.Field{
		field.String("username").
			Unique().
			NotEmpty(),
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Immutable(),
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
