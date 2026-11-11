package form

type rowKind int

const (
	nameKind rowKind = iota
	starterKind
	featureKind
	optionKind
)

type row struct {
	kind    rowKind
	feature string
	option  string
}
