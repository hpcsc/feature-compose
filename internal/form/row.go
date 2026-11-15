package form

type rowKind int

const (
	nameKind rowKind = iota
	starterKind
	featuresKind
	optionKind
	writeKind
)

type row struct {
	kind    rowKind
	feature string
	option  string
}
