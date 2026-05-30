package fixtures

// Dog is a sample type used by the fixtures.
type Dog struct{}

// Woof does nothing useful.
func (d *Dog) Woof() {}

// CheckDog uses a negative nil check: this is an assumption. We assume that
// because dog is not nil it is a usable *Dog, instead of asserting it.
func CheckDog(dog *Dog) {
	if dog != nil {
		dog.Woof()
	}
}
