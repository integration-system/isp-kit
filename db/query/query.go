package query

import "github.com/Masterminds/squirrel"

type Query squirrel.StatementBuilderType

func New() Query {
	return Query(squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar))
}
