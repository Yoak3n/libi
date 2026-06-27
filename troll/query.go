package main

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
	"troll/internal/util"
	"troll/service"
)

type queryArgs struct {
	top     string
	count   int
	user    string
	keyword string
}

func queryCommand() *cli.Command {
	q := &queryArgs{}
	return &cli.Command{
		Name:  "query",
		Usage: "query data from troll database",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			ensureService()
			return queryEntry(q)
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "top",
				Value:       "",
				Aliases:     []string{"t"},
				Usage:       "show top users or comments (user|comment)",
				Destination: &q.top,
			},
			&cli.IntFlag{
				Name:        "count",
				Value:       10,
				Aliases:     []string{"n"},
				Usage:       "limit the count of results",
				Destination: &q.count,
			},
			&cli.StringFlag{
				Name:        "user",
				Value:       "",
				Aliases:     []string{"u"},
				Usage:       "show comments of a user (by uid)",
				Destination: &q.user,
			},
			&cli.StringFlag{
				Name:        "keyword",
				Value:       "",
				Aliases:     []string{"k"},
				Usage:       "search comments by keyword",
				Destination: &q.keyword,
			},
		},
	}
}

func queryEntry(q *queryArgs) error {
	if q.top != "" {
		return queryTop(q.top, q.count)
	}
	if q.user != "" {
		return queryUserComments(q.user)
	}
	if q.keyword != "" {
		return querySearch(q.keyword)
	}
	return cli.Exit("Use --top user|comment, --user <uid>, or --keyword <text>", 1)
}

func queryTop(flag string, count int) error {
	switch flag {
	case "user":
		users, err := service.QueryTopNUser(title, count)
		if err != nil {
			return fmt.Errorf("query top users: %w", err)
		}
		rows := make([][]string, len(users))
		for i, u := range users {
			rows[i] = []string{fmt.Sprintf("%d", i+1), u.Username, fmt.Sprintf("%d", u.UID), fmt.Sprintf("%d", u.Count)}
		}
		printTable([]string{"#", "Username", "UID", "Comments"}, rows)
	case "comment":
		comments, err := service.QuerySimilarComments(title, count)
		if err != nil {
			return fmt.Errorf("query similar comments: %w", err)
		}
		if len(comments) == 0 {
			fmt.Println("No similar comments found")
			return nil
		}
		rows := make([][]string, len(comments))
		for i, c := range comments {
			rows[i] = []string{c.Text, fmt.Sprintf("%d", c.Count), c.CommentIds}
		}
		printTable([]string{"Text", "Count", "Comment IDs"}, rows)
	default:
		return cli.Exit("Invalid --top value. Use 'user' or 'comment'", 1)
	}
	return nil
}

func queryUserComments(uidStr string) error {
	var uid uint
	fmt.Sscanf(uidStr, "%d", &uid)
	if uid == 0 {
		return cli.Exit("Invalid user ID", 1)
	}
	// Fetch user info
	service.AddUserByUid(uid)

	// Get comments grouped by video
	groups := service.TrollRepo.GetCommentsWithVideoFromUserInTopic(uid, title)
	if len(groups) == 0 {
		fmt.Printf("No comments found for user %d in topic %q\n", uid, title)
		return nil
	}
	for _, g := range groups {
		bvid := g.Bvid
		if bvid == "" {
			bvid = util.Avid2Bvid(int64(g.Avid))
		}
		fmt.Printf("\nVideo: %s - %s\n", bvid, g.Title)
		for _, c := range g.Comments {
			fmt.Printf("  %s\n", c.Content)
		}
	}
	return nil
}

func querySearch(keyword string) error {
	results := service.SearchCommentWithKeyword(keyword)
	if len(results) == 0 {
		fmt.Println("No comments found")
		return nil
	}
	rows := make([][]string, len(results))
	for i, c := range results {
		content := c.Content
		if len(content) > 60 {
			content = content[:60] + "..."
		}
		rows[i] = []string{fmt.Sprintf("%d", c.Id), c.Owner.Name, content}
	}
	printTable([]string{"ID", "Owner", "Content"}, rows)
	return nil
}
