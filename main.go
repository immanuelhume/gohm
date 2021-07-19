package main

import (
	"fmt"
	"go/ast"
	"go/types"
	"os"

	"golang.org/x/tools/go/packages"
)

// An Entity contains raw syntax information for a gohm-tagged struct
type Entity struct {
	Pkg  *packages.Package
	Data *types.Struct
}

// Used to walk the AST tree
type Visitor struct {
	PkgIndex int
	Pkgs     []*packages.Package
	Entities []Entity
}

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
	// create entity
	en := Entity{Pkg: pkg, Data: tobj}
	v.Entities = append(v.Entities, en)
	return v
}

func main() {
	dir, err := os.Getwd()
	fmt.Println(dir)
	if err != nil {
		panic(err)
	}
	CollectEntities(dir)
}

func CollectEntities(dir string) []Entity {
	cfg := &packages.Config{Mode: packages.NeedSyntax | packages.NeedName |
		packages.NeedTypes | packages.NeedTypesInfo}
	pkgs, err := packages.Load(cfg, dir+"/...")
	if err != nil {
		panic(err)
	}
	v := &Visitor{Pkgs: pkgs, Entities: []Entity{}}
	for i, pkg := range pkgs {
		v.PkgIndex = i
		for _, file := range pkg.Syntax {
			ast.Walk(v, file)
		}
	}
	return v.Entities
}
