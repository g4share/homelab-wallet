package resolver

type cleartextResolver struct{}

// NewCleartextHandler creates a handler that returns values unchanged.
func NewCleartextHandler() InputHandler {
	return new(cleartextResolver)
}

func (*cleartextResolver) Match(value string) (string, bool) {
	return value, true
}

func (*cleartextResolver) Read(value string) (string, error) {
	return value, nil
}
