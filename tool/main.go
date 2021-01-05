package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/alecthomas/participle"

	. "github.com/dave/jennifer/jen"
)

type TypeDecls struct {
	Declarations []*Declaration `@@*`
}

type TCase struct {
	Name string `@Ident "of"`
	Kind string `(@Ident | @String | @RawString)`
}

type Declaration struct {
	Name  string   `"type" @Ident "="`
	Plain *string  `(  (@Ident | @String | @RawString)`
	Many  *[]TCase ` | ("|" (@@))*)`
	I     struct{} `";"`
}

func (t *TypeDecls) IsSumType(name string) bool {
	for _, decls := range t.Declarations {
		if decls.Name == name && decls.Many != nil {
			return true
		}
	}
	return false
}

func GenerateDecls(pkgname string, t *TypeDecls) string {
	f := NewFile(pkgname)

	for _, decl := range t.Declarations {

		if decl.Plain != nil {
			f.Type().Id(decl.Name).Id(*decl.Plain)
		} else if decl.Many != nil {
			f.Type().Id(decl.Name).Interface(
				Id("is_" + decl.Name).Params(),
			)

			for _, it := range *decl.Many {
				if t.IsSumType(it.Kind) {
					f.Type().Id(it.Name).Struct(Id(it.Kind))
				} else {
					f.Type().Id(it.Name).Id(it.Kind)
				}

				f.Func().Params(Id("v").Id(it.Name)).Id("is_" + decl.Name).Params().Block()
			}
		}
	}

	return fmt.Sprintf("%#v", f)
}

func main() {
	parser := participle.MustBuild(&TypeDecls{})

	in := os.Args[1]
	out := os.Args[2]
	pkgname := os.Args[3]

	inData, err := ioutil.ReadFile(in)
	if err != nil {
		panic(err)
	}

	ast := TypeDecls{}
	err = parser.ParseBytes(inData, &ast)
	if err != nil {
		panic(err)
	}

	err = ioutil.WriteFile(out, []byte(GenerateDecls(pkgname, &ast)), os.ModePerm)
	if err != nil {
		panic(err)
	}
}
