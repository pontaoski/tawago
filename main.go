package main

import (
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "tawago",
		Usage: "tawa compiler",
		Commands: []*cli.Command{
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

					fis, err := ioutil.ReadDir("./")
					if err != nil {
						log.Fatalf("%+v", err)
					}

					var t []TopLevel

					for _, fi := range fis {
						if strings.HasSuffix(fi.Name(), ".tawa") {
							handle, err := os.Open(fi.Name())
							if err != nil {
								log.Fatalf("%+v", err)
							}

							l := NewLexer(handle)
							p := NewParser(l)
							err = p.Parse()

							if err != nil {
								log.Fatalf("%+v", err)
							}

							t = append(t, p.ast.Toplevels...)
						}
					}

					module := codegen(t).String()

					if c.Bool("dump") {
						println(module)
						os.Exit(0)
					}

					clang, err := exec.LookPath("clang")
					if err != nil {
						log.Fatalf("%+v", err)
					}

					cmd := exec.Cmd{
						Path:   clang,
						Args:   []string{"-Wl,-e,_tawa_main", "-o", out, "-x", "ir", "-"},
						Stdin:  strings.NewReader(module),
						Stdout: os.Stdout,
						Stderr: os.Stderr,
					}

					err = cmd.Run()
					if err != nil {
						log.Fatalf("%+v", err)
					}

					return nil
				},
			},
		},
	}
	app.Run(os.Args)
}
