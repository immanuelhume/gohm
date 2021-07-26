package main

import (
	"bytes"
	"go/format"
	"io/ioutil"
	"os"
	"strings"
)

// TODO: marshall basic types
// TODO: check validity of types at generation time
// unsupported types: byte, rune

func main() {
	Gen()

	// ctx := context.Background()
	// rdb := gohm.NewClient(&redis.Options{})
	// err := rdb.User.Save(ctx, &playground.User{})
	// if err != nil {
	// 	panic(err)
	// }
}

func Gen() {
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	entities := CollectEntities(dir)
	td := &TemplateData{entities}
	b := &bytes.Buffer{}
	WritePackage("main.go.tmpl", "templates/*", b, td, map[string]interface{}{
		"toReceiverCase":  toReceiverCase,
		"toLower":         strings.ToLower,
		"newMarshallData": newMarshallData,
	})
	src, err := format.Source(b.Bytes())
	if err != nil {
		panic(err)
	}
	ioutil.WriteFile("./local/generated.go", src, 0644)
}
