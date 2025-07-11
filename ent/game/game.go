// Code generated by ent, DO NOT EDIT.

package game

import (
	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
)

const (
	// Label holds the string label denoting the game type in the database.
	Label = "game"
	// FieldID holds the string denoting the id field in the database.
	FieldID = "id"
	// FieldName holds the string denoting the name field in the database.
	FieldName = "name"
	// FieldDescription holds the string denoting the description field in the database.
	FieldDescription = "description"
	// EdgeScores holds the string denoting the scores edge name in mutations.
	EdgeScores = "scores"
	// Table holds the table name of the game in the database.
	Table = "games"
	// ScoresTable is the table that holds the scores relation/edge.
	ScoresTable = "scores"
	// ScoresInverseTable is the table name for the Score entity.
	// It exists in this package in order to avoid circular dependency with the "score" package.
	ScoresInverseTable = "scores"
	// ScoresColumn is the table column denoting the scores relation/edge.
	ScoresColumn = "game_scores"
)

// Columns holds all SQL columns for game fields.
var Columns = []string{
	FieldID,
	FieldName,
	FieldDescription,
}

// ValidColumn reports if the column name is valid (part of the table columns).
func ValidColumn(column string) bool {
	for i := range Columns {
		if column == Columns[i] {
			return true
		}
	}
	return false
}

var (
	// NameValidator is a validator for the "name" field. It is called by the builders before save.
	NameValidator func(string) error
)

// OrderOption defines the ordering options for the Game queries.
type OrderOption func(*sql.Selector)

// ByID orders the results by the id field.
func ByID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldID, opts...).ToFunc()
}

// ByName orders the results by the name field.
func ByName(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldName, opts...).ToFunc()
}

// ByDescription orders the results by the description field.
func ByDescription(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldDescription, opts...).ToFunc()
}

// ByScoresCount orders the results by scores count.
func ByScoresCount(opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborsCount(s, newScoresStep(), opts...)
	}
}

// ByScores orders the results by scores terms.
func ByScores(term sql.OrderTerm, terms ...sql.OrderTerm) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newScoresStep(), append([]sql.OrderTerm{term}, terms...)...)
	}
}
func newScoresStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(ScoresInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.O2M, false, ScoresTable, ScoresColumn),
	)
}
