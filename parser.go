package main

import (
	"fmt"
	"go/ast"
	"go/types"
	"io"
	"strings"
	"text/template"

	"golang.org/x/tools/go/packages"
)

// An Entity contains raw syntax information for a gohm-tagged struct.
type Entity struct {
	// package in which the struct is declared
	Pkg *packages.Package

	// name of this entity
	Name *ast.Ident

	// contains info on fields
	Fields *types.Struct
}

// Shape of the data we're gonna pass into our template file.
type TemplateData struct {
	Entities []Entity
}

// Collects all the unique package paths for entities in a set.
func (td *TemplateData) Packages() map[string]bool {
	paths := map[string]bool{}
	for _, en := range td.Entities {
		_, ok := paths[en.Pkg.PkgPath]
		if ok {
			continue
		}
		paths[en.Pkg.PkgPath] = true
	}
	return paths
}

// Constructs package names as part of import statement.
func (td *TemplateData) StringPackages() string {
	pkgs := td.Packages()
	var str strings.Builder
	for pkg := range pkgs {
		str.Write([]byte(fmt.Sprintf("\"%s\"\n", pkg)))
	}
	return str.String()
}

// Used to walk the AST tree.
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
	en := Entity{Pkg: pkg, Name: spec.Name, Fields: tobj}
	v.Entities = append(v.Entities, en)
	return v
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

// Write into the template.
func WritePackage(in string, out io.Writer, data *TemplateData) {
	funcMap := make(map[string]interface{})
	tmpl, err := template.New(in).Funcs(funcMap).ParseFiles(in)
	if err != nil {
		panic(err)
	}
	err = tmpl.Execute(out, data)
	if err != nil {
		panic(err)
	}
}
