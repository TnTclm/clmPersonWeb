package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// ============================================================
// 分页小工具 —— 全站列表接口共用
//
// 前端怎么用：
//   GET /api/talks?page=1&size=20
//   → 返回 {"code":200, "data":[...20条], "has_more":true}
//   has_more=true 表示后面还有，前端显示"加载更多"按钮
// ============================================================

const (
	defaultPageSize = 20 // 不传 size 时每页 20 条
	maxPageSize     = 50 // 一页最多 50 条（防止有人传 size=999999 把服务器拖死）
)

// pageParams 从查询参数里读"第几页、每页几条"，并做好兜底
// 返回值：page 从 1 开始，size 已做上下限约束
func pageParams(c *gin.Context) (page, size int) {
	// DefaultQuery：参数不存在时用默认值；Atoi 失败时 page/size 会是 0，下面兜底
	page, _ = strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ = strconv.Atoi(c.DefaultQuery("size", strconv.Itoa(defaultPageSize)))

	if page < 1 { // 传了 0、负数、乱码 → 当作第 1 页
		page = 1
	}
	if size < 1 || size > maxPageSize { // 太小或太大 → 回到默认 20
		size = defaultPageSize
	}
	return
}

// pageResult 统一的"带分页信息"的返回
// 设计取舍：data 仍然直接是数组（老前端代码不用改），
// 只是额外多一个 has_more 字段
func pageResult(c *gin.Context, list any, hasMore bool) {
	c.JSON(http.StatusOK, gin.H{
		"code":     200,
		"data":     list,
		"has_more": hasMore,
	})
}
