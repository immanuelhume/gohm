package main

import (
	"bytes"
	"go/format"
	"io/ioutil"
	"os"
)

// gohm
type User struct {
	Name string
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
	ioutil.WriteFile("./local/gohm.go", src, 0644)
	// rdb := redis.NewClient(&redis.Options{})
	// rdb.hs
}
