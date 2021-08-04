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
	// each field is represented by a *types.Var variable
	Fields []*types.Var
}

// Shape of the data we're gonna pass into our template file.
type TemplateData struct {
	Models   []Model
	Packages []*packages.Package
}

type SimpleField struct {
	Name string
	Type string
}

// LsFields returns the fields on a model as a slice of SimpleField(s).
func (m *Model) TLsFields() []SimpleField {
	var fds []SimpleField
	for _, f := range m.Fields {
		fds = append(fds, SimpleField{f.Name(), f.Type().String()})
	}
	return fds
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
// representation into a map. I want to make use of type switching and code
// auto-completion, so this part is not in the template.
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

// TParseField is the reverse of TStringField. For each field's string
// representation in Redis, attempt to return code converting to its actual
// Go type. The value is stored in a variable with the same name as the field.
//
// To generate the corresponding code we need to supply two string expressions:
// 1) @raw is the expression used to access the string value, and 2) @onError is
// the expression returned if an error is caught.
func TParseField(f *types.Var, raw, onError string) string {
	fname := f.Name()
	switch t := f.Type().Underlying().(type) {
	case *types.Basic:
		switch t.Kind() {
		case types.Int:
			return fmt.Sprintf(`%s, err := strconv.Atoi(%s)
if err != nil {
	return %s, err
}`, fname, raw, onError)
		case types.Int8:
			return fmt.Sprintf(`_%s, err := strconv.ParseInt(%s, 10, 8)
if err != nil {
	return %s, err
}
%s := int8(_%s)`, fname, raw, onError, fname, fname)
		case types.Int16:
			return fmt.Sprintf(`_%s, err := strconv.ParseInt(%s, 10, 16)
if err != nil {
	return %s, err
}
%s := int16(_%s)`, fname, raw, onError, fname, fname)
		case types.Int32:
			return fmt.Sprintf(`_%s, err := strconv.ParseInt(%s, 10, 32)
if err != nil {
	return %s, err
}
%s := int32(_%s)`, fname, raw, onError, fname, fname)
		case types.Int64:
			return fmt.Sprintf(`%s, err := strconv.ParseInt(%s, 10, 64)
if err != nil {
	return %s, err
}`, fname, raw, onError)
		case types.Uint:
			return fmt.Sprintf(`_%s, err := strconv.ParseUint(%s, 10, 64)
if err != nil {
	return %s, err
}
%s := uint(_%s)`, fname, raw, onError, fname, fname)
		case types.Uint8:
			return fmt.Sprintf(`_%s, err := strconv.ParseUint(%s, 10, 8)
if err != nil {
	return %s, err
}
%s := uint8(_%s)`, fname, raw, onError, fname, fname)
		case types.Uint16:
			return fmt.Sprintf(`_%s, err := strconv.ParseUint(%s, 10, 16)
if err != nil {
	return %s, err
}
%s := uint16(_%s)`, fname, raw, onError, fname, fname)
		case types.Uint32:
			return fmt.Sprintf(`_%s, err := strconv.ParseUint(%s, 10, 32)
if err != nil {
	return %s, err
}
%s := uint32(_%s)`, fname, raw, onError, fname, fname)
		case types.Uint64:
			return fmt.Sprintf(`%s, err := strconv.ParseUint(%s, 10, 64)
if err != nil {
	return %s, err
}`, fname, raw, onError)
		case types.Float32:
			return fmt.Sprintf(`_%s, err := strconv.ParseFloat(%s, 32)
if err != nil {
	return %s, err
}
%s := float32(_%s)`, fname, raw, onError, fname, fname)
		case types.Float64:
			return fmt.Sprintf(`%s, err := strconv.ParseFloat(%s, 64)
if err != nil {
	return %s, err
}`, fname, raw, onError)
		case types.String:
			return fmt.Sprintf(`%s := %s`, fname, raw)
		case types.Complex64:
			return fmt.Sprintf(`_%s, err := strconv.ParseComplex(%s, 64)
if err != nil {
	return %s, err
}
%s := complex64(_%s)`, fname, raw, onError, fname, fname)
		case types.Complex128:
			return fmt.Sprintf(`%s, err := strconv.ParseComplex(%s, 128)
if err != nil {
	return %s, err
}`, fname, raw, onError)
		case types.Bool:
			return fmt.Sprintf(`%s, err := strconv.ParseBool(%s)
if err != nil {
	return %s, err
}`, fname, raw, onError)
		default:
			return raw
		}
	default:
		return raw
	}
}
