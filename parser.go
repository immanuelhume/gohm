package main

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/packages"
)

// Visitor to walk the AST tree.
type Visitor struct {
	PkgIndex int
	Pkgs     []*packages.Package
	Entities []Entity
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
		logger.Fatalf("Cannot read from package main. (type %s in %s)",
			spec.Name.Name, pkg.PkgPath)
	}
	// create entity and check each field's type
	en := Entity{Pkg: pkg, Name: spec.Name, Fields: tobj}
	en.ValidateFieldType()

	v.Entities = append(v.Entities, en)
	return v
}

// CollectEntities takes a root directory and returns all entities found
// in all packages.
func CollectEntities(dir string) []Entity {
	// TODO: check for re-declarations
	cfg := &packages.Config{Mode: packages.NeedSyntax | packages.NeedName |
		packages.NeedTypes | packages.NeedTypesInfo | packages.NeedModule}
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

// Checks a field's
func (e *Entity) ValidateFieldType() {
	badTypes := map[string]bool{
		"byte": true,
		"rune": true,
	}
	for i := 0; i < e.Fields.NumFields(); i++ {
		t := e.Fields.Field(i).Type().Underlying().String()
		_, ok := badTypes[t]
		if ok {
			logger.Fatalf("Cannot use field of type %s in type %s (%s)",
				t, e.Name.Name, e.Pkg.PkgPath)
		}
	}
}
