package tui

import (
	"fmt"
	"strings"

	runewidth "github.com/mattn/go-runewidth"
)

func (a *App) View() string {
	if a.quitting {
		return ""
	}
	var b strings.Builder
	b.WriteString(TitleStyle.Render("Troll - Bilibili Comment Analyzer"))
	b.WriteString("\n\n")

	switch a.state {
	case viewMenu:
		b.WriteString(a.viewMenu())
	case viewTopics:
		b.WriteString(a.viewTopics())
	case viewVideos:
		b.WriteString(a.viewVideos())
	case viewComments:
		b.WriteString(a.viewComments())
	case viewSearch:
		b.WriteString(a.viewSearch())
	case viewSearchResults:
		b.WriteString(a.viewSearchResults())
	case viewDashboard:
		b.WriteString(a.viewDashboard())
	case viewTopUsers:
		b.WriteString(a.viewTopUsers())
	case viewUserComments:
		b.WriteString(a.viewUserComments())
	case viewSimilar:
		b.WriteString(a.viewSimilar())
	case viewTopicSelect:
		b.WriteString(a.viewTopicSelect())
	case viewSignedUsers:
		b.WriteString(a.viewSignedUsers())
	case viewAddSignedUser:
		b.WriteString(a.viewAddSignedUser())
	case viewEditSignedUser:
		b.WriteString(a.viewEditSignedUser())
	}

	if a.err != nil {
		b.WriteString("\n" + ErrorStyle.Render(fmt.Sprintf("Error: %v", a.err)))
	}

	help := a.currentHelp()
	if help != "" {
		b.WriteString("\n" + HelpStyle.Render("  "+help))
	}
	return b.String()
}

func (a *App) currentHelp() string {
	switch a.state {
	case viewMenu:
		return "j/k: navigate • enter: select • /: search • q: quit"
	case viewSearch:
		return "type keyword • enter: search • esc: back"
	case viewComments:
		return "j/k: navigate • s: sort • a: mark user • esc: back"
	case viewUserComments:
		return "j/k: navigate • esc: back"
	case viewSignedUsers:
		return "j/k: navigate • a: add • e: edit description • d: delete • esc: back"
	case viewAddSignedUser:
		return "tab: next field • enter: confirm • esc: cancel"
	case viewEditSignedUser:
		return "type description • enter: save • esc: cancel"
	default:
		return "j/k: navigate • enter: select • esc: back • q: quit"
	}
}

func (a *App) viewMenu() string {
	var b strings.Builder
	b.WriteString(HeaderStyle.Render("Main Menu"))
	b.WriteString("\n\n")
	for i, item := range menuItems {
		cursor := "  "
		if i == a.menuCursor {
			cursor = "> "
		}
		label := fmt.Sprintf("%s%s", cursor, item.label)
		desc := StatusStyle.Render(" - " + item.description)
		if i == a.menuCursor {
			b.WriteString(SelectedItemStyle.Render(label) + desc)
		} else {
			b.WriteString(NormalItemStyle.Render(label) + desc)
		}
		b.WriteString("\n")
	}
	return b.String()
}

func (a *App) viewTopics() string {
	var b strings.Builder
	b.WriteString(HeaderStyle.Render("Topics"))
	b.WriteString("\n")
	if len(a.topics) == 0 {
		b.WriteString("  No topics found. Use 'troll fetch --topic <name>' to fetch data.\n")
		return b.String()
	}
	for i, t := range a.topics {
		cursor := "  "
		if i == a.cursor {
			cursor = "> "
		}
		line := fmt.Sprintf("%s%s %s", cursor, t.Name, CountStyle.Render(fmt.Sprintf("(%d videos)", t.Count)))
		if i == a.cursor {
			b.WriteString(SelectedItemStyle.Render(line))
		} else {
			b.WriteString(NormalItemStyle.Render(line))
		}
		b.WriteString("\n")
	}
	return b.String()
}

func (a *App) viewVideos() string {
	var b strings.Builder
	b.WriteString(HeaderStyle.Render(fmt.Sprintf("Videos in [%s]", a.topicName)))
	b.WriteString("\n")
	if len(a.videos) == 0 {
		b.WriteString("  No videos found.\n")
		return b.String()
	}
	for i, v := range a.videos {
		cursor := "  "
		if i == a.cursor {
			cursor = "> "
		}
		line := fmt.Sprintf("%s%s %s %s",
			cursor,
			v.Title,
			CountStyle.Render(fmt.Sprintf("(%d comments)", v.Count)),
			StatusStyle.Render(v.UpdateAt))
		if i == a.cursor {
			b.WriteString(SelectedItemStyle.Render(line))
		} else {
			b.WriteString(NormalItemStyle.Render(line))
		}
		b.WriteString("\n")
	}
	return b.String()
}

func (a *App) viewComments() string {
	var b strings.Builder
	sortLabel := "hot"
	if a.commentSortMode == sortByTime {
		sortLabel = "newest"
	} else if a.commentSortMode == sortByTimeOld {
		sortLabel = "oldest"
	}
	header := "Comments"
	if a.videoTitle != "" {
		header = fmt.Sprintf("Comments - %s (%d) [sort: %s]", a.videoTitle, a.totalCommentCount(), sortLabel)
	}
	b.WriteString(HeaderStyle.Render(header))
	b.WriteString("\n")
	if len(a.flatComments) == 0 {
		b.WriteString("  No comments found.\n")
		return b.String()
	}

	maxW := a.width - 2
	lines := a.buildCommentLines(maxW)

	// Scroll: find the line range that contains the cursor item
	cursorLine := 0
	for i, l := range lines {
		if l.itemIdx == a.cursor {
			cursorLine = i
			break
		}
	}
	maxVisible := a.height - 4
	if maxVisible < 1 {
		maxVisible = 1
	}
	start := 0
	if a.scrollToTop {
		a.scrollToTop = false
		start = 0
	} else if cursorLine >= maxVisible {
		start = cursorLine - maxVisible + 1
	}
	end := start + maxVisible
	if end > len(lines) {
		end = len(lines)
	}

	for i := start; i < end; i++ {
		l := lines[i]
		if l.itemIdx == a.cursor {
			b.WriteString(SelectedItemStyle.Render(l.text))
		} else if l.isChild {
			b.WriteString(SubCommentStyle.Render(l.text))
		} else {
			b.WriteString(NormalItemStyle.Render(l.text))
		}
		b.WriteString("\n")
	}
	b.WriteString(HelpStyle.Render("  j/k: navigate  s: sort  a: mark user  esc: back"))
	return b.String()
}

func (a *App) buildCommentLines(maxW int) []displayLine {
	var lines []displayLine
	for i, c := range a.flatComments {
		prefix := "  "
		if c.IsChild {
			prefix = "    └ "
		}
		timeTag := ""
		if !c.CreatedAt.IsZero() {
			timeTag = c.CreatedAt.Format("01-02 15:04") + " "
		}
		ownerTag := fmt.Sprintf("[%s] %s", c.Owner, timeTag)
		prefixW := runewidth.StringWidth(prefix + ownerTag)
		wrapped := wrapText(c.Content, maxW, prefixW)
		for j, wl := range wrapped {
			if j == 0 {
				lines = append(lines, displayLine{text: prefix + ownerTag + wl, itemIdx: i, isChild: c.IsChild})
			} else {
				indent := strings.Repeat(" ", prefixW)
				lines = append(lines, displayLine{text: indent + wl, itemIdx: i, isChild: c.IsChild})
			}
		}
	}
	return lines
}

func (a *App) viewSearch() string {
	var b strings.Builder
	b.WriteString(HeaderStyle.Render("Search Comments"))
	b.WriteString("\n\n")
	b.WriteString("  Keyword: " + a.searchInput + "█\n")
	return b.String()
}

func (a *App) viewSearchResults() string {
	var b strings.Builder
	b.WriteString(HeaderStyle.Render(fmt.Sprintf("Search Results (%d)", len(a.searchResults))))
	b.WriteString("\n")
	if len(a.searchResults) == 0 {
		b.WriteString("  No comments found.\n")
		return b.String()
	}

	maxW := a.width - 2
	var lines []displayLine
	for i, c := range a.searchResults {
		prefix := "  "
		ownerTag := fmt.Sprintf("[%s] ", c.Owner)
		prefixW := runewidth.StringWidth(prefix + ownerTag)
		wrapped := wrapText(c.Content, maxW, prefixW)
		for j, wl := range wrapped {
			if j == 0 {
				lines = append(lines, displayLine{text: prefix + ownerTag + wl, itemIdx: i})
			} else {
				lines = append(lines, displayLine{text: strings.Repeat(" ", prefixW) + wl, itemIdx: i})
			}
		}
	}

	cursorLine := 0
	for i, l := range lines {
		if l.itemIdx == a.cursor {
			cursorLine = i
			break
		}
	}
	maxVisible := a.height - 4
	if maxVisible < 1 {
		maxVisible = 1
	}
	start := 0
	if cursorLine >= maxVisible {
		start = cursorLine - maxVisible + 1
	}
	end := start + maxVisible
	if end > len(lines) {
		end = len(lines)
	}
	for i := start; i < end; i++ {
		l := lines[i]
		if l.itemIdx == a.cursor {
			b.WriteString(SelectedItemStyle.Render(l.text))
		} else {
			b.WriteString(NormalItemStyle.Render(l.text))
		}
		b.WriteString("\n")
	}
	return b.String()
}

func (a *App) viewDashboard() string {
	var b strings.Builder
	b.WriteString(HeaderStyle.Render("Dashboard"))
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("  %-20s %s\n", "Topics:", StatsValueStyle.Render(fmt.Sprintf("%d", a.stats.Topics))))
	b.WriteString(fmt.Sprintf("  %-20s %s\n", "Videos:", StatsValueStyle.Render(fmt.Sprintf("%d", a.stats.Videos))))
	b.WriteString(fmt.Sprintf("  %-20s %s\n", "Users:", StatsValueStyle.Render(fmt.Sprintf("%d", a.stats.Users))))
	b.WriteString(fmt.Sprintf("  %-20s %s\n", "Comments:", StatsValueStyle.Render(fmt.Sprintf("%d", a.stats.Comments))))
	return b.String()
}

func (a *App) viewTopUsers() string {
	var b strings.Builder
	b.WriteString(HeaderStyle.Render(fmt.Sprintf("Top Users in [%s]", a.topicName)))
	b.WriteString("\n")
	if len(a.topUsers) == 0 {
		b.WriteString("  No users found.\n")
		return b.String()
	}
	start := 0
	maxVisible := a.height - 6
	if maxVisible < 1 {
		maxVisible = 1
	}
	if a.cursor >= maxVisible {
		start = a.cursor - maxVisible + 1
	}
	end := start + maxVisible
	if end > len(a.topUsers) {
		end = len(a.topUsers)
	}
	for i := start; i < end; i++ {
		u := a.topUsers[i]
		cursor := "  "
		if i == a.cursor {
			cursor = "> "
		}
		rank := RankStyle.Render(fmt.Sprintf("#%-3d", u.Rank))
		name := fmt.Sprintf("%-20s", u.Username)
		count := CountStyle.Render(fmt.Sprintf("%d comments", u.Count))
		uid := StatusStyle.Render(fmt.Sprintf("UID:%d", u.UID))
		line := fmt.Sprintf("%s%s %s %s %s", cursor, rank, name, count, uid)
		if i == a.cursor {
			b.WriteString(SelectedItemStyle.Render(line))
		} else {
			b.WriteString(NormalItemStyle.Render(line))
		}
		b.WriteString("\n")
	}
	if len(a.topUsers) > maxVisible {
		b.WriteString(StatusStyle.Render(fmt.Sprintf("  Showing %d-%d of %d", start+1, end, len(a.topUsers))))
	}
	return b.String()
}

func (a *App) viewUserComments() string {
	var b strings.Builder
	topicLabel := a.topicName
	if topicLabel == "" {
		topicLabel = "All"
	}
	b.WriteString(HeaderStyle.Render(fmt.Sprintf("%s (UID:%d) Comments in [%s] (%d)", a.selectedUser, a.selectedUID, topicLabel, len(a.flatUserItems))))
	b.WriteString("\n")
	if len(a.flatUserItems) == 0 {
		b.WriteString("  No comments found.\n")
		return b.String()
	}

	maxW := a.width - 2
	var lines []displayLine
	for i, item := range a.flatUserItems {
		ts := ""
		if !item.CreatedAt.IsZero() {
			ts = item.CreatedAt.Format("01-02 15:04 ")
		}
		videoTag := ""
		if item.VideoTitle != "" {
			short := item.VideoTitle
			if len([]rune(short)) > 40 {
				short = string([]rune(short)[:40]) + ".."
			}
			videoTag = fmt.Sprintf("[%s] ", short)
		}
		prefix := "  " + ts + videoTag
		prefixW := runewidth.StringWidth(prefix)
		wrapped := wrapText(item.Content, maxW, prefixW)
		for j, wl := range wrapped {
			if j == 0 {
				lines = append(lines, displayLine{text: prefix + wl, itemIdx: i})
			} else {
				lines = append(lines, displayLine{text: strings.Repeat(" ", prefixW) + wl, itemIdx: i})
			}
		}
	}

	cursorLine := 0
	for i, l := range lines {
		if l.itemIdx == a.cursor {
			cursorLine = i
			break
		}
	}
	maxVisible := a.height - 4
	if maxVisible < 1 {
		maxVisible = 1
	}
	start := 0
	if cursorLine >= maxVisible {
		start = cursorLine - maxVisible + 1
	}
	end := start + maxVisible
	if end > len(lines) {
		end = len(lines)
	}
	for i := start; i < end; i++ {
		l := lines[i]
		if l.itemIdx == a.cursor {
			b.WriteString(SelectedItemStyle.Render(l.text))
		} else {
			b.WriteString(NormalItemStyle.Render(l.text))
		}
		b.WriteString("\n")
	}
	return b.String()
}

func (a *App) viewSimilar() string {
	var b strings.Builder
	b.WriteString(HeaderStyle.Render(fmt.Sprintf("Similar Comments in [%s]", a.topicName)))
	b.WriteString("\n")
	if len(a.similarComments) == 0 {
		b.WriteString("  No similar comments found.\n")
		return b.String()
	}
	start := 0
	maxVisible := a.height - 6
	if maxVisible < 1 {
		maxVisible = 1
	}
	if a.cursor >= maxVisible {
		start = a.cursor - maxVisible + 1
	}
	end := start + maxVisible
	if end > len(a.similarComments) {
		end = len(a.similarComments)
	}
	for i := start; i < end; i++ {
		s := a.similarComments[i]
		cursor := "  "
		if i == a.cursor {
			cursor = "> "
		}
		text := s.Text
		if len(text) > 80 {
			text = text[:80] + "..."
		}
		count := CountStyle.Render(fmt.Sprintf("(x%d)", s.Count))
		line := fmt.Sprintf("%s%s %s", cursor, text, count)
		if i == a.cursor {
			b.WriteString(SelectedItemStyle.Render(line))
		} else {
			b.WriteString(NormalItemStyle.Render(line))
		}
		b.WriteString("\n")
	}
	if len(a.similarComments) > maxVisible {
		b.WriteString(StatusStyle.Render(fmt.Sprintf("  Showing %d-%d of %d", start+1, end, len(a.similarComments))))
	}
	return b.String()
}

func (a *App) viewTopicSelect() string {
	var b strings.Builder
	label := "Top Users"
	if a.topicSelectFor == "similar" {
		label = "Similar Comments"
	}
	b.WriteString(HeaderStyle.Render(fmt.Sprintf("Select Topic for %s", label)))
	b.WriteString("\n")
	if len(a.topics) == 0 {
		b.WriteString("  No topics found.\n")
		return b.String()
	}
	for i, t := range a.topics {
		cursor := "  "
		if i == a.cursor {
			cursor = "> "
		}
		line := fmt.Sprintf("%s%s %s", cursor, t.Name, CountStyle.Render(fmt.Sprintf("(%d videos)", t.Count)))
		if i == a.cursor {
			b.WriteString(SelectedItemStyle.Render(line))
		} else {
			b.WriteString(NormalItemStyle.Render(line))
		}
		b.WriteString("\n")
	}
	return b.String()
}

func (a *App) viewSignedUsers() string {
	var b strings.Builder
	b.WriteString(HeaderStyle.Render("Marked Users"))
	b.WriteString("\n")
	if len(a.signedUsers) == 0 {
		b.WriteString("  No marked users. Press 'a' to add one.\n")
		return b.String()
	}
	for i, u := range a.signedUsers {
		cursor := "  "
		if i == a.cursor {
			cursor = "> "
		}
		desc := u.Description
		if desc == "" {
			desc = "(no description)"
		}
		timeStr := ""
		if !u.LastViewed.IsZero() {
			timeStr = " viewed " + u.LastViewed.Format("01-02 15:04")
		}
		line := fmt.Sprintf("%s%s %s%s", cursor, u.Username, CountStyle.Render(desc), StatusStyle.Render(timeStr))
		if i == a.cursor {
			b.WriteString(SelectedItemStyle.Render(line))
		} else {
			b.WriteString(NormalItemStyle.Render(line))
		}
		b.WriteString("\n")
	}
	return b.String()
}

func (a *App) viewAddSignedUser() string {
	var b strings.Builder
	b.WriteString(HeaderStyle.Render("Add Marked User"))
	b.WriteString("\n\n")
	uidLabel := "  UID: "
	descLabel := "  Description: "
	if a.signedUserField == "uid" {
		b.WriteString(uidLabel + SelectedItemStyle.Render(a.signedUserUID+"█") + "\n")
		b.WriteString(descLabel + NormalItemStyle.Render(a.signedUserInput) + "\n")
	} else {
		b.WriteString(uidLabel + NormalItemStyle.Render(a.signedUserUID) + "\n")
		b.WriteString(descLabel + SelectedItemStyle.Render(a.signedUserInput+"█") + "\n")
	}
	b.WriteString("\n" + StatusStyle.Render("tab: switch field • enter: confirm • esc: cancel"))
	return b.String()
}

func (a *App) viewEditSignedUser() string {
	var b strings.Builder
	b.WriteString(HeaderStyle.Render(fmt.Sprintf("Edit Description (UID: %d)", a.editTargetUID)))
	b.WriteString("\n\n")
	b.WriteString("  Description: " + SelectedItemStyle.Render(a.signedUserInput+"█") + "\n")
	b.WriteString("\n" + StatusStyle.Render("enter: save • esc: cancel"))
	return b.String()
}
