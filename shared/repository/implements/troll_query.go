package implements

import (
	"fmt"
	"time"

	"github.com/Yoak3n/libi/shared/domain/model/schema"
	"github.com/Yoak3n/libi/shared/domain/model/table"
	"gorm.io/gorm"
)

type TrollQueryRepository struct {
	db *gorm.DB
}

func NewTrollQueryRepository(db *gorm.DB) *TrollQueryRepository {
	return &TrollQueryRepository{db: db}
}

func (r *TrollQueryRepository) QuerySimilarComments(topic string, n int) ([]schema.SimilarCommentResult, error) {
	var comments []schema.SimilarCommentResult
	subQuery := r.db.Table("comment_tables AS c").
		Select("SUBSTR(c.text, 1, 30) AS text, COUNT(*) AS count, GROUP_CONCAT(c.comment_id, ', ') AS comment_ids").
		Joins("INNER JOIN video_tables v ON c.video_avid = v.avid").
		Where("c.text IS NOT NULL AND LENGTH(c.text) >= 10 AND v.topic = ?", topic).
		Group("SUBSTR(c.text, 1, 30)").
		Having("COUNT(*) > 1")

	query := r.db.Table("(?) AS similar_groups", subQuery).
		Order("count DESC, text").
		Limit(n)

	if err := query.Scan(&comments).Error; err != nil {
		return nil, err
	}
	return comments, nil
}

func (r *TrollQueryRepository) QueryTopNUserInTopic(topic string, n int) ([]schema.UserQuery, error) {
	var ret []schema.UserQuery
	query := `
	SELECT
		u.uid, u.name AS username, u.avatar,
		COUNT(c.comment_id) AS count
	FROM
		user_tables u
		INNER JOIN comment_tables c ON u.uid = c.owner
		INNER JOIN video_tables v ON c.video_avid = v.avid
	WHERE
		v.topic = ?
	GROUP BY
		u.uid
	ORDER BY
		count DESC
	LIMIT ?`
	err := r.db.Raw(query, topic, n).Scan(&ret).Error
	return ret, err
}

// commentScanRow avoids GORM field conflicts when scanning joined comment+user rows.
// Using SELECT * with two embedded structs that share column names (created_at, etc.)
// causes GORM to mis-map fields, resulting in CommentId=0.
type commentScanRow struct {
	CommentId     uint      `gorm:"column:comment_id"`
	Text          string    `gorm:"column:text"`
	Owner         uint      `gorm:"column:owner"`
	VideoAvid     uint      `gorm:"column:video_avid"`
	ParentComment uint      `gorm:"column:parent_comment"`
	Like          uint64    `gorm:"column:like"`
	CommentTime   time.Time `gorm:"column:comment_time"`
	Uid           uint      `gorm:"column:uid"`
	Name          string    `gorm:"column:name"`
	Avatar        string    `gorm:"column:avatar"`
}

func (r *TrollQueryRepository) GetCommentsByVideo(avid uint) schema.CommentGroupByVideo {
	videoData := r.getVideoByAvid(avid)

	var rootComments []commentScanRow
	query1 := `
	SELECT c.comment_id, c.text, c.owner, c.video_avid, c.parent_comment, c.like, c.comment_time,
	       u.uid, u.name, u.avatar
	FROM comment_tables c
	INNER JOIN user_tables u ON c.owner = u.uid
	WHERE c.video_avid = ? AND c.parent_comment = 0 AND c.deleted_at IS NULL`
	r.db.Raw(query1, avid).Scan(&rootComments)

	var subComments []commentScanRow
	query2 := `
	SELECT c.comment_id, c.text, c.owner, c.video_avid, c.parent_comment, c.like, c.comment_time,
	       u.uid, u.name, u.avatar
	FROM comment_tables c
	INNER JOIN user_tables u ON c.owner = u.uid
	WHERE c.video_avid = ? AND c.parent_comment != 0 AND c.deleted_at IS NULL`
	r.db.Raw(query2, avid).Scan(&subComments)

	commentMap := make(map[uint]schema.CommentData)
	for _, item := range rootComments {
		commentMap[item.CommentId] = schema.CommentData{
			Id:        item.CommentId,
			Content:   item.Text,
			Like:      item.Like,
			CreatedAt: item.CommentTime,
			Owner: schema.Author{
				Uid:    item.Uid,
				Name:   item.Name,
				Avatar: item.Avatar,
			},
			Children: make([]schema.CommentData, 0),
		}
	}
	for _, item := range subComments {
		if parent, ok := commentMap[item.ParentComment]; ok {
			child := schema.CommentData{
				Id:        item.CommentId,
				Content:   item.Text,
				Like:      item.Like,
				CreatedAt: item.CommentTime,
				Owner: schema.Author{
					Uid:    item.Uid,
					Name:   item.Name,
					Avatar: item.Avatar,
				},
			}
			parent.Children = append(parent.Children, child)
			commentMap[item.ParentComment] = parent
		}
	}

	comments := make([]schema.CommentData, 0, len(commentMap))
	for _, c := range commentMap {
		comments = append(comments, c)
	}
	return schema.CommentGroupByVideo{
		VideoData: videoData,
		Comments:  comments,
	}
}

func (r *TrollQueryRepository) SearchCommentWithKeyword(keyword string) []schema.CommentData {
	query := `
	SELECT c.comment_id, c.text, c.owner, c.video_avid, c.parent_comment, c.like, c.comment_time,
	       u.uid, u.name, u.avatar
	FROM comment_tables c
	INNER JOIN user_tables u ON c.owner = u.uid
	WHERE c.text LIKE ? AND c.deleted_at IS NULL`
	var result []commentScanRow
	if err := r.db.Raw(query, "%"+keyword+"%").Scan(&result).Error; err != nil {
		return nil
	}
	comments := make([]schema.CommentData, 0, len(result))
	for _, item := range result {
		comments = append(comments, schema.CommentData{
			Id:        item.CommentId,
			Content:   item.Text,
			Like:      item.Like,
			CreatedAt: item.CommentTime,
			Owner: schema.Author{
				Uid:    item.Uid,
				Name:   item.Name,
				Avatar: item.Avatar,
			},
		})
	}
	return comments
}

type videoWithUser struct {
	table.VideoTable
	table.UserTable
	Count int `json:"count"`
}

func (r *TrollQueryRepository) GetVideosByTopic(topic string) []schema.VideoWithCommentCount {
	var result []videoWithUser
	query := `
	SELECT v.*, u.*, COUNT(c.comment_id) AS count
	FROM comment_tables c
	INNER JOIN video_tables v ON c.video_avid = v.avid
	LEFT JOIN user_tables u ON v.owner = u.uid
	WHERE v.topic = ? AND v.deleted_at IS NULL
	GROUP BY v.avid, u.uid`
	if err := r.db.Raw(query, topic).Scan(&result).Error; err != nil {
		return nil
	}
	videos := make([]schema.VideoWithCommentCount, 0, len(result))
	for _, v := range result {
		videos = append(videos, schema.VideoWithCommentCount{
			VideoData: schema.VideoData{
				Avid:        v.VideoTable.Avid,
				Bvid:        v.VideoTable.Bvid,
				Title:       v.VideoTable.Title,
				Topic:       v.VideoTable.Topic,
				Description: v.VideoTable.Description,
				Cover:       v.VideoTable.Cover,
				Author: schema.Author{
					Uid:      v.UserTable.UID,
					Name:     v.UserTable.Name,
					Avatar:   v.UserTable.Avatar,
				},
			},
			Count:    v.Count,
			UpdateAt: v.VideoTable.UpdatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return videos
}

func (r *TrollQueryRepository) GetAllTopicsList() []schema.TopicsData {
	var videos []table.VideoTable
	r.db.Find(&videos)
	topics := make(map[string]int64)
	for _, v := range videos {
		topics[v.Topic]++
	}
	ret := make([]schema.TopicsData, 0, len(topics))
	for k, v := range topics {
		ret = append(ret, schema.TopicsData{Name: k, Count: v})
	}
	return ret
}

func (r *TrollQueryRepository) GetDashboardStats() schema.DashboardStats {
	var stats schema.DashboardStats
	r.db.Raw("SELECT COUNT(DISTINCT topic) FROM video_tables").Scan(&stats.Topics)
	r.db.Model(&table.VideoTable{}).Count(&stats.Videos)
	r.db.Model(&table.UserTable{}).Count(&stats.Users)
	r.db.Model(&table.CommentTable{}).Count(&stats.Comments)
	return stats
}

type commentVideoScanRow struct {
	CommentId     uint      `gorm:"column:comment_id"`
	Text          string    `gorm:"column:text"`
	VideoAvid     uint      `gorm:"column:video_avid"`
	ParentComment uint      `gorm:"column:parent_comment"`
	CommentTime   time.Time `gorm:"column:comment_time"`
	VideoTitle    string    `gorm:"column:video_title"`
	Bvid          string    `gorm:"column:bvid"`
	VideoOwner    uint      `gorm:"column:video_owner"`
	Topic         string    `gorm:"column:topic"`
	Uid           uint      `gorm:"column:uid"`
	Name          string    `gorm:"column:name"`
	Avatar        string    `gorm:"column:avatar"`
}

func (r *TrollQueryRepository) GetCommentsWithVideoFromUserInTopic(uid uint, topic string) []schema.CommentGroupByVideo {
	var rows []commentVideoScanRow
	query := `
	SELECT c.comment_id, c.text, c.video_avid, c.parent_comment, c.comment_time,
	       v.title AS video_title, v.bvid, v.owner AS video_owner, v.topic,
	       u.uid, u.name, u.avatar
	FROM comment_tables c
	INNER JOIN video_tables v ON c.video_avid = v.avid
	INNER JOIN user_tables u ON c.owner = u.uid
	WHERE c.owner = ? AND c.deleted_at IS NULL`
	if topic == "" || topic == "all" || topic == "*" {
		r.db.Raw(query, uid).Scan(&rows)
	} else {
		query += " AND v.topic = ?"
		r.db.Raw(query, uid, topic).Scan(&rows)
	}
	r.db.Model(&table.SignedUserTable{}).Where("uid = ?", uid).Update("last_viewed", time.Now())

	groupMap := make(map[uint]schema.CommentGroupByVideo)
	for _, item := range rows {
		current, ok := groupMap[item.VideoAvid]
		if !ok {
			current = schema.CommentGroupByVideo{
				VideoData: schema.VideoData{
					Avid: item.VideoAvid,
					Bvid: item.Bvid,
					Title: item.VideoTitle,
					Topic: item.Topic,
					Author: schema.Author{
						Uid: item.VideoOwner,
					},
				},
			}
		}
		c := schema.CommentData{
			Id:        item.CommentId,
			Content:   item.Text,
			CreatedAt: item.CommentTime,
			Owner: schema.Author{
				Uid:    item.Uid,
				Name:   item.Name,
				Avatar: item.Avatar,
			},
		}
		current.Comments = append(current.Comments, c)
		groupMap[item.VideoAvid] = current
	}
	groups := make([]schema.CommentGroupByVideo, 0, len(groupMap))
	for _, v := range groupMap {
		groups = append(groups, v)
	}
	return groups
}

func (r *TrollQueryRepository) UpdateTopic(topic string, newTopic string) error {
	return r.db.Model(&table.VideoTable{}).Where("topic = ?", topic).Update("topic", newTopic).Error
}

func (r *TrollQueryRepository) DeleteTopic(topic string) error {
	var avids []uint
	if err := r.db.Model(&table.VideoTable{}).Where("topic = ?", topic).Pluck("avid", &avids).Error; err != nil {
		return err
	}
	if len(avids) > 0 {
		if err := r.db.Where("video_avid IN ?", avids).Delete(&table.CommentTable{}).Error; err != nil {
			return err
		}
	}
	return r.db.Delete(&table.VideoTable{}, "topic = ?", topic).Error
}

func (r *TrollQueryRepository) UpdateTopicOfVideos(avid []uint, topic string) error {
	return r.db.Model(&table.VideoTable{}).Where("avid IN ?", avid).Update("topic", topic).Error
}

func (r *TrollQueryRepository) DeleteVideos(avidList []uint) error {
	if err := r.db.Where("avid IN ?", avidList).Delete(&table.VideoTable{}).Error; err != nil {
		return err
	}
	return r.db.Where("video_avid IN ?", avidList).Delete(&table.CommentTable{}).Error
}

func (r *TrollQueryRepository) GetSignedUserRecord() ([]*table.UserTable, error) {
	var ids []uint
	if err := r.db.Model(&table.SignedUserTable{}).Pluck("uid", &ids).Error; err != nil {
		return nil, err
	}
	var users []*table.UserTable
	if err := r.db.Where("uid IN ?", ids).Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

func (r *TrollQueryRepository) GetSignedUserRecordByUID(uid uint) (*table.SignedUserTable, error) {
	var record table.SignedUserTable
	if err := r.db.Where("uid = ?", uid).First(&record).Error; err != nil {
		return nil, err
	}
	return &record, nil
}

func (r *TrollQueryRepository) getVideoByAvid(avid uint) schema.VideoData {
	var result videoWithUser
	query := `
	SELECT v.*, u.*
	FROM video_tables v
	INNER JOIN user_tables u ON v.owner = u.uid
	WHERE v.avid = ? AND v.deleted_at IS NULL`
	if err := r.db.Raw(query, avid).Scan(&result).Error; err != nil {
		return schema.VideoData{}
	}
	return schema.VideoData{
		Avid:        result.VideoTable.Avid,
		Bvid:        result.VideoTable.Bvid,
		Title:       result.VideoTable.Title,
		Topic:       result.VideoTable.Topic,
		Description: result.VideoTable.Description,
		Cover:       result.VideoTable.Cover,
		Author: schema.Author{
			Uid:      result.UserTable.UID,
			Name:     result.UserTable.Name,
			Avatar:   result.UserTable.Avatar,
		},
	}
}

func (r *TrollQueryRepository) GetUserCommentsByRange(uid uint, name string, rangeType string, rangeData string) []schema.CommentGroupByVideo {
	var rows []commentVideoScanRow
	extraQuery := ""
	switch rangeType {
	case "video":
		extraQuery = fmt.Sprintf(" AND v.bvid IN (%s)", rangeData)
	case "topic":
		extraQuery = fmt.Sprintf(" AND v.topic IN (%s)", rangeData)
	}
	if uid != 0 && name == "" {
		q := `
		SELECT c.comment_id, c.text, c.video_avid, c.parent_comment, c.comment_time,
		       v.title AS video_title, v.bvid, v.owner AS video_owner, v.topic,
		       u.uid, u.name, u.avatar
		FROM comment_tables c
		INNER JOIN video_tables v ON c.video_avid = v.avid
		INNER JOIN user_tables u ON c.owner = u.uid
		WHERE c.owner = ? AND c.deleted_at IS NULL` + extraQuery
		r.db.Raw(q, uid).Scan(&rows)
	} else if uid == 0 && name != "" {
		q := `
		SELECT c.comment_id, c.text, c.video_avid, c.parent_comment, c.comment_time,
		       v.title AS video_title, v.bvid, v.owner AS video_owner, v.topic,
		       u.uid, u.name, u.avatar
		FROM user_tables u
		INNER JOIN comment_tables c ON u.uid = c.owner
		INNER JOIN video_tables v ON c.video_avid = v.avid
		WHERE u.name = ? AND c.deleted_at IS NULL` + extraQuery
		r.db.Raw(q, name).Scan(&rows)
	}

	groupMap := make(map[uint]schema.CommentGroupByVideo)
	for _, item := range rows {
		current, ok := groupMap[item.VideoAvid]
		if !ok {
			current = schema.CommentGroupByVideo{
				VideoData: schema.VideoData{
					Avid: item.VideoAvid,
					Bvid: item.Bvid,
					Title: item.VideoTitle,
					Topic: item.Topic,
					Author: schema.Author{
						Uid: item.VideoOwner,
					},
				},
			}
		}
		c := schema.CommentData{
			Id:        item.CommentId,
			Content:   item.Text,
			CreatedAt: item.CommentTime,
			Owner: schema.Author{
				Uid:    item.Uid,
				Name:   item.Name,
				Avatar: item.Avatar,
			},
		}
		current.Comments = append(current.Comments, c)
		groupMap[item.VideoAvid] = current
	}
	groups := make([]schema.CommentGroupByVideo, 0, len(groupMap))
	for _, v := range groupMap {
		groups = append(groups, v)
	}
	return groups
}
