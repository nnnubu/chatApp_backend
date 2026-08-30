package model

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

// Book 图书基础信息表
type Book struct {
	ID          uint      `gorm:"primaryKey;autoIncrement" json:"-"`
	BookUID     string    `gorm:"size:36;not null;unique;comment:图书唯一标识" json:"bookUid"`
	Title       string    `gorm:"size:200;not null;comment:书名" json:"title"`
	Author      string    `gorm:"size:100;not null;default:'';comment:作者" json:"author"`
	Cover       string    `gorm:"size:255;default:'';comment:封面图片URL" json:"cover"`
	Intro       string    `gorm:"type:text;comment:内容简介" json:"intro"`
	Category    string    `gorm:"size:50;default:'';comment:分类（文学/科技/历史等）" json:"category"`
	ISBN        string    `gorm:"size:20;default:'';comment:ISBN号" json:"isbn"`
	Publisher   string    `gorm:"size:100;default:'';comment:出版社" json:"publisher"`
	PublishDate time.Time `gorm:"type:date;default:null;comment:出版日期" json:"publishDate"`
	TotalPages  int       `gorm:"default:0;comment:总页数" json:"totalPages"`
	WordCount   int       `gorm:"default:0;comment:字数" json:"wordCount"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updatedAt"`
}

// UserBook 用户书架关系表
type UserBook struct {
	ID          uint      `gorm:"primaryKey;autoIncrement" json:"-"`
	UID         string    `gorm:"size:36;not null;index:idx_uid_book;comment:用户UID" json:"uid"`
	BookUID     string    `gorm:"size:36;not null;index:idx_uid_book;comment:图书UID" json:"bookUid"`
	Status      int8      `gorm:"type:tinyint;default:1;comment:状态(1:想读,2:在读,3:已读)" json:"status"`
	CurrentPage int       `gorm:"default:0;comment:当前阅读页码" json:"currentPage"`
	Rating      int8      `gorm:"type:tinyint;default:0;comment:评分(0-5)" json:"rating"`
	Note        string    `gorm:"type:text;comment:读书笔记/备注" json:"note"`
	AddedAt     time.Time `gorm:"autoCreateTime" json:"addedAt"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updatedAt"`
}

// ============ Book 操作 ============

func CreateBook(ctx context.Context, txDB *gorm.DB, book *Book) error {
	return txDB.WithContext(ctx).Create(book).Error
}

func GetBookByUID(ctx context.Context, db *gorm.DB, bookUID string) (*Book, bool, error) {
	var book Book
	err := db.WithContext(ctx).Model(&Book{}).Where("book_uid = ?", bookUID).First(&book).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return &book, true, nil
}

func GetBookList(ctx context.Context, db *gorm.DB, category string, page, pageSize int) ([]Book, int64, error) {
	var books []Book
	var total int64
	query := db.WithContext(ctx).Model(&Book{})
	if category != "" {
		query = query.Where("category = ?", category)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&books).Error
	return books, total, err
}

func SearchBooks(ctx context.Context, db *gorm.DB, keyword string, page, pageSize int) ([]Book, int64, error) {
	var books []Book
	var total int64
	query := db.WithContext(ctx).Model(&Book{}).
		Where("title LIKE ? OR author LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&books).Error
	return books, total, err
}

func GetBookCategories(ctx context.Context, db *gorm.DB) ([]string, error) {
	var categories []string
	err := db.WithContext(ctx).Model(&Book{}).
		Distinct("category").
		Where("category != ''").
		Pluck("category", &categories).Error
	return categories, err
}

// ============ UserBook 操作 ============

func AddToShelf(ctx context.Context, txDB *gorm.DB, userBook *UserBook) error {
	return txDB.WithContext(ctx).Create(userBook).Error
}

func GetUserShelf(ctx context.Context, db *gorm.DB, uid string, status int8, page, pageSize int) ([]UserBook, int64, error) {
	var userBooks []UserBook
	var total int64
	query := db.WithContext(ctx).Model(&UserBook{}).Where("uid = ?", uid)
	if status > 0 {
		query = query.Where("status = ?", status)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	err := query.Order("added_at DESC").Offset(offset).Limit(pageSize).Find(&userBooks).Error
	return userBooks, total, err
}

func GetUserBook(ctx context.Context, db *gorm.DB, uid, bookUID string) (*UserBook, bool, error) {
	var ub UserBook
	err := db.WithContext(ctx).Model(&UserBook{}).
		Where("uid = ? AND book_uid = ?", uid, bookUID).
		First(&ub).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return &ub, true, nil
}

func UpdateUserBook(ctx context.Context, txDB *gorm.DB, uid, bookUID string, updates map[string]any) error {
	return txDB.WithContext(ctx).Model(&UserBook{}).
		Where("uid = ? AND book_uid = ?", uid, bookUID).
		Updates(updates).Error
}

func RemoveFromShelf(ctx context.Context, txDB *gorm.DB, uid, bookUID string) error {
	return txDB.WithContext(ctx).
		Where("uid = ? AND book_uid = ?", uid, bookUID).
		Delete(&UserBook{}).Error
}

// GetShelfBooks 获取用户书架的图书详情（联表查询）
func GetShelfBooks(ctx context.Context, db *gorm.DB, uid string, status int8, page, pageSize int) ([]Book, []UserBook, int64, error) {
	var userBooks []UserBook
	var total int64
	query := db.WithContext(ctx).Model(&UserBook{}).Where("uid = ?", uid)
	if status > 0 {
		query = query.Where("status = ?", status)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, nil, 0, err
	}
	offset := (page - 1) * pageSize
	if err := query.Order("added_at DESC").Offset(offset).Limit(pageSize).Find(&userBooks).Error; err != nil {
		return nil, nil, 0, err
	}
	if len(userBooks) == 0 {
		return []Book{}, userBooks, 0, nil
	}
	bookUIDs := make([]string, len(userBooks))
	for i, ub := range userBooks {
		bookUIDs[i] = ub.BookUID
	}
	var books []Book
	if err := db.WithContext(ctx).Model(&Book{}).Where("book_uid IN ?", bookUIDs).Find(&books).Error; err != nil {
		return nil, nil, 0, err
	}
	return books, userBooks, total, nil
}
