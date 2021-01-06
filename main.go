package main

import (
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
						Name:     "input",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "output",
						Required: true,
					},
				},
				Action: func(c *cli.Context) error {
					in := c.String("input")
					out := c.String("output")

					fi, err := os.Open(in)
					if err != nil {
						log.Fatalf("%+v", err)
					}
					defer fi.Close()

					l := NewLexer(fi)
					p := NewParser(l)
					err = p.Parse()

					if err != nil {
						log.Fatalf("%+v", err)
					}

					module := codegen(p.ast.Toplevels).String()

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
