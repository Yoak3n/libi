package main

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
	"troll/service"
)

type fetchArgs struct {
	topic    string
	AVId     int64
	BVId     string
	maxPages int
}

func fetchCommand() *cli.Command {
	f := &fetchArgs{}
	return &cli.Command{
		Name:  "fetch",
		Usage: "fetch comments from bilibili",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.String("bvid") != "" || cmd.Int64("avid") != -1 {
				if title == "" {
					return cli.Exit("You need to specify a title with --title/-T to save this video's data", 400)
				}
			}
			ensureService()
			fetchEntry(f)
			return nil
		},
		MutuallyExclusiveFlags: []cli.MutuallyExclusiveFlags{
			{
				Required: true,
				Flags: [][]cli.Flag{
					{
						&cli.Int64Flag{
							Name:        "avid",
							Value:       -1,
							Aliases:     []string{"a"},
							Usage:       "specify a video by avid",
							Destination: &f.AVId,
						},
					},
					{
						&cli.StringFlag{
							Name:        "bvid",
							Value:       "",
							Aliases:     []string{"b"},
							Usage:       "specify a video by bvid",
							Destination: &f.BVId,
						},
					},
					{
						&cli.StringFlag{
							Name:        "topic",
							Value:       "",
							Aliases:     []string{"t"},
							Usage:       "specify many videos by topic keyword",
							Destination: &f.topic,
						},
					},
				},
			},
		},
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:        "pages",
				Value:       1,
				Aliases:     []string{"p"},
				Usage:       "max search result pages to fetch (topic mode)",
				Destination: &f.maxPages,
			},
		},
	}
}

func fetchEntry(f *fetchArgs) {
	if title == "" {
		title = f.topic
	}
	h := service.NewHandler("", title, f.topic, f.BVId, f.AVId, f.maxPages)
	h.Run()
	fmt.Println("Fetch completed.")
}
