package fixtures

// CheckCat asserts the concrete type instead of assuming non-nil, so the
// analyser finds no assumptions here. This is the "assertion" counterpart to
// CheckDog.
func CheckCat(cat any) {
	if c, ok := cat.(*Dog); ok {
		c.Woof()
	}
}
