package handler

import (
	"blog/internal/config"
	"blog/internal/database"
	"blog/internal/model"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// ============================================================
// 相册模块 —— 博主专属：一条内容 = 一段说明 + 多个照片/视频
// 一次可以多选文件上传，它们同属一张卡片（像发一条带九宫格的说说）
// 权限：发布/编辑/删除只有博主；浏览公开
// ============================================================

// isOwner 当前登录人是不是博主（相册的独家发布权）
func isOwner(c *gin.Context) bool {
	return c.GetUint("user_id") == uint(config.Cfg.OwnerID)
}

// CreateAlbumPost 发布相册内容（博主专属）
// POST /api/album  multipart：title(说明，可空100字内) + files(1~9个文件，多选)
// 流程：先校验全部文件 → 建内容主体 → 逐个落盘 → 媒体批量入库
func CreateAlbumPost(c *gin.Context) {
	// 门槛：必须是博主
	if !isOwner(c) {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "msg": "相册是博主的专属地盘哦"})
		return
	}
	// 说明文字（可空）
	title := strings.TrimSpace(c.PostForm("title"))
	if len([]rune(title)) > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "说明最长100字"})
		return
	}
	// MultipartForm：一次拿整个表单，多选的文件都在 File["files"] 里
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "请选择文件"})
		return
	}
	files := form.File["files"]
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "至少选一个照片或视频"})
		return
	}
	if len(files) > 9 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "一次最多9个文件"})
		return
	}

	// ---------- 第一遍：全部文件先校验 ----------
	// 任何一个不合法就整个拒绝，不留"发了一半"的半成品
	for _, file := range files {
		ext := strings.ToLower(filepath.Ext(file.Filename))
		switch {
		case isImageExt(ext):
			if file.Size > 10<<20 { // 单张图片 10MB
				c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "单张图片不能超过 10MB"})
				return
			}
		case isVideoExt(ext):
			if file.Size > 200<<20 { // 单个视频 200MB
				c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "单个视频不能超过 200MB"})
				return
			}
		default:
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "只支持图片(jpg/png/gif/webp)或视频(mp4/webm/mov)"})
			return
		}
	}

	// ---------- 第二遍：建主体 → 逐个落盘 → 收集媒体记录 ----------
	post := model.AlbumPost{UserID: c.GetUint("user_id"), Title: title}
	if err := database.DB.Create(&post).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "发布失败，请稍后再试"})
		return
	}

	medias := []model.AlbumItem{}
	for _, file := range files {
		ext := strings.ToLower(filepath.Ext(file.Filename))
		var mediaType, name string
		if isImageExt(ext) {
			mediaType = "image"
			name = fmt.Sprintf("album_img_%d_%d%s", c.GetUint("user_id"), time.Now().UnixNano(), ext)
		} else {
			mediaType = "video"
			name = fmt.Sprintf("album_vid_%d_%d%s", c.GetUint("user_id"), time.Now().UnixNano(), ext)
		}
		savePath := filepath.Join("uploads", "album", name)
		// 子目录不存在就自动创建
		if err := os.MkdirAll(filepath.Dir(savePath), 0755); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "创建目录失败"})
			return
		}
		if err := c.SaveUploadedFile(file, savePath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "保存文件失败"})
			return
		}
		medias = append(medias, model.AlbumItem{
			PostID:    post.ID,
			MediaType: mediaType,
			URL:       "/uploads/album/" + name,
		})
	}
	// 批量插入：Create 传切片 = 一条 SQL 插多行
	if err := database.DB.Create(&medias).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "媒体入库失败"})
		return
	}

	TouchActivity(c.GetUint("user_id")) // 活跃度+1（热力图）
	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "发布成功",
		"data": gin.H{"post": post, "medias": medias},
	})
}

// isImageExt / isVideoExt：后缀白名单（上传校验用）
func isImageExt(ext string) bool {
	return map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true}[ext]
}

func isVideoExt(ext string) bool {
	return map[string]bool{".mp4": true, ".webm": true, ".mov": true}[ext]
}

// ListAlbum 相册列表（公开）：一条 = 一段说明 + 它的所有媒体
// GET /api/album
func ListAlbum(c *gin.Context) {
	posts := []model.AlbumPost{}
	database.DB.Order("created_at DESC").Limit(50).Find(&posts)

	items := []model.AlbumItem{}
	database.DB.Order("created_at ASC").Find(&items)

	// 内存组装：把媒体按 post_id 归堆挂到各自的 post 上
	data := []gin.H{}
	for _, p := range posts {
		medias := []model.AlbumItem{} // 空切片初始化，防 null
		for _, it := range items {
			if it.PostID == p.ID {
				medias = append(medias, it)
			}
		}
		data = append(data, gin.H{
			"id":         p.ID,
			"title":      p.Title,
			"created_at": p.CreatedAt,
			"medias":     medias,
		})
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": data})
}

// UpdateAlbumPost 编辑相册内容的说明（博主专属）
// PUT /api/album/posts/:id  { "title": "..." }
func UpdateAlbumPost(c *gin.Context) {
	if !isOwner(c) {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "msg": "相册是博主的专属地盘哦"})
		return
	}
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "ID不合法"})
		return
	}
	var req struct {
		Title string `json:"title" binding:"max=100"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "说明最长100字"})
		return
	}
	var post model.AlbumPost
	if err := database.DB.First(&post, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "内容不存在"})
		return
	}
	if err := database.DB.Model(&post).Update("title", req.Title).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "保存失败，请稍后再试"})
		return
	}
	post.Title = req.Title
	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "保存成功", "data": post})
}

// DeleteAlbumPost 删除相册内容（博主专属）：主体+媒体行+媒体文件一起删
// DELETE /api/album/posts/:id
func DeleteAlbumPost(c *gin.Context) {
	if !isOwner(c) {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "msg": "相册是博主的专属地盘哦"})
		return
	}
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "ID不合法"})
		return
	}
	var post model.AlbumPost
	if err := database.DB.First(&post, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "内容不存在"})
		return
	}
	// 找出这条内容下的所有媒体，把文件也删掉（视频大，不留垃圾）
	items := []model.AlbumItem{}
	database.DB.Where("post_id = ?", id).Find(&items)
	for _, it := range items {
		os.Remove("." + it.URL)
	}
	// 删媒体行 + 删主体
	database.DB.Where("post_id = ?", id).Delete(&model.AlbumItem{})
	if err := database.DB.Delete(&post).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "删除失败，请稍后再试"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "删除成功"})
}
