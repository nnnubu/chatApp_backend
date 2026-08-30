package book

import (
	"ChatApp/dto"
	"ChatApp/model"
	"ChatApp/utils"
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

type BookService struct{}

func NewBookService() *BookService {
	return &BookService{}
}

// PullBooks 拉取图书列表（支持分类筛选和搜索）
func (bs *BookService) PullBooks(ctx context.Context, db *gorm.DB, category, keyword string, page, pageSize int) ([]dto.BookResp, int64, bool, error) {
	var books []model.Book
	var total int64
	var err error

	if keyword != "" {
		books, total, err = model.SearchBooks(ctx, db, keyword, page, pageSize)
	} else {
		books, total, err = model.GetBookList(ctx, db, category, page, pageSize)
	}
	if err != nil {
		return nil, 0, false, errors.New("系统繁忙，请稍后再试")
	}

	result := make([]dto.BookResp, 0, len(books))
	for _, b := range books {
		result = append(result, bookToResp(&b))
	}
	hasMore := int64(page*pageSize) < total
	return result, total, hasMore, nil
}

// GetBookDetail 获取图书详情
func (bs *BookService) GetBookDetail(ctx context.Context, db *gorm.DB, bookUID string) (*dto.BookResp, error) {
	book, exists, err := model.GetBookByUID(ctx, db, bookUID)
	if err != nil {
		return nil, errors.New("系统繁忙，请稍后再试")
	}
	if !exists {
		return nil, errors.New("图书不存在")
	}
	resp := bookToResp(book)
	return &resp, nil
}

// GetCategories 获取图书分类列表
func (bs *BookService) GetCategories(ctx context.Context, db *gorm.DB) ([]string, error) {
	categories, err := model.GetBookCategories(ctx, db)
	if err != nil {
		return nil, errors.New("系统繁忙，请稍后再试")
	}
	return categories, nil
}

// AddToShelf 添加图书到书架
func (bs *BookService) AddToShelf(ctx context.Context, db *gorm.DB, uid, bookUID string, status int8) error {
	// 校验图书存在
	_, exists, err := model.GetBookByUID(ctx, db, bookUID)
	if err != nil {
		return errors.New("系统繁忙，请稍后再试")
	}
	if !exists {
		return errors.New("图书不存在")
	}

	// 检查是否已在书架
	_, exists, err = model.GetUserBook(ctx, db, uid, bookUID)
	if err != nil {
		return errors.New("系统繁忙，请稍后再试")
	}
	if exists {
		return errors.New("该书已在书架中")
	}

	if status < 1 || status > 3 {
		status = 1
	}

	userBook := &model.UserBook{
		UID:     uid,
		BookUID: bookUID,
		Status:  status,
	}
	return model.AddToShelf(ctx, db, userBook)
}

// PullShelf 拉取用户书架
func (bs *BookService) PullShelf(ctx context.Context, db *gorm.DB, uid string, status int8, page, pageSize int) ([]dto.ShelfBookResp, int64, bool, error) {
	books, userBooks, total, err := model.GetShelfBooks(ctx, db, uid, status, page, pageSize)
	if err != nil {
		return nil, 0, false, errors.New("系统繁忙，请稍后再试")
	}

	bookMap := make(map[string]model.Book, len(books))
	for _, b := range books {
		bookMap[b.BookUID] = b
	}

	result := make([]dto.ShelfBookResp, 0, len(userBooks))
	for _, ub := range userBooks {
		book, ok := bookMap[ub.BookUID]
		if !ok {
			continue
		}
		result = append(result, dto.ShelfBookResp{
			BookResp:    bookToResp(&book),
			Status:      ub.Status,
			CurrentPage: ub.CurrentPage,
			Rating:      ub.Rating,
			Note:        ub.Note,
			AddedAt:     ub.AddedAt.Format(time.RFC3339),
		})
	}
	hasMore := int64(page*pageSize) < total
	return result, total, hasMore, nil
}

// UpdateShelf 更新书架信息（阅读进度、评分、笔记、状态）
func (bs *BookService) UpdateShelf(ctx context.Context, db *gorm.DB, uid, bookUID string, updates map[string]any) error {
	_, exists, err := model.GetUserBook(ctx, db, uid, bookUID)
	if err != nil {
		return errors.New("系统繁忙，请稍后再试")
	}
	if !exists {
		return errors.New("该书不在书架中")
	}
	if len(updates) == 0 {
		return nil
	}
	return model.UpdateUserBook(ctx, db, uid, bookUID, updates)
}

// RemoveFromShelf 从书架移除
func (bs *BookService) RemoveFromShelf(ctx context.Context, db *gorm.DB, uid, bookUID string) error {
	return model.RemoveFromShelf(ctx, db, uid, bookUID)
}

// bookToResp 模型转响应 DTO
func bookToResp(b *model.Book) dto.BookResp {
	publishDate := ""
	if !b.PublishDate.IsZero() {
		publishDate = b.PublishDate.Format("2006-01-02")
	}
	return dto.BookResp{
		BookUID:     b.BookUID,
		Title:       b.Title,
		Author:      b.Author,
		Cover:       b.Cover,
		Intro:       b.Intro,
		Category:    b.Category,
		ISBN:        b.ISBN,
		Publisher:   b.Publisher,
		PublishDate: publishDate,
		TotalPages:  b.TotalPages,
		WordCount:   b.WordCount,
	}
}

// GenerateBookUID 生成图书 UID
func GenerateBookUID() string {
	return utils.GenAutoUuid()
}
