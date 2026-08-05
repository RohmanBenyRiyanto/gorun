package mysql

import "reflect"

// Registry is this package's gorun.SeederRegistry. It lives next to the
// seeder structs it registers, not in cmd/gorun-runner/main.go, so adding
// a new seeder is a change entirely within this package - a new
// <Name>Seeder.go file plus one more case here and one more entry in
// GetSeederOrder - rather than something that also has to be remembered
// in an unrelated main.go far away.
type Registry struct{}

// GetSeederOrder is the authoritative run order - see
// gorun.SeederRegistry's doc comment for what happens to a seeder found
// on disk but missing from this list (it still runs, just last, with a
// warning).
func (Registry) GetSeederOrder() []string {
	return []string{"ProductsSeeder"}
}

// RegisterSeeder maps a struct name gorun found on disk under this
// package's directory to the real Go type, so gorun can instantiate it.
func (Registry) RegisterSeeder(typeName string) (reflect.Type, bool) {
	switch typeName {
	case "ProductsSeeder":
		return reflect.TypeOf(ProductsSeeder{}), true
	default:
		return nil, false
	}
}
