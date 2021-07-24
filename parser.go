package main

import (
	"fmt"
	"go/ast"
	"go/types"
	"io"
	"log"
	"reflect"
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

type Field struct {
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

func (e *Entity) LsFields() []Field {
	var fds []Field
	for i := 0; i < e.Fields.NumFields(); i++ {
		f := e.Fields.Field(i)
		fds = append(fds, Field{f.Name(), f.Type().String()})
	}
	return fds
}

// Writes entity names for template.
func (td *TemplateData) TemplateGohmFields() string {
	var names strings.Builder
	for _, name := range td.EntityNames() {
		names.WriteString(fmt.Sprintf("%s *%s\n", name, name))
	}
	return names.String()
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
func WritePackage(in string, out io.Writer, data *TemplateData) {
	funcMap := map[string]interface{}{
		"toReceiverCase": toReceiverCase,
		"toLower":        strings.ToLower,
	}
	tmpl, err := template.New(in).Funcs(funcMap).ParseFiles(in)
	if err != nil {
		panic(err)
	}
	err = tmpl.Execute(out, data)
	if err != nil {
		panic(err)
	}
}

// Extracts first letter of word as lower-case.
func toReceiverCase(thing string) string {
	return strings.ToLower(string(thing[0]))
}

// For use in redis.HSet. Produces a map from the fields of a struct.
func MapStruct(i interface{}) map[string]interface{} {
	v := reflect.ValueOf(i)
	t := v.Type()

	res := make(map[string]interface{})
	for i := 0; i < v.NumField(); i++ {
		res[t.Field(i).Name] = v.Field(i).Interface()
	}
	return res
}
