package ports

type OrderDirection string

const (
	OrderAsc  OrderDirection = "ASC"
	OrderDesc OrderDirection = "DESC"
)

type OrderBy struct {
	Column    string
	Direction OrderDirection
}

type ListOptions struct {
	OrderBy []OrderBy
	Limit   *int
	Offset  *int
}

type Operator string

const (
	OpEq  Operator = "="
	OpNeq Operator = "!="
	OpGt  Operator = ">"
	OpGte Operator = ">="
	OpLt  Operator = "<"
	OpLte Operator = "<="
	OpIn  Operator = "IN"
)

type Filter struct {
	Column string
	Op     Operator
	Value  any
}
