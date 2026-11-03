// Package response 统一管理 API 的响应结构，沿用若依协议以兼容现有前端。
//
//	成功：{"code":200,"msg":"操作成功","data":...}
//	分页：{"code":200,"msg":"查询成功","rows":[...],"total":n}
//	失败：{"code":<业务码>,"msg":"..."}
package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Body 通用响应体。
type Body struct {
	Code  int    `json:"code"`
	Msg   string `json:"msg"`
	Data  any    `json:"data,omitempty"`
	Rows  any    `json:"rows,omitempty"`
	Total *int64 `json:"total,omitempty"`
}

// Success 返回成功。msg 为空时用「操作成功」。
func Success(c *gin.Context, data any, msg ...string) {
	m := "操作成功"
	if len(msg) > 0 && msg[0] != "" {
		m = msg[0]
	}
	c.JSON(http.StatusOK, Body{Code: 200, Msg: m, Data: data})
}

// Page 返回分页数据。rows 为 nil 时输出空数组而不是 null。
func Page(c *gin.Context, rows any, total int64) {
	if rows == nil {
		rows = []any{}
	}
	c.JSON(http.StatusOK, Body{Code: 200, Msg: "查询成功", Rows: rows, Total: &total})
}

// Error 返回业务错误。HTTP 状态保持 200，由 code 表达结果，与若依前端的拦截器一致；
// 401/403 同时设置 HTTP 状态，便于网关与日志识别。
func Error(c *gin.Context, code int, msg string) {
	status := http.StatusOK
	if code == http.StatusUnauthorized || code == http.StatusForbidden {
		status = code
	}
	c.AbortWithStatusJSON(status, Body{Code: code, Msg: msg})
}

// Fail 返回 500 业务错误。
func Fail(c *gin.Context, msg string) {
	Error(c, http.StatusInternalServerError, msg)
}
