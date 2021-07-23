package main

import (
	"bytes"
	"context"
	"fmt"
	"go/format"
	"io/ioutil"
	"os"
	"strings"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
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

	stub := map[string]interface{}{
		"Name": "Becky",
		"Age":  12,
	}

	ctx := context.Background()
	rdb := redis.NewClient(&redis.Options{})
	pipe := rdb.Pipeline()
	id := uuid.NewString()
	// Add field entries
	for k, v := range stub {
		pipe.SAdd(ctx, GenKey(k, v), id)
	}
	pipe.HSet(ctx, GenKey("user", id), stub)
}

func GenKey(tags ...interface{}) string {
	n := len(tags)
	sl := []string{}
	for i := 0; i < n; i++ {
		sl = append(sl, "%v")
	}
	format := strings.Join(sl, ":")
	return fmt.Sprintf(format, tags...)
}
