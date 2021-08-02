package main

import (
	"fmt"
	"go/ast"
	"go/types"
	"log"

	"golang.org/x/tools/go/packages"
)

// Visitor to walk the AST tree.
type Visitor struct {
	PkgIndex int
	Pkgs     []*packages.Package
	Models   []Model
}

// Visit will add nodes which are a struct type tagged with the // gohm
// comment. All other nodes are ignored.
func (v *Visitor) Visit(node ast.Node) ast.Visitor {
	// cast to GenDecl
	gd, ok := node.(*ast.GenDecl)
	if !ok {
		return v
	}
	// check for gohm tag
	if gd.Doc == nil {
		return v
	}
	var hasGohm bool
	for _, doc := range gd.Doc.List {
		if doc.Text == "// gohm" {
			hasGohm = true
		}
	}
	if !hasGohm {
		return v
	}
	// cast to TypeSpec
	spec, ok := gd.Specs[0].(*ast.TypeSpec)
	if !ok {
		return v
	}
	// cast to Struct
	pkg := v.Pkgs[v.PkgIndex]
	tobj, ok := pkg.TypesInfo.Defs[spec.Name].Type().Underlying().(*types.Struct)
	if !ok {
		return v
	}
	// check if it's in package main
	if pkg.Name == "main" {
		log.Fatalf(`Cannot read from package main (type %s in %s).
Please place the model in a package that isn't main.`,
			spec.Name.Name, pkg.PkgPath)
	}
	// create model and check each field's type
	m := Model{Pkg: pkg, Name: spec.Name, Fields: tobj}
	m.ValidateFieldType()

	v.Models = append(v.Models, m)
	return v
}

// CollectModels takes a root directory and returns all models found
// in all packages.
func CollectModels(dir string) []Model {
	// TODO: check for re-declarations
	cfg := &packages.Config{Mode: packages.NeedSyntax | packages.NeedName |
		packages.NeedTypes | packages.NeedTypesInfo | packages.NeedModule}
	pkgs, err := packages.Load(cfg, dir+"/...")
	if err != nil {
		panic(err)
	}
	v := &Visitor{Pkgs: pkgs, Models: []Model{}}
	for i, pkg := range pkgs {
		v.PkgIndex = i
		for _, file := range pkg.Syntax {
			ast.Walk(v, file)
		}
	}
	return v.Models
}

// These types are not supported. They cannot be sensibly marshalled into a string
// and stored in Redis.
var badTypes = map[interface{}]bool{}

// Checks if a struct contains any unsupported data types. Exits with error if found.
func (m *Model) ValidateFieldType() {
	for i := 0; i < m.Fields.NumFields(); i++ {
		t := m.Fields.Field(i).Type().Underlying()
		switch t := t.(type) {
		case *types.Basic:
			_, got := badTypes[t.Kind()]
			if got {
				log.Fatalf("Cannot use field of type %s in type %s (%s)",
					t, m.Name.Name, m.Pkg.PkgPath)
			}
		case *types.Array:
			fmt.Print(t.Elem())
		}
	}
}
