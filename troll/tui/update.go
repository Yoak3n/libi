package tui

import (
	"fmt"
	"os/exec"
	"runtime"
	"unicode"

	"troll/service"

	"github.com/Yoak3n/libi/shared/domain/model/schema"
	tea "github.com/charmbracelet/bubbletea"
)

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		return a, nil

	case tea.KeyMsg:
		return a.handleKey(msg)

	case topicsLoadedMsg:
		a.topics = msg
		a.cursor = 0
		if a.state == viewTopicSelect {
			a.topicSelected = make([]bool, len(msg))
		}
	case videosLoadedMsg:
		a.videos = msg
		a.cursor = 0
	case commentsLoadedMsg:
		a.commentSortMode = sortByHot
		a.comments = sortComments(msg.items, a.commentSortMode)
		a.videoTitle = msg.title
		a.flatComments = flattenCommentItems(a.comments)
		a.cursor = 0
	case searchResultsLoadedMsg:
		a.searchResults = msg
		a.state = viewSearchResults
		a.cursor = 0
	case topUsersLoadedMsg:
		a.topUsers = msg
		a.state = viewTopUsers
		a.cursor = 0
	case similarLoadedMsg:
		a.similarComments = msg
		a.state = viewSimilar
		a.cursor = 0
	case dashboardLoadedMsg:
		a.stats = schema.DashboardStats(msg)
		a.state = viewDashboard
	case userCommentsLoadedMsg:
		a.userComments = msg
		a.flatUserItems = flattenUserComments(msg)
		a.state = viewUserComments
		a.cursor = 0
	case signedUsersLoadedMsg:
		a.signedUsers = msg
		a.cursor = 0
	case errMsg:
		a.err = msg
	case browserOpenedMsg:
		if msg.err != nil {
			a.err = fmt.Errorf("open browser: %v", msg.err)
		}
	}
	return a, nil
}

func (a *App) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Global quit
	if key == "ctrl+c" {
		a.quitting = true
		return a, tea.Quit
	}

	// Search input mode
	if a.state == viewSearch {
		switch key {
		case "esc":
			a.state = viewMenu
			a.searchInput = ""
			a.err = nil
			return a, nil
		case "enter":
			if a.searchInput != "" {
				return a, a.doSearch(a.searchInput)
			}
			return a, nil
		case "backspace":
			if len(a.searchInput) > 0 {
				a.searchInput = a.searchInput[:len(a.searchInput)-1]
			}
			return a, nil
		default:
			runes := []rune(key)
			if len(runes) == 1 && unicode.IsPrint(runes[0]) {
				a.searchInput += key
			}
			return a, nil
		}
	}

	// Menu uses menuCursor
	if a.state == viewMenu {
		switch key {
		case "q":
			a.quitting = true
			return a, tea.Quit
		case "up", "k":
			if a.menuCursor > 0 {
				a.menuCursor--
			}
		case "down", "j":
			if a.menuCursor < len(menuItems)-1 {
				a.menuCursor++
			}
		case "enter":
			return a.handleMenuEnter()
		case "/":
			a.state = viewSearch
			a.searchInput = ""
		}
		return a, nil
	}

	// Signed user views have their own key handling (text input, etc.)
	if a.state == viewSignedUsers || a.state == viewAddSignedUser || a.state == viewEditSignedUser {
		if key == "q" && a.state == viewSignedUsers {
			a.state = viewMenu
			a.cursor = 0
			return a, nil
		}
		return a.handleSignedUserKey(key)
	}

	// Comment view: toggle sort mode (handled before main switch)
	if a.state == viewComments && key == "s" {
		a.commentSortMode = (a.commentSortMode + 1) % 3
		a.comments = sortComments(a.comments, a.commentSortMode)
		a.flatComments = flattenCommentItems(a.comments)
		a.cursor = 0
		a.scrollToTop = true
		return a, nil
	}

	// Comment view: add comment author to marked users
	if a.state == viewComments && key == "a" {
		if a.cursor >= 0 && a.cursor < len(a.flatComments) {
			c := a.flatComments[a.cursor]
			if c.OwnerUID > 0 {
				if err := service.CreateSignedUser(c.OwnerUID, ""); err != nil {
					a.err = fmt.Errorf("mark %s failed: %v", c.Owner, err)
				} else {
					a.err = fmt.Errorf("marked: %s (UID:%d)", c.Owner, c.OwnerUID)
				}
			}
		}
		return a, nil
	}

	// Topic select: space toggles, 'a' selects all
	if a.state == viewTopicSelect {
		switch key {
		case " ":
			if a.cursor >= 0 && a.cursor < len(a.topicSelected) {
				a.topicSelected[a.cursor] = !a.topicSelected[a.cursor]
			}
			return a, nil
		case "a":
			allSelected := true
			for _, sel := range a.topicSelected {
				if !sel {
					allSelected = false
					break
				}
			}
			v := !allSelected
			for i := range a.topicSelected {
				a.topicSelected[i] = v
			}
			return a, nil
		}
	}

	switch key {
	case "q":
		// q goes back to menu from sub-views
		a.state = viewMenu
		a.cursor = 0
		a.err = nil
	case "esc":
		return a.handleEsc()
	case "up", "k":
		if a.cursor > 0 {
			a.cursor--
		}
	case "down", "j":
		max := a.maxCursor()
		if a.cursor < max {
			a.cursor++
		}
	case "enter":
		return a.handleEnter()
	}
	return a, nil
}

func (a *App) handleSignedUserKey(key string) (tea.Model, tea.Cmd) {
	switch a.state {
	case viewSignedUsers:
		switch key {
		case "up", "k":
			if a.cursor > 0 {
				a.cursor--
			}
			return a, nil
		case "down", "j":
			if a.cursor < len(a.signedUsers)-1 {
				a.cursor++
			}
			return a, nil
		case "a":
			a.state = viewAddSignedUser
			a.signedUserUID = ""
			a.signedUserInput = ""
			a.signedUserField = "uid"
			return a, nil
		case "e":
			if len(a.signedUsers) > 0 && a.cursor < len(a.signedUsers) {
				u := a.signedUsers[a.cursor]
				a.state = viewEditSignedUser
				a.editTargetUID = u.UID
				a.signedUserInput = u.Description
				return a, nil
			}
		case "d":
			if len(a.signedUsers) > 0 && a.cursor < len(a.signedUsers) {
				uid := a.signedUsers[a.cursor].UID
				_ = service.DeleteSignedUser(uid)
				return a, a.loadSignedUsers()
			}
		case "enter":
			if len(a.signedUsers) > 0 && a.cursor < len(a.signedUsers) {
				u := a.signedUsers[a.cursor]
				a.selectedUser = u.Username
				a.selectedUID = u.UID
				a.state = viewUserComments
				return a, a.loadUserComments(u.UID, "")
			}
		}
	case viewAddSignedUser:
		switch key {
		case "tab":
			if a.signedUserField == "uid" {
				a.signedUserField = "description"
			} else {
				a.signedUserField = "uid"
			}
		case "enter":
			if a.signedUserField == "uid" {
				a.signedUserField = "description"
			} else {
				var uid uint
				n, _ := fmt.Sscanf(a.signedUserUID, "%d", &uid)
				if n == 1 && uid > 0 {
					if err := service.CreateSignedUser(uid, a.signedUserInput); err != nil {
						a.err = fmt.Errorf("add failed: %v", err)
						return a, nil
					}
					a.state = viewSignedUsers
					a.signedUserUID = ""
					a.signedUserInput = ""
					return a, a.loadSignedUsers()
				}
				a.err = fmt.Errorf("invalid UID: %q", a.signedUserUID)
			}
		default:
			if a.signedUserField == "uid" {
				a.signedUserUID = handleTextInput(a.signedUserUID, key)
			} else {
				a.signedUserInput = handleTextInput(a.signedUserInput, key)
			}
		}
	case viewEditSignedUser:
		switch key {
		case "enter":
			if err := service.UpdateSignedUserDescription(a.editTargetUID, a.signedUserInput); err != nil {
				a.err = fmt.Errorf("update failed: %v", err)
				return a, nil
			}
			a.state = viewSignedUsers
			a.signedUserInput = ""
			return a, a.loadSignedUsers()
		default:
			a.signedUserInput = handleTextInput(a.signedUserInput, key)
		}
	}
	return a, nil
}

func handleTextInput(input string, key string) string {
	switch key {
	case "backspace":
		if len(input) > 0 {
			input = input[:len(input)-1]
		}
	default:
		runes := []rune(key)
		if len(runes) == 1 && unicode.IsPrint(runes[0]) {
			input += key
		}
	}
	return input
}

func (a *App) handleEsc() (tea.Model, tea.Cmd) {
	a.err = nil
	switch a.state {
	case viewTopics, viewSearchResults, viewDashboard, viewTopUsers, viewSimilar, viewTopicSelect, viewSignedUsers:
		a.state = viewMenu
		a.cursor = 0
	case viewAddSignedUser, viewEditSignedUser:
		a.state = viewSignedUsers
		a.cursor = 0
		a.signedUserUID = ""
		a.signedUserInput = ""
		return a, a.loadSignedUsers()
	case viewUserComments:
		a.state = viewTopUsers
		a.cursor = 0
	case viewVideos:
		a.state = viewTopics
		a.cursor = 0
		return a, a.loadTopics()
	case viewComments:
		a.state = viewVideos
		a.cursor = 0
	}
	return a, nil
}

func (a *App) openBrowser(url string) tea.Cmd {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return browserOpenedMsg{err: err}
	})
}

func (a *App) handleEnter() (tea.Model, tea.Cmd) {
	switch a.state {
	case viewUserComments:
		if a.cursor >= 0 && a.cursor < len(a.flatUserItems) {
			item := a.flatUserItems[a.cursor]
			if item.Bvid != "" {
				url := fmt.Sprintf("https://www.bilibili.com/video/%s#reply%d", item.Bvid, item.CommentId)
				return a, a.openBrowser(url)
			}
		}
	case viewTopics:
		if len(a.topics) > 0 && a.cursor < len(a.topics) {
			a.topicName = a.topics[a.cursor].Name
			a.state = viewVideos
			return a, a.loadVideos(a.topicName)
		}
	case viewVideos:
		if len(a.videos) > 0 && a.cursor < len(a.videos) {
			a.state = viewComments
			return a, a.loadComments(a.videos[a.cursor].Avid)
		}
	case viewTopicSelect:
		if a.topicSelectFor == "topUsers" {
			var selected []string
			for i, sel := range a.topicSelected {
				if sel && i < len(a.topics) {
					selected = append(selected, a.topics[i].Name)
				}
			}
			if len(selected) == 0 {
				return a, nil
			}
			a.selectedTopics = selected
			if len(selected) == 1 {
				a.topicName = selected[0]
			} else {
				a.topicName = "all"
			}
			return a, a.loadTopUsers(selected)
		}
		// similar: single topic
		if len(a.topics) > 0 && a.cursor < len(a.topics) {
			a.topicName = a.topics[a.cursor].Name
			return a, a.loadSimilar(a.topicName)
		}
	case viewTopUsers:
		if len(a.topUsers) > 0 && a.cursor < len(a.topUsers) {
			u := a.topUsers[a.cursor]
			a.selectedUser = u.Username
			a.selectedUID = u.UID
			topic := a.topicName
			if len(a.selectedTopics) > 1 {
				topic = "all"
			}
			return a, a.loadUserComments(u.UID, topic)
		}
	}
	return a, nil
}

func (a *App) handleMenuEnter() (tea.Model, tea.Cmd) {
	switch a.menuCursor {
	case 0: // Browse
		a.state = viewTopics
		return a, a.loadTopics()
	case 1: // Search
		a.state = viewSearch
		a.searchInput = ""
		return a, nil
	case 2: // Top Users
		a.state = viewTopicSelect
		a.topicSelectFor = "topUsers"
		return a, a.loadTopics()
	case 3: // Similar Comments
		a.state = viewTopicSelect
		a.topicSelectFor = "similar"
		return a, a.loadTopics()
	case 4: // Marked Users
		a.state = viewSignedUsers
		a.cursor = 0
		return a, a.loadSignedUsers()
	case 5: // Dashboard
		return a, a.loadDashboard()
	}
	return a, nil
}

func (a *App) maxCursor() int {
	switch a.state {
	case viewMenu:
		return len(menuItems) - 1
	case viewTopics:
		return len(a.topics) - 1
	case viewVideos:
		return len(a.videos) - 1
	case viewComments:
		return len(a.flatComments) - 1
	case viewSearchResults:
		return len(a.searchResults) - 1
	case viewTopUsers:
		return len(a.topUsers) - 1
	case viewUserComments:
		return len(a.flatUserItems) - 1
	case viewSimilar:
		return len(a.similarComments) - 1
	case viewTopicSelect:
		return len(a.topics) - 1
	case viewSignedUsers:
		return len(a.signedUsers) - 1
	}
	return 0
}
