package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/alecthomas/repr"
	"github.com/urfave/cli/v2"
	"github.com/ztrue/tracerr"
	"gopkg.in/yaml.v2"
)

func parseDirectory(dir string) []TopLevel {
	var t []TopLevel

	fis, err := ioutil.ReadDir("./")
	if err != nil {
		tracerr.PrintSourceColor(err)
		os.Exit(1)
	}

	for _, fi := range fis {
		if strings.HasSuffix(fi.Name(), ".Tawa Source File") {
			handle, err := os.Open(fi.Name())
			if err != nil {
				tracerr.PrintSourceColor(err)
				os.Exit(1)
			}

			l := NewLexer(handle, fi.Name())
			p := NewParser(l)
			err = p.Parse()

			if err != nil {
				tracerr.PrintSourceColor(err)
				os.Exit(1)
			}

			t = append(t, p.ast.Toplevels...)
		}
	}

	return t
}

type tawaModule struct {
	Package string `yaml:"Package"`
}

func main() {
	app := &cli.App{
		Name:  "tawago",
		Usage: "tawa compiler",
		ExitErrHandler: func(context *cli.Context, err error) {
			log.Fatal("error with tawac: %w", err)
		},
		Commands: []*cli.Command{
			{
				Name:  "init",
				Usage: "init a directory",
				Action: func(c *cli.Context) error {
					name := c.Args().First()
					if name == "" {
						fmt.Printf("no module name provided")
						os.Exit(1)
					}
					yml := tawaModule{
						Package: name,
					}
					fi, err := os.Create("Tawa Module Information")
					if err != nil {
						fmt.Printf("error creating Tawa Module Information: %s", err)
						os.Exit(1)
					}
					defer fi.Close()

					out, err := yaml.Marshal(yml)
					if err != nil {
						fmt.Printf("error creating Tawa Module Information: %s", err)
						os.Exit(1)
					}

					_, err = fi.Write(out)
					if err != nil {
						fmt.Printf("error creating Tawa Module Information: %s", err)
						os.Exit(1)
					}

					return nil
				},
			},
			{
				Name:  "typeinfo",
				Usage: "dump typeinfo from a compiled module",
				Action: func(c *cli.Context) error {
					file := c.Args().Get(0)
					data, err := getTypeInfoFromFile(file)
					if err != nil {
						return err
					}
					repr.Println(data)
					return nil
				},
			},
			{
				Name:  "build",
				Usage: "build a file",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name: "output",
					},
					&cli.BoolFlag{
						Name:  "dump",
						Value: false,
					},
					&cli.BoolFlag{
						Name:  "library",
						Value: false,
					},
					&cli.StringSliceFlag{
						Name:  "force-import",
						Value: cli.NewStringSlice(),
					},
				},
				Action: func(c *cli.Context) error {
					out := c.String("output")

					data, err := ioutil.ReadFile("Tawa Module Information")
					if err != nil {
						fmt.Printf("error reading Tawa Module Information: %s", err)
						os.Exit(1)
					}

					var doc tawaModule
					err = yaml.Unmarshal(data, &doc)
					if err != nil {
						fmt.Printf("error reading Tawa Module Information: %s", err)
						os.Exit(1)
					}
					if out == "" {
						out = doc.Package
					}
					if c.Bool("library") {
						out += ".Dynamically Linked Tawa Module"
					}

					t := parseDirectory("./")

					module := codegen(
						t,
						settings{
							isLibrary:       c.Bool("library"),
							packageName:     doc.Package,
							forceimportlibs: c.StringSlice("force-import"),
						},
					).String()

					if c.Bool("dump") {
						println(module)
						os.Exit(0)
					}

					cmd := exec.Command("clang", "-nostdlib", "-o", out)

					for _, lib := range c.StringSlice("force-import") {
						cmd.Args = append(cmd.Args, lib)
					}

					if c.Bool("library") {
						cmd.Args = append(cmd.Args, "-shared", "-no-pie")
					} else {
						cmd.Args = append(cmd.Args, "-Wl,-e,_tawa_main")
					}

					fi, err := ioutil.TempFile("/tmp", "*.ll")
					if err != nil {
						return err
					}
					defer fi.Close()
					_, err = io.Copy(fi, strings.NewReader(module))
					if err != nil {
						return err
					}

					cmd.Args = append(cmd.Args, fi.Name())

					cmd.Stdout = os.Stdout
					cmd.Stderr = os.Stderr

					err = cmd.Run()
					if err != nil {
						tracerr.PrintSourceColor(err)
						os.Exit(1)
					}

					return nil
				},
			},
		},
	}
	app.Run(os.Args)
}
