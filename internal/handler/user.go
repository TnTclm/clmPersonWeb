package handler

import (
	"blog/internal/config"
	"blog/internal/database"
	"blog/internal/middleware"
	"blog/internal/model"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// Me 查看当前登录用户信息（需要登录）
// GET /api/me
// 之前不查库（直接用JWT里的字段）；现在要返回头像——头像改过之后
// 令牌里存的还是老的，所以改成查库拿【最新】的。一次主键查询，很快
func Me(c *gin.Context) {
	var user model.User
	if err := database.DB.First(&user, c.GetUint("user_id")).Error; err != nil {
		// 令牌合法但人没了（比如被删号），按未登录处理
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "用户不存在，请重新登录"})
		return
	}
	TouchActivity(user.ID) // 上线记一笔（热力图，静默失败不影响本接口）
	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"user_id":  user.ID,
			"username": user.Username,
			"nickname": user.Nickname,
			"avatar":   user.Avatar,                         // 空串=还没传过头像，前端显示默认头像
			"is_owner": user.ID == uint(config.Cfg.OwnerID), // 是不是博主（相册上传/删除的入口只给TA显示）
		},
	})
}

// UploadAvatar 上传头像（需要登录）
// POST /api/avatar  请求体是 multipart/form-data，文件字段名叫 avatar
// 流程：浏览器选文件 → 文件随表单发到后端 → 后端存到 uploads/avatar/ 目录 →
//
//	数据库记下访问地址 → 以后 <img src> 直接用这个地址
func UploadAvatar(c *gin.Context) {
	// FormFile：从表单里取名叫 avatar 的那个文件
	file, err := c.FormFile("avatar")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "请选择图片文件"})
		return
	}
	// 大小限制 2MB（2<<20 = 2×1024×1024 字节，位运算写法，Go 圈常见）
	if file.Size > 2<<20 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "图片不能超过 2MB"})
		return
	}
	// 后缀白名单：只放行图片格式，防有人传个 .exe 上来
	ext := strings.ToLower(filepath.Ext(file.Filename))
	allowed := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true}
	if !allowed[ext] {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "只支持 jpg/png/gif/webp 格式"})
		return
	}
	// 起文件名：avatar_用户ID_毫秒时间戳.后缀
	//   带用户ID → 好认是谁的；带时间戳 → 换头像时文件名变了，浏览器不会拿旧缓存糊弄你
	name := fmt.Sprintf("avatar_%d_%d%s", c.GetUint("user_id"), time.Now().UnixMilli(), ext)
	// 落盘到 uploads/avatar/ 子目录（资源分类管理，和相册 uploads/album/ 一个套路）
	savePath := filepath.Join("uploads", "avatar", name)
	// 子目录不存在就自动创建（MkdirAll 递归建目录，已存在也不报错）
	if err := os.MkdirAll(filepath.Dir(savePath), 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "创建目录失败"})
		return
	}
	if err := c.SaveUploadedFile(file, savePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "保存文件失败"})
		return
	}
	// 数据库记下访问地址（Update 只改 avatar 一个字段）
	url := "/uploads/avatar/" + name
	if err := database.DB.Model(&model.User{}).Where("id = ?", c.GetUint("user_id")).Update("avatar", url).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "头像地址保存失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "头像上传成功", "data": gin.H{"avatar": url}})
}

// UpdateProfileRequest 修改资料参数
type UpdateProfileRequest struct {
	Nickname string `json:"nickname" binding:"required,min=1,max=20"` // 昵称必填，最长20字
	Bio      string `json:"bio" binding:"max=100"`                    // 个性签名可空，最长100字
}

// UpdateProfile 修改自己的资料（昵称/个性签名），需要登录
// PUT /api/me
// 改完把新 JWT 一起发回去——令牌里存着旧昵称，不换卡的话
// 之后发的说说/评论还会署旧名
func UpdateProfile(c *gin.Context) {
	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "昵称1-20字，签名最多100字"})
		return
	}
	nickname := strings.TrimSpace(req.Nickname)
	bio := strings.TrimSpace(req.Bio)
	if nickname == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "昵称不能是空白"})
		return
	}
	uid := c.GetUint("user_id")
	// 先查出完整用户（GenerateToken 需要）
	var user model.User
	if err := database.DB.First(&user, uid).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "用户不存在，请重新登录"})
		return
	}
	// Updates 传 map：多字段一起更，空串也照更
	if err := database.DB.Model(&user).Updates(map[string]interface{}{
		"nickname": nickname,
		"bio":      bio,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "保存失败，请稍后再试"})
		return
	}
	// 同步历史数据里的旧昵称（说说/评论/留言都冗余存了一份），改名后一起换新
	database.DB.Model(&model.Talk{}).Where("user_id = ?", uid).Update("nickname", nickname)
	database.DB.Model(&model.Comment{}).Where("user_id = ?", uid).Update("nickname", nickname)
	database.DB.Model(&model.Message{}).Where("user_id = ?", uid).Update("nickname", nickname)
	// 签发新令牌（新昵称写进令牌）
	token, err := middleware.GenerateToken(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "签发令牌失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "保存成功",
		"data": gin.H{
			"token":    token,
			"nickname": nickname,
			"bio":      bio,
		},
	})
}
