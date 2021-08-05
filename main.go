package main

import (
	"bytes"
	"go/format"
	"io/ioutil"
	"os"
	"strings"
)

// TODO: support composite types
// TODO: add more pointer conversion functions

// NOTE: findOpt must be different for composite types - for basic types we
// just check direct matching, but slices, for instance, should support
// things like membership.

func main() {
	var dir string
	var b *bytes.Buffer

	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	b = &bytes.Buffer{}
	Codegen(dir, b, "./local/generated.go")
}

// Codegen is the main function which writes the generated code.
func Codegen(dir string, b *bytes.Buffer, out string) {
	td := CollectTemplateData(dir)
	WritePackage("00-main.go.tmpl", "templates/*", b, td, map[string]interface{}{
		"toReceiverCase":  toReceiverCase,
		"lowerFirst":      lowerFirst,
		"toLower":         strings.ToLower,
		"TStringifyField": TStringifyField,
		"TParseField":     TParseField,
	})
	src, err := format.Source(b.Bytes())
	if err != nil {
		panic(err)
	}
	ioutil.WriteFile(out, src, 0644)
}
