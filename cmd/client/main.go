package main

import (
	"log"
	"os"

	"github.com/fr3shw3b/ably-protocol-exercise/internal/clientapp"
	"github.com/urfave/cli/v2"
)

func main() {
	app := cli.App{
		Name:  "client",
		Usage: "The number sequence protocol client",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "server-host",
				Value: "localhost",
				Usage: "The host on which the server is accessible",
			},
			&cli.IntFlag{
				Name:  "server-port",
				Value: 3000,
				Usage: "The port the server is running on",
			},
			&cli.IntFlag{
				Name:  "sequence-count",
				Value: -1,
				Usage: "The length of the sequence of numbers the server should send",
			},
		},
		Action: func(cCtx *cli.Context) error {
			host := cCtx.String("server-host")
			port := cCtx.Int("server-port")
			sequenceCount := cCtx.Int("sequence-count")
			return clientapp.Run(host, port, sequenceCount)
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
