package catalog

type delimiters struct {
	left  string
	right string
}

var defaultDelimiters = delimiters{left: "{{", right: "}}"}
