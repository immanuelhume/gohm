package main

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
)

// gohm
type User struct {
	Name string
}

// gohm
type Anime struct {
	Title string
}

func main() {
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	entities := CollectEntities(dir)
	td := &TemplateData{entities}
	b := &bytes.Buffer{}
	WritePackage("main.go.tmpl", b, td)
	src, err := format.Source(b.Bytes())
	if err != nil {
		panic(err)
	}
	fmt.Print(string(src))
}
