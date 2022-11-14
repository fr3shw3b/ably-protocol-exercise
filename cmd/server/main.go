package main

import (
	"log"
	"os"

	"github.com/fr3shw3b/ably-protocol-exercise/internal/serverapp"
	"github.com/urfave/cli/v2"
)

func main() {
	app := cli.App{
		Name:  "server",
		Usage: "The number sequence protocol server",
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:  "port",
				Value: 3000,
				Usage: "The port to run the server on",
			},
		},
		Action: func(cCtx *cli.Context) error {
			port := cCtx.Int("port")
			return serverapp.Run(port)
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
