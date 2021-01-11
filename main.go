package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

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
		if strings.HasSuffix(fi.Name(), ".tawa") {
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
	Package string `yaml:"package"`
}

func main() {
	app := &cli.App{
		Name:  "tawago",
		Usage: "tawa compiler",
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
					fi, err := os.Create("TawaModule")
					if err != nil {
						fmt.Printf("error creating TawaModule: %s", err)
						os.Exit(1)
					}
					defer fi.Close()

					out, err := yaml.Marshal(yml)
					if err != nil {
						fmt.Printf("error creating TawaModule: %s", err)
						os.Exit(1)
					}

					_, err = fi.Write(out)
					if err != nil {
						fmt.Printf("error creating TawaModule: %s", err)
						os.Exit(1)
					}

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
				},
				Action: func(c *cli.Context) error {
					out := c.String("output")

					t := parseDirectory("./")

					module := codegen(t).String()

					if c.Bool("dump") {
						println(module)
						os.Exit(0)
					}

					data, err := ioutil.ReadFile("TawaModule")
					if err != nil {
						fmt.Printf("error reading TawaModule: %s", err)
						os.Exit(1)
					}

					var doc tawaModule
					err = yaml.Unmarshal(data, &doc)
					if err != nil {
						fmt.Printf("error reading TawaModule: %s", err)
						os.Exit(1)
					}
					if out == "" {
						out = doc.Package
					}

					cmd := exec.Command("clang", "-nostdlib", "-Wl,-e,_tawa_main", "-o", out, "-x", "ir", "-")

					cmd.Stdin = strings.NewReader(module)
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
