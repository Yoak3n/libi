package tui

import (
	"sort"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	runewidth "github.com/mattn/go-runewidth"
)

func sortComments(comments []commentItem, mode sortMode) []commentItem {
	sorted := make([]commentItem, len(comments))
	copy(sorted, comments)
	switch mode {
	case sortByHot:
		sort.Slice(sorted, func(i, j int) bool {
			scoreI := int(sorted[i].Like)*2 + len(sorted[i].Children)*5
			scoreJ := int(sorted[j].Like)*2 + len(sorted[j].Children)*5
			return scoreI > scoreJ
		})
	case sortByTime:
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].CreatedAt.After(sorted[j].CreatedAt)
		})
	case sortByTimeOld:
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].CreatedAt.Before(sorted[j].CreatedAt)
		})
	}
	return sorted
}

func flattenCommentItems(comments []commentItem) []commentItem {
	flat := make([]commentItem, 0, len(comments))
	for _, c := range comments {
		flat = append(flat, c)
		for _, child := range c.Children {
			child.IsChild = true
			flat = append(flat, child)
		}
	}
	return flat
}

func flattenUserComments(groups []userVideoGroup) []userCommentDisplayItem {
	type tsItem struct {
		item      userCommentDisplayItem
		timestamp time.Time
	}
	var all []tsItem
	for _, g := range groups {
		for _, c := range g.Comments {
			all = append(all, tsItem{
				item: userCommentDisplayItem{
					IsHeader:   false,
					VideoTitle: g.VideoTitle,
					Bvid:       g.Bvid,
					Content:    c.Content,
					CommentId:  c.Id,
					CreatedAt:  c.CreatedAt,
				},
				timestamp: c.CreatedAt,
			})
		}
	}
	sort.Slice(all, func(i, j int) bool {
		return all[i].timestamp.After(all[j].timestamp)
	})

	flat := make([]userCommentDisplayItem, 0, len(all))
	for _, s := range all {
		flat = append(flat, s.item)
	}
	return flat
}

func (a *App) totalCommentCount() int {
	return len(a.flatComments)
}

// wrapText wraps text to fit within maxWidth display columns, respecting CJK wide chars.
// prefixWidth is the display width already consumed on the first line (cursor + tags).
func wrapText(text string, maxWidth int, prefixWidth int) []string {
	if maxWidth <= prefixWidth {
		return []string{text}
	}
	firstLineW := maxWidth - prefixWidth
	if firstLineW <= 0 {
		firstLineW = 10
	}
	continuationW := maxWidth - 4 // indent for wrapped lines

	var lines []string
	runes := []rune(text)
	pos := 0
	first := true
	for pos < len(runes) {
		limit := continuationW
		if first {
			limit = firstLineW
			first = false
		}
		// Find the break point
		w := 0
		breakAt := pos
		lastSpace := -1
		for i := pos; i < len(runes); i++ {
			rw := runewidth.RuneWidth(runes[i])
			if w+rw > limit {
				break
			}
			w += rw
			if runes[i] == ' ' || runes[i] == '\t' {
				lastSpace = i
			}
			breakAt = i + 1
		}
		// Try to break at last space
		if breakAt < len(runes) && lastSpace > pos {
			breakAt = lastSpace + 1
		}
		lines = append(lines, string(runes[pos:breakAt]))
		pos = breakAt
	}
	if len(lines) == 0 {
		lines = []string{text}
	}
	return lines
}

func Run() error {
	p := tea.NewProgram(NewApp(), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
