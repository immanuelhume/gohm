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

type SimpleField struct {
	Name string
	Type string
}

// Shape of the data we're gonna pass into our template file.
type TemplateData struct {
	Entities []Entity
}

// Constructs the type with its package.
func (e *Entity) TNamespacedEntity() string {
	return fmt.Sprintf("%s.%s", e.Pkg.Name, e.Name.Name)
}

// Generates a slice of strings to denote each field of
// the struct as a pointer.
func (e *Entity) TPtrFields() []string {
	// TODO: handle case where field is already a pointer
	fields := []string{}
	for i := 0; i < e.Fields.NumFields(); i++ {
		f := e.Fields.Field(i)
		line := fmt.Sprintf("%s *%s", f.Name(), f.Type().String())
		fields = append(fields, line)
	}
	return fields
}

// LsFields returns the fields on an entity as a slice of SimpleField(s).
func (e *Entity) TLsFields() []SimpleField {
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
func (td *TemplateData) TImports() string {
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
func (td *TemplateData) TGohmFields() string {
	var names strings.Builder
	for _, name := range td.EntityNames() {
		names.WriteString(fmt.Sprintf("%s *%s\n", name, name))
	}
	return names.String()
}

// Write the template into out.
func WritePackage(
	main string,
	in string,
	out io.Writer,
	data interface{},
	funcMap map[string]interface{},
) {
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

// Utility type for templating string conversions.
type MarshallData struct {
	Type    string
	RawExp  string
	ResExp  string
	OnError string
}

// Creates a new type containing the necessary information for creating
// expressions marshalling to-and-fro strings.
func newMarshallData(f SimpleField, rawExp, resExp, onError string) MarshallData {
	return MarshallData{f.Type, rawExp, resExp, onError}
}
