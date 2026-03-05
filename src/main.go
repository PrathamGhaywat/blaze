package main

import (
	"log"
	"os"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "blaze",
		Usage: "Cross-platform package manager",
		Commands: []*cli.Command{
			cmdAdd,
			cmdRemove,
			cmdUse,
			cmdList,
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

var cmdAdd = &cli.Command{
	Name:      "add",
	Usage:     "Add a package from a manifest URL",
	ArgsUsage: "<manifest-url>",
	Action: func(c *cli.Context) error {
		if c.NArg() < 1 {
			return cli.Exit("manifest URL required", 1)
		}
		return handleAdd(c.Args().Get(0))
	},
}

var cmdRemove = &cli.Command{
	Name:      "remove",
	Usage:     "Remove an installed package",
	ArgsUsage: "<package> [version]",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "all",
			Usage: "Remove all versions of the package",
		},
	},
	Action: func(c *cli.Context) error {
		if c.NArg() < 1 {
			return cli.Exit("package name required", 1)
		}
		pkgName := c.Args().Get(0)
		all := c.Bool("all")
		version := c.Args().Get(1)
		return handleRemove(pkgName, version, all)
	},
}

var cmdUse = &cli.Command{
	Name:      "use",
	Usage:     "Use a specific version of an installed package",
	ArgsUsage: "<package@version>",
	Action: func(c *cli.Context) error {
		if c.NArg() < 1 {
			return cli.Exit("package@version required", 1)
		}
		return handleUse(c.Args().Get(0))
	},
}

var cmdList = &cli.Command{
	Name:  "list",
	Usage: "List all installed packages",
	Action: func(c *cli.Context) error {
		return handleList()
	},
}
