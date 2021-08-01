package main

import (
	"bytes"
	"go/format"
	"io/ioutil"
	"os"
	"strings"
)

// unsupported types: byte, rune
// TODO: support composite types
// TODO: add more pointer conversion functions

func main() {
	Gen()

	// ctx := context.Background()
	// rdb := gohm.NewClient(&redis.Options{})
	// err := rdb.Anime.Save(ctx,
	// 	&playground.Anime{Title: "Mugen Train", Year: 2021, Rating: float64(100)})
	// if err != nil {
	// 	panic(err)
	// }
	// as, err := rdb.Anime.FindMany(ctx,
	// 	&gohm.AnimeFilter{Title: gohm.String("Mugen Train")})
	// if err != nil {
	// 	panic(err)
	// }
	// fmt.Print(as)
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
