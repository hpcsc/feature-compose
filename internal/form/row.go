package form

type rowKind int

const (
	nameKind rowKind = iota
	starterKind
	featureKind
)

type row struct {
	kind    rowKind
	feature string
}

func (r row) parent() row {
	if r.kind == featureKind {
		return row{kind: starterKind}
	}
	return r
}
