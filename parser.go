package main

import (
	"go/ast"
	"go/types"
	"log"

	"golang.org/x/tools/go/packages"
)

// Visitor to walk the AST tree.
type Visitor struct {
	PkgIndex  int
	Pkgs      []*packages.Package // all packages found within the module
	Models    []Model
	ModelPkgs map[*packages.Package]bool // only packages which contain models
}

// Visit will add nodes which are a struct type tagged with the // gohm
// comment. All other nodes are ignored. A Model type is constructed and added
// to a slice.
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
	// If we reached here then the model is valid. Create model and check each field's type
	m := Model{Pkg: pkg, Name: spec.Name, Fields: CollectFields(tobj)}
	m.ValidateFields()
	// collect the data in our visitor
	v.Models = append(v.Models, m)
	v.ModelPkgs[m.Pkg] = true
	return v
}

// Convenience function to dump all fields of a struct into a slice.
func CollectFields(s *types.Struct) []*types.Var {
	var fields []*types.Var
	for i := 0; i < s.NumFields(); i++ {
		fields = append(fields, s.Field(i))
	}
	return fields
}

// CollectTemplateData takes a root directory and returns all models found
// in all packages.
func CollectTemplateData(dir string) TemplateData {
	// TODO: check for re-declarations
	cfg := &packages.Config{Mode: packages.NeedSyntax | packages.NeedName |
		packages.NeedTypes | packages.NeedTypesInfo | packages.NeedModule}
	pkgs, err := packages.Load(cfg, dir+"/...")
	if err != nil {
		panic(err)
	}
	v := &Visitor{Pkgs: pkgs, Models: []Model{}, ModelPkgs: make(map[*packages.Package]bool)}
	for i, pkg := range pkgs {
		v.PkgIndex = i
		for _, file := range pkg.Syntax {
			ast.Walk(v, file)
		}
	}
	// Visitor collected the packages as a set. Here we'll convert it into a slice.
	var modelPkgs []*packages.Package
	for pkg := range v.ModelPkgs {
		modelPkgs = append(modelPkgs, pkg)
	}
	return TemplateData{Models: v.Models, Packages: modelPkgs}
}

// These types are not supported. They cannot be sensibly marshalled into a string
// and stored in Redis.
var badTypes = map[interface{}]bool{}

// Validates each field. Exits if a problem was discovered.
func (m *Model) ValidateFields() {
	for _, field := range m.Fields {
		t := field.Type().Underlying()
		switch t := t.(type) {
		case *types.Basic:
			_, got := badTypes[t.Kind()]
			if got {
				log.Fatalf("Cannot use field of type %s in type %s (%s)",
					t, m.Name.Name, m.Pkg.PkgPath)
			}
		}
	}
}
