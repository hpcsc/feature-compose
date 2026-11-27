package catalog

type Steps interface {
	Start(label string) (end func(err error))
}

type NoSteps struct{}

func (NoSteps) Start(string) func(error) {
	return func(error) {}
}
