package fixtures

func helper(v bool) bool { return v }

// Example mixes assumptions and assertions so the analyser can be calibrated.
func Example(bla string, test bool) {
	if test && len(bla) > 0 {
		_ = bla
	} else if !test {
		_ = bla
	}

	for test {
		test = false
	}

	for i := 0; i != 0; i++ {
		_ = i
	}

	if bla == "hi" {
		_ = bla
	}

	if helper(test) {
		_ = bla
	}
}
