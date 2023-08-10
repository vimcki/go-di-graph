package deptree

type Dependency struct {
	Name string       `json:"name,omitempty"`
	Deps []Dependency `json:"deps,omitempty"`
	Desc string       `json:"desc,omitempty"`
}
