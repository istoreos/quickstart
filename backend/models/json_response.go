package models

// swagger:model jsonResponse
type JSONResponse struct {

	// 具体的错误码，由 SDK 来翻译成具体语言的错误
	// Example: 登录错误，请重新登录
	Error string `json:"error,omitempty"`

	// 错误码所属于范围，比如 system 等
	// Enum: [system network fileshare raid]
	Scope string `json:"scope,omitempty"`

	// 0 表示 OK，小于 0 表示错误。大于 0 得看接口具体说明
	// Example: -1
	Success int64 `json:"success,omitempty"`
}
