package implements

import (
	"github.com/Yoak3n/libi/shared/domain/model/table"
	"gorm.io/gorm"
)

type CommentRepository struct {
	db *gorm.DB
}

func NewCommentRepository(db *gorm.DB) *CommentRepository {
	return &CommentRepository{db: db}
}

func (r *CommentRepository) CreateComment(comment *table.CommentTable) error {
	return r.db.Create(comment).Error
}

func (r *CommentRepository) CreateCommentBatch(comments []*table.CommentTable) error {
	return r.db.Create(&comments).Error
}

func (r *CommentRepository) ReadComment(commentId uint) (*table.CommentTable, error) {
	var comment table.CommentTable
	if err := r.db.First(&comment, "comment_id = ?", commentId).Error; err != nil {
		return nil, err
	}
	return &comment, nil
}

func (r *CommentRepository) ReadCommentsByVideo(avid uint, limit int) ([]*table.CommentTable, error) {
	var comments []*table.CommentTable
	q := r.db.Where("video_avid = ?", avid)
	if limit > 0 {
		q = q.Limit(limit)
	}
	if err := q.Find(&comments).Error; err != nil {
		return nil, err
	}
	return comments, nil
}

func (r *CommentRepository) UpdateComment(comment *table.CommentTable) error {
	return r.db.Save(comment).Error
}

func (r *CommentRepository) DeleteComment(commentId uint) error {
	return r.db.Delete(&table.CommentTable{}, "comment_id = ?", commentId).Error
}
