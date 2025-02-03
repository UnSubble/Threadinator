package models

type Command struct {
	Command    string
	Args       []string
	Times      int
	Delay      *int
	Dependency *int
}
