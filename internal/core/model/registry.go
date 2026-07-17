package model

type Registry interface {
	Find(alias string) (*Model, error)

	Mappings(modelID string) ([]Mapping, error)

	List() []Model
}
