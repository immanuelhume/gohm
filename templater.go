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

// An Model contains raw syntax information for a gohm-tagged struct.
type Model struct {
	// package in which the struct is declared
	Pkg *packages.Package
	// name of this model
	Name *ast.Ident
	// Each field is represented by a *types.Var variable
	Fields []*types.Var
}

type SimpleField struct {
	Name string
	Type string
}

// Shape of the data we're gonna pass into our template file.
type TemplateData struct {
	Models []Model
}

// Constructs the type with its package.
func (m *Model) TNamespacedModel() string {
	return fmt.Sprintf("%s.%s", m.Pkg.Name, m.Name.Name)
}

// Generates a slice of strings denoting each field of
// the model as a pointer.
func (m *Model) TPtrFields() []string {
	// TODO: handle case where field is already a pointer
	ptrFields := []string{}
	for _, f := range m.Fields {
		line := fmt.Sprintf("%s *%s", f.Name(), f.Type().String())
		ptrFields = append(ptrFields, line)
	}
	return ptrFields
}

// LsFields returns the fields on a model as a slice of SimpleField(s).
func (m *Model) TLsFields() []SimpleField {
	var fds []SimpleField
	for _, f := range m.Fields {
		fds = append(fds, SimpleField{f.Name(), f.Type().String()})
	}
	return fds
}

// Collects all the unique packages for models in a set.
func (td *TemplateData) Packages() map[*packages.Package]bool {
	paths := map[*packages.Package]bool{}
	for _, m := range td.Models {
		_, ok := paths[m.Pkg]
		if ok {
			continue
		}
		paths[m.Pkg] = true
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

// Collects raw names of models.
func (td *TemplateData) ModelNames() []string {
	names := []string{}
	for _, m := range td.Models {
		names = append(names, m.Name.Name)
	}
	return names
}

// Writes model names for template. This is used when constructing the
// new client.
func (td *TemplateData) TGohmFields() string {
	var names strings.Builder
	for _, name := range td.ModelNames() {
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

// Lowers the first character of a string.
func lowerFirst(s string) string {
	return strings.ToLower(string(s[0])) + s[1:]
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

// For each field, generate code which loads the appropriate string
// representation into a map.
func TStringifyField(mapName string, f *types.Var, raw string) string {
	switch t := f.Type().Underlying().(type) {
	case *types.Basic:
		switch t.Kind() {
		case types.Int, types.Int8, types.Int16, types.Int32, types.Int64:
			return fmt.Sprintf("%s[%q] = strconv.FormatInt(int64(%s), 10)", mapName, f.Name(), raw)
		case types.Uint, types.Uint8, types.Uint16, types.Uint32, types.Uint64:
			return fmt.Sprintf("%s[%q] = strconv.FormatUint(uint64(%s), 10)", mapName, f.Name(), raw)
		case types.Float32:
			return fmt.Sprintf("%s[%q] = strconv.FormatFloat(float64(%s), 'E', -1, 32)", mapName, f.Name(), raw)
		case types.Float64:
			return fmt.Sprintf("%s[%q] = strconv.FormatFloat(%s, 'E', -1, 64)", mapName, f.Name(), raw)
		case types.Complex64:
			return fmt.Sprintf("%s[%q] = strconv.FormatComplex(complex128(%s), 'E', -1, 64)", mapName, f.Name(), raw)
		case types.Complex128:
			return fmt.Sprintf("%s[%q] = strconv.FormatComplex(%s, 'E', -1, 128)", mapName, f.Name(), raw)
		case types.Bool:
			return fmt.Sprintf("%s[%q] = strconv.FormatBool(%s)", mapName, f.Name(), raw)
		default:
			return fmt.Sprintf("%s[%q] = %s", mapName, f.Name(), raw)
		}
	default:
		return ""
	}
}
