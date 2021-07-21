package main

import (
	"bytes"
	"fmt"
	"os"
)

// gohm
type User struct {
	Name string
}

func main() {
	dir, err := os.Getwd()
	fmt.Println(dir)
	if err != nil {
		panic(err)
	}
	entities := CollectEntities(dir)
	td := &TemplateData{entities}
	b := &bytes.Buffer{}
	WritePackage("main.go.tmpl", b, td)
	fmt.Print(b.String())
}
