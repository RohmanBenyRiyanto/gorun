package mysql

import "reflect"

type Registry struct{}

func (Registry) GetSeederOrder() []string {
	return []string{"NotesSeeder"}
}

func (Registry) RegisterSeeder(typeName string) (reflect.Type, bool) {
	switch typeName {
	case "NotesSeeder":
		return reflect.TypeOf(NotesSeeder{}), true
	default:
		return nil, false
	}
}
