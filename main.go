package main

import (
	"bytes"
	"context"
	"fmt"
	"go/format"
	"io/ioutil"
	"os"

	"github.com/go-redis/redis/v8"
	gohm "github.com/immanuelhume/gohm/local"
)

func main() {
	// Gen()

	ctx := context.Background()
	rdb := gohm.NewClient(&redis.Options{})
	// err := rdb.User.Create(ctx, &playground.User{Name: "Yo Mama", Age: 21})
	// if err != nil {
	// 	panic(err)
	// }
	users, err := rdb.User.FindMany(ctx, &gohm.UserFilter{Age: gohm.Int(20)})
	if err != nil {
		panic(err)
	}
	fmt.Print(users)
}

func Gen() {
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
	ioutil.WriteFile("./local/generated.go", src, 0644)
}
