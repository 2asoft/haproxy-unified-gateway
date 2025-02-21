package main

type arguments struct {
	Name string
	Type string
}

type docItem struct {
	Name    string
	Comment string
	Args    []arguments
}
