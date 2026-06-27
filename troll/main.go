package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sort"
	"time"

	"github.com/urfave/cli/v3"
	"troll/service"
	"troll/tui"
)

var (
	title           string
	requestInterval time.Duration
)

func main() {
	cli.VersionFlag = &cli.BoolFlag{
		Name:    "version",
		Aliases: []string{"V"},
		Usage:   "print only the version",
	}
	cmd := &cli.Command{
		Name:    "troll",
		Version: "0.5.0",
		Usage:   "Bilibili comment analyzer - identify trolls and patterns",
		Commands: []*cli.Command{
			fetchCommand(),
			queryCommand(),
			configCommand(),
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "title",
				Usage:       "specify title as directory name",
				Aliases:     []string{"T"},
				Destination: &title,
			},
			&cli.DurationFlag{
				Name:        "interval",
				Usage:       "request interval per account (e.g. 1s, 500ms)",
				Aliases:     []string{"I"},
				Value:       2 * time.Second,
				Destination: &requestInterval,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			service.Init(requestInterval)
			return tui.Run()
		},
	}
	sort.Sort(cli.FlagsByName(cmd.Flags))
	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}

func ensureService() {
	service.Init(requestInterval)
	service.EnsureDB()
}

func printTable(headers []string, rows [][]string) {
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}
	for i, h := range headers {
		fmt.Printf("%-*s  ", widths[i], h)
	}
	fmt.Println()
	for i := range headers {
		fmt.Printf("%s  ", repeat("-", widths[i]))
	}
	fmt.Println()
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) {
				fmt.Printf("%-*s  ", widths[i], cell)
			}
		}
		fmt.Println()
	}
}

func repeat(s string, n int) string {
	result := ""
	for i := 0; i < n; i++ {
		result += s
	}
	return result
}
