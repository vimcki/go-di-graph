package interpreter

type Dependency struct {
	Name         string
	Deps         []Dependency
	Flatten      bool
	Created      string
	ImportedFrom string
}
