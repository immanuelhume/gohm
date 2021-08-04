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
// things like membership

func main() {
	Gen()

	// 	ctx := context.Background()
	// 	rdb := gohm.NewClient(&redis.Options{})
	// 	err := rdb.Anime.Save(ctx,
	// 		&playground.Anime{Title: "Mugen Train", Year: 2021, Rating: float64(100)})
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// 	as, err := rdb.Anime.FindMany(ctx,
	// 		&gohm.AnimeFindOpt{Title: gohm.String("Mugen Train")})
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// 	fmt.Print(as)
}

func Gen() {
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	td := CollectTemplateData(dir)
	b := &bytes.Buffer{}
	WritePackage("0-main.go.tmpl", "templates/*", b, td, map[string]interface{}{
		"toReceiverCase":  toReceiverCase,
		"lowerFirst":      lowerFirst,
		"toLower":         strings.ToLower,
		"newMarshallData": newMarshallData,
		"TStringifyField": TStringifyField,
	})
	src, err := format.Source(b.Bytes())
	if err != nil {
		panic(err)
	}
	ioutil.WriteFile("./local/generated.go", src, 0644)
}
