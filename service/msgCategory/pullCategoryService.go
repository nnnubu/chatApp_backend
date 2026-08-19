package msgCategory

import (
	"ChatApp/dto"
	"ChatApp/model"
	"context"
	"errors"

	"gorm.io/gorm"
)

type PullMsgCategory struct{}

func NewPullMsgCategory() *PullMsgCategory {
	return &PullMsgCategory{}
}

func (nmc *PullMsgCategory) PullCategory(ctx context.Context, db *gorm.DB, uid string) (*[]dto.PullCategoryResp, error) {
	categoryList, err := model.GetCategoryByUID(ctx, db, uid)
	if err != nil {
		return nil, errors.New("系统繁忙，请稍后再试")
	}
	if categoryList == nil {
		return nil, errors.New("系统繁忙，请稍后重试")
	}
	var list []dto.PullCategoryResp
	for _, category := range categoryList {
		list = append(list, dto.PullCategoryResp{
			CategoryName: category.CategoryName,
			CategoryType: category.CategoryType,
			Sort:         category.Sort,
		})
	}
	return &list, nil
}
