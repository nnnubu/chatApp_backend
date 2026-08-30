package dto

// 图书相关请求/响应 DTO

// PullBooksReq 拉取图书列表请求
type PullBooksReq struct {
	Category string `form:"category"`
	Keyword  string `form:"keyword"`
	Page     int    `form:"page"`
	PageSize int    `form:"pageSize"`
}

// BookResp 图书信息响应
type BookResp struct {
	BookUID     string `json:"bookUid"`
	Title       string `json:"title"`
	Author      string `json:"author"`
	Cover       string `json:"cover"`
	Intro       string `json:"intro"`
	Category    string `json:"category"`
	ISBN        string `json:"isbn"`
	Publisher   string `json:"publisher"`
	PublishDate string `json:"publishDate"`
	TotalPages  int    `json:"totalPages"`
	WordCount   int    `json:"wordCount"`
}

// ShelfBookResp 书架图书响应（含阅读状态）
type ShelfBookResp struct {
	BookResp
	Status      int8   `json:"status"`
	CurrentPage int    `json:"currentPage"`
	Rating      int8   `json:"rating"`
	Note        string `json:"note"`
	AddedAt     string `json:"addedAt"`
}

// AddToShelfReq 添加到书架请求
type AddToShelfReq struct {
	BookUID string `json:"bookUid" binding:"required"`
	Status  int8   `json:"status"` // 1:想读 2:在读 3:已读
}

// UpdateShelfReq 更新书架信息请求
type UpdateShelfReq struct {
	BookUID     string `json:"bookUid" binding:"required"`
	Status      int8   `json:"status"`
	CurrentPage int    `json:"currentPage"`
	Rating      int8   `json:"rating"`
	Note        string `json:"note"`
}

// PullShelfReq 拉取书架请求
type PullShelfReq struct {
	Status   int8 `form:"status"`
	Page     int  `form:"page"`
	PageSize int  `form:"pageSize"`
}
