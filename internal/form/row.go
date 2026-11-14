package form

type rowKind int

const (
	nameKind rowKind = iota
	starterKind
	groupKind
	featureKind
	optionKind
)

type group string

const (
	optionsGroup  group = "options"
	featuresGroup group = "features"
)

type row struct {
	kind    rowKind
	group   group
	feature string
	option  string
}
