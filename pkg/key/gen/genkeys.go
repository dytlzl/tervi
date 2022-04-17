//go:generate go run .

package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
)

const filename = "../key.go"

func main() {

	// Preparation of the data

	type Code struct {
		name  string
		value int
	}

	codes := []Code{
		{name: "Null", value: 0},
		{name: "Enter", value: 13},
		{name: "Esc", value: 27},
		{name: "Del", value: 127},
		{name: "ArrowUp", value: 0x1b5b00 + 'A'},
		{name: "ArrowDown", value: 0x1b5b00 + 'B'},
		{name: "ArrowRight", value: 0x1b5b00 + 'C'},
		{name: "ArrowLeft", value: 0x1b5b00 + 'D'},
	}

	for i := 0; i < 26; i++ {
		codes = append(codes, Code{name: "Ctrl" + string(rune('A'+i)), value: i + 1})
	}

	sort.SliceStable(codes, func(a int, b int) bool { return codes[a].value < codes[b].value })

	// Package Name
	af := &ast.File{
		Name: &ast.Ident{Name: "key"},
	}

	// Declarations of the constants
	decl := &ast.GenDecl{
		Tok: token.CONST,
	}

	for _, v := range codes {
		decl.Specs = append(decl.Specs, &ast.ValueSpec{
			Names: []*ast.Ident{
				{
					Name: v.name,
				},
			},
			Type: &ast.Ident{
				Name: "rune",
			},
			Values: []ast.Expr{
				&ast.BasicLit{
					Kind:  token.INT,
					Value: strconv.Itoa(v.value),
				},
			},
		})
	}

	af.Decls = []ast.Decl{
		decl,
	}

	fileSet := token.NewFileSet()

	var out bytes.Buffer
	_, generatorFilename, _, _ := runtime.Caller(0)
	fmt.Fprintf(&out, "// Code generated by %s; DO NOT EDIT.\n\n", filepath.Base(generatorFilename))
	if err := format.Node(&out, fileSet, af); err != nil {
		log.Fatalf("format.Node: %v", err)
	}

	f, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatalf("os.OpenFile: %v", err)
	}

	defer f.Close()
	f.Write(out.Bytes())
}
