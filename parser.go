package main

import (
	"fmt"
	"go/ast"
	"go/types"
	"io"
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

// Shape of the data we're gonna pass into our template file.
type TemplateData struct {
	Entities []Entity
}

// Constructs the type with its package.
func (e *Entity) NamespacedEntity() string {
	return fmt.Sprintf("%s.%s", Namespace(e.Pkg), e.Name.Name)
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
func (td *TemplateData) TemplatePackages() string {
	pkgs := td.Packages()
	var str strings.Builder
	for pkg := range pkgs {
		str.WriteString(fmt.Sprintf("\"%s\"\n", pkg))
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
	// create entity
	en := Entity{Pkg: pkg, Name: spec.Name, Fields: tobj}
	v.Entities = append(v.Entities, en)
	return v
}

func CollectEntities(dir string) []Entity {
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

func Namespace(pkg *packages.Package) string {
	var namespace string
	if pkg.Name == "main" {
		spl := strings.Split(pkg.PkgPath, "/")
		namespace = spl[len(spl)-1]
	} else {
		namespace = pkg.Name
	}
	return namespace
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
