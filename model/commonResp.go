package model

type CommonResp struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"` // omitempty 当序列化转 JSON 时，如果该字段值为 nil / 空 / 零值，直接忽略这个 key，不输出到 JSON
}
