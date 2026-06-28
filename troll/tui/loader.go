package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"troll/service"
)

func NewApp() *App {
	return &App{
		state:  viewMenu,
		width:  80,
		height: 24,
	}
}

func (a *App) Init() tea.Cmd {
	return nil
}

func (a *App) loadTopics() tea.Cmd {
	return func() tea.Msg {
		service.EnsureDB()
		topics := service.GetAllTopicsList()
		items := make([]topicItem, len(topics))
		for i, t := range topics {
			items[i] = topicItem{Name: t.Name, Count: t.Count}
		}
		return topicsLoadedMsg(items)
	}
}

func (a *App) loadVideos(topic string) tea.Cmd {
	return func() tea.Msg {
		videos := service.GetVideosByTopic(topic)
		items := make([]videoItem, len(videos))
		for i, v := range videos {
			items[i] = videoItem{
				Avid:     v.Avid,
				Bvid:     v.Bvid,
				Title:    v.Title,
				Count:    v.Count,
				UpdateAt: v.UpdateAt,
			}
		}
		return videosLoadedMsg(items)
	}
}

func (a *App) loadComments(avid uint) tea.Cmd {
	return func() tea.Msg {
		result := service.GetCommentsByVideo(avid)
		items := make([]commentItem, 0, len(result.Comments))
		for _, c := range result.Comments {
			item := commentItem{
				Id:        c.Id,
				Content:   c.Content,
				Owner:     c.Owner.Name,
				OwnerUID:  c.Owner.Uid,
				Like:      c.Like,
				CreatedAt: c.CreatedAt,
			}
			for _, child := range c.Children {
				item.Children = append(item.Children, commentItem{
					Id:        child.Id,
					Content:   child.Content,
					Owner:     child.Owner.Name,
					OwnerUID:  child.Owner.Uid,
					Like:      child.Like,
					CreatedAt: child.CreatedAt,
				})
			}
			items = append(items, item)
		}
		return commentsLoadedMsg{items: items, title: result.Title}
	}
}

func (a *App) doSearch(keyword string) tea.Cmd {
	return func() tea.Msg {
		service.EnsureDB()
		results := service.SearchCommentWithKeyword(keyword)
		items := make([]commentItem, 0, len(results))
		for _, c := range results {
			items = append(items, commentItem{
				Id:        c.Id,
				Content:   c.Content,
				Owner:     c.Owner.Name,
				OwnerUID:  c.Owner.Uid,
				CreatedAt: c.CreatedAt,
			})
		}
		return searchResultsLoadedMsg(items)
	}
}

func (a *App) loadTopUsers(topics []string) tea.Cmd {
	return func() tea.Msg {
		users, err := service.QueryTopNUserMultiTopic(topics, 30)
		if err != nil {
			return errMsg(err)
		}
		items := make([]topUserItem, len(users))
		for i, u := range users {
			items[i] = topUserItem{
				Rank:     i + 1,
				Username: u.Username,
				UID:      u.UID,
				Count:    u.Count,
			}
		}
		return topUsersLoadedMsg(items)
	}
}

func (a *App) loadSimilar(topic string) tea.Cmd {
	return func() tea.Msg {
		groups, err := service.QuerySimilarComments(topic, 30)
		if err != nil {
			return errMsg(err)
		}
		items := make([]similarGroup, len(groups))
		for i, g := range groups {
			details := make([]similarDetail, len(g.Comments))
			for j, c := range g.Comments {
				details[j] = similarDetail{
					CommentId:  c.CommentId,
					Username:   c.Username,
					VideoTitle: c.VideoTitle,
					Bvid:       c.Bvid,
					CreatedAt:  c.CreatedAt,
				}
			}
			items[i] = similarGroup{
				Text:     g.Text,
				Count:    g.Count,
				Comments: details,
			}
		}
		return similarLoadedMsg(items)
	}
}

func (a *App) loadDashboard() tea.Cmd {
	return func() tea.Msg {
		service.EnsureDB()
		stats := service.GetDashboardStats()
		return dashboardLoadedMsg(stats)
	}
}

func (a *App) loadUserComments(uid uint, topic string) tea.Cmd {
	return func() tea.Msg {
		groups := service.GetUserCommentsInTopic(uid, topic)
		items := make([]userVideoGroup, 0, len(groups))
		for _, g := range groups {
			group := userVideoGroup{
				VideoTitle: g.Title,
				Bvid:       g.Bvid,
			}
			for _, c := range g.Comments {
				group.Comments = append(group.Comments, commentItem{
					Id:        c.Id,
					Content:   c.Content,
					Owner:     c.Owner.Name,
					OwnerUID:  c.Owner.Uid,
					CreatedAt: c.CreatedAt,
				})
			}
			items = append(items, group)
		}
		return userCommentsLoadedMsg(items)
	}
}

func (a *App) loadSignedUsers() tea.Cmd {
	return func() tea.Msg {
		service.EnsureDB()
		users, err := service.GetAllSignedUsers()
		if err != nil {
			return errMsg(err)
		}
		items := make([]signedUserItem, 0, len(users))
		for _, u := range users {
			username := fmt.Sprintf("UID:%d", u.UID)
			if user, err := service.UserRepo.ReadUser(u.UID); err == nil && user != nil {
				username = user.Name
			}
			items = append(items, signedUserItem{
				UID:         u.UID,
				Username:    username,
				Description: u.Description,
				LastViewed:  u.LastViewed,
			})
		}
		return signedUsersLoadedMsg(items)
	}
}
