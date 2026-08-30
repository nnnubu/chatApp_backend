package book

import (
	"ChatApp/dto"
	"ChatApp/model"
	"ChatApp/service/book"
	"ChatApp/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

var bookService = book.NewBookService()

// PullBooks 拉取图书列表（支持分类/搜索）
func PullBooks(c *gin.Context) {
	var req dto.PullBooksReq
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusOK, model.CommonResp{Code: 400, Message: "参数错误"})
		return
	}
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 {
		req.PageSize = 20
	}
	if req.PageSize > 100 {
		req.PageSize = 100
	}

	reqCtx, db, _, err := utils.GetRequestSource(c)
	if err != nil {
		c.JSON(http.StatusOK, model.CommonResp{Code: 500, Message: err.Error()})
		return
	}

	list, total, hasMore, err := bookService.PullBooks(reqCtx, db, req.Category, req.Keyword, req.Page, req.PageSize)
	if err != nil {
		c.JSON(http.StatusOK, model.CommonResp{Code: 400, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, model.CommonResp{
		Code:    200,
		Message: "success",
		Data: map[string]any{
			"books":    list,
			"total":    total,
			"page":     req.Page,
			"pageSize": req.PageSize,
			"hasMore":  hasMore,
		},
	})
}

// GetBookDetail 获取图书详情
func GetBookDetail(c *gin.Context) {
	bookUID := c.Query("bookUid")
	if bookUID == "" {
		c.JSON(http.StatusOK, model.CommonResp{Code: 400, Message: "bookUid 不能为空"})
		return
	}

	reqCtx, db, _, err := utils.GetRequestSource(c)
	if err != nil {
		c.JSON(http.StatusOK, model.CommonResp{Code: 500, Message: err.Error()})
		return
	}

	detail, err := bookService.GetBookDetail(reqCtx, db, bookUID)
	if err != nil {
		c.JSON(http.StatusOK, model.CommonResp{Code: 400, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, model.CommonResp{Code: 200, Message: "success", Data: detail})
}

// GetCategories 获取图书分类列表
func GetCategories(c *gin.Context) {
	reqCtx, db, _, err := utils.GetRequestSource(c)
	if err != nil {
		c.JSON(http.StatusOK, model.CommonResp{Code: 500, Message: err.Error()})
		return
	}
	categories, err := bookService.GetCategories(reqCtx, db)
	if err != nil {
		c.JSON(http.StatusOK, model.CommonResp{Code: 400, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, model.CommonResp{Code: 200, Message: "success", Data: map[string]any{"categories": categories}})
}

// AddToShelf 添加图书到书架
func AddToShelf(c *gin.Context) {
	var req dto.AddToShelfReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, model.CommonResp{Code: 400, Message: "参数错误"})
		return
	}

	uidAny, hasUid := c.Get("uid")
	if !hasUid {
		c.JSON(http.StatusOK, model.CommonResp{Code: 401, Message: "登录态缺失"})
		return
	}
	uid, _ := uidAny.(string)

	reqCtx, db, _, err := utils.GetRequestSource(c)
	if err != nil {
		c.JSON(http.StatusOK, model.CommonResp{Code: 500, Message: err.Error()})
		return
	}

	if err := bookService.AddToShelf(reqCtx, db, uid, req.BookUID, req.Status); err != nil {
		c.JSON(http.StatusOK, model.CommonResp{Code: 400, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, model.CommonResp{Code: 200, Message: "已添加到书架"})
}

// PullShelf 拉取用户书架
func PullShelf(c *gin.Context) {
	var req dto.PullShelfReq
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusOK, model.CommonResp{Code: 400, Message: "参数错误"})
		return
	}
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 {
		req.PageSize = 20
	}
	if req.PageSize > 100 {
		req.PageSize = 100
	}

	uidAny, hasUid := c.Get("uid")
	if !hasUid {
		c.JSON(http.StatusOK, model.CommonResp{Code: 401, Message: "登录态缺失"})
		return
	}
	uid, _ := uidAny.(string)

	reqCtx, db, _, err := utils.GetRequestSource(c)
	if err != nil {
		c.JSON(http.StatusOK, model.CommonResp{Code: 500, Message: err.Error()})
		return
	}

	list, total, hasMore, err := bookService.PullShelf(reqCtx, db, uid, req.Status, req.Page, req.PageSize)
	if err != nil {
		c.JSON(http.StatusOK, model.CommonResp{Code: 400, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, model.CommonResp{
		Code:    200,
		Message: "success",
		Data: map[string]any{
			"books":    list,
			"total":    total,
			"page":     req.Page,
			"pageSize": req.PageSize,
			"hasMore":  hasMore,
		},
	})
}

// UpdateShelf 更新书架信息（阅读进度/评分/笔记/状态）
func UpdateShelf(c *gin.Context) {
	var req dto.UpdateShelfReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, model.CommonResp{Code: 400, Message: "参数错误"})
		return
	}

	uidAny, hasUid := c.Get("uid")
	if !hasUid {
		c.JSON(http.StatusOK, model.CommonResp{Code: 401, Message: "登录态缺失"})
		return
	}
	uid, _ := uidAny.(string)

	reqCtx, db, _, err := utils.GetRequestSource(c)
	if err != nil {
		c.JSON(http.StatusOK, model.CommonResp{Code: 500, Message: err.Error()})
		return
	}

	updates := make(map[string]any)
	if req.Status >= 1 && req.Status <= 3 {
		updates["status"] = req.Status
	}
	if req.CurrentPage > 0 {
		updates["current_page"] = req.CurrentPage
	}
	if req.Rating > 0 && req.Rating <= 5 {
		updates["rating"] = req.Rating
	}
	if req.Note != "" {
		updates["note"] = req.Note
	}

	if err := bookService.UpdateShelf(reqCtx, db, uid, req.BookUID, updates); err != nil {
		c.JSON(http.StatusOK, model.CommonResp{Code: 400, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, model.CommonResp{Code: 200, Message: "更新成功"})
}

// RemoveFromShelf 从书架移除
func RemoveFromShelf(c *gin.Context) {
	bookUID := c.Query("bookUid")
	if bookUID == "" {
		c.JSON(http.StatusOK, model.CommonResp{Code: 400, Message: "bookUid 不能为空"})
		return
	}

	uidAny, hasUid := c.Get("uid")
	if !hasUid {
		c.JSON(http.StatusOK, model.CommonResp{Code: 401, Message: "登录态缺失"})
		return
	}
	uid, _ := uidAny.(string)

	reqCtx, db, _, err := utils.GetRequestSource(c)
	if err != nil {
		c.JSON(http.StatusOK, model.CommonResp{Code: 500, Message: err.Error()})
		return
	}

	if err := bookService.RemoveFromShelf(reqCtx, db, uid, bookUID); err != nil {
		c.JSON(http.StatusOK, model.CommonResp{Code: 400, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, model.CommonResp{Code: 200, Message: "已从书架移除"})
}
