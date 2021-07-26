package main

import (
	"fmt"
	"go/ast"
	"go/types"
	"io"
	"log"
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

type SimpleField struct {
	Name string
	Type string
}

// Shape of the data we're gonna pass into our template file.
type TemplateData struct {
	Entities []Entity
}

// Constructs the type with its package.
func (e *Entity) NamespacedEntity() string {
	return fmt.Sprintf("%s.%s", e.Pkg.Name, e.Name.Name)
}

// Generates a slice of strings to denote each field of
// the struct as a pointer.
func (e *Entity) PtrFields() []string {
	// TODO: handle case where field is already a pointer
	fields := []string{}
	for i := 0; i < e.Fields.NumFields(); i++ {
		f := e.Fields.Field(i)
		line := fmt.Sprintf("%s *%s", f.Name(), f.Type().String())
		fields = append(fields, line)
	}
	return fields
}

// LsFields returns the fields on an entity as a slice of SimpleField's.
func (e *Entity) LsFields() []SimpleField {
	var fds []SimpleField
	for i := 0; i < e.Fields.NumFields(); i++ {
		f := e.Fields.Field(i)
		fds = append(fds, SimpleField{f.Name(), f.Type().String()})
	}
	return fds
}

// Collects all the unique packages for entities in a set.
func (td *TemplateData) Packages() map[*packages.Package]bool {
	paths := map[*packages.Package]bool{}
	for _, en := range td.Entities {
		_, ok := paths[en.Pkg]
		if ok {
			continue
		}
		paths[en.Pkg] = true
	}
	return paths
}

// Constructs package names as part of import statement.
func (td *TemplateData) TemplateImports() string {
	pkgs := td.Packages()
	var str strings.Builder
	for pkg := range pkgs {
		str.WriteString(fmt.Sprintf("%s \"%s\"\n", pkg.Name, pkg.PkgPath))
	}
	return str.String()
}

// Collects raw names of entities.
func (td *TemplateData) EntityNames() []string {
	names := []string{}
	for _, en := range td.Entities {
		names = append(names, en.Name.Name)
	}
	return names
}

// Writes entity names for template. This is used when constructing the
// new client.
func (td *TemplateData) TemplateGohmFields() string {
	var names strings.Builder
	for _, name := range td.EntityNames() {
		names.WriteString(fmt.Sprintf("%s *%s\n", name, name))
	}
	return names.String()
}

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
		log.Fatalf("Cannot read from package main. (type %s in %s)",
			spec.Name.Name, pkg.PkgPath)
	}
	// create entity
	en := Entity{Pkg: pkg, Name: spec.Name, Fields: tobj}
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

// Write into the template.
func WritePackage(main string, in string, out io.Writer, data interface{}, funcMap map[string]interface{}) {
	tmpl, err := template.New(main).Funcs(funcMap).ParseGlob(in)
	if err != nil {
		panic(err)
	}
	err = tmpl.ExecuteTemplate(out, main, data)
	if err != nil {
		panic(err)
	}
}

// Extracts first letter of word as lower-case.
func toReceiverCase(thing string) string {
	return strings.ToLower(string(thing[0]))
}

// Utility type for converting to-and-fro strings.
type MarshallData struct {
	Type    string
	RawExp  string
	ResExp  string
	OnError string
}

// Creates a new type containing the necessary information for marshalling
// to-and-fro strings.
func newMarshallData(f SimpleField, rawExp, resExp, onError string) MarshallData {
	return MarshallData{f.Type, rawExp, resExp, onError}
}
