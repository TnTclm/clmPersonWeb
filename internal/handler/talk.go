package handler

import (
	"blog/internal/database"
	"blog/internal/model"
	"context"
	"encoding/json"
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
// 说说模块 —— 全站唯一的内容形态（文章已并入说说）
// 能力：短文字 + 九宫格配图 + 点赞（Redis）+ 评论（MySQL）
// 防护：发言频率限制 + 博主有"执法权"（可删任何人的说说）
// ============================================================

// CreateTalkRequest 发表说说参数
type CreateTalkRequest struct {
	Content string   `json:"content" binding:"max=1000"` // 文字可空（允许纯图说说）
	Images  []string `json:"images"`                     // 图片地址列表，最多9张（下面自己校验）
}

// UploadImage 通用图片上传（说说配图用；头像有专用接口）
// POST /api/images  multipart 表单，文件字段名叫 file （需要登录）
func UploadImage(c *gin.Context) {
	// 从表单里取名叫 file 的那个文件
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "请选择图片文件"})
		return
	}
	// 配图放宽到 5MB（2<<20 是2MB，5<<20 就是5MB）
	if file.Size > 5<<20 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "图片不能超过 5MB"})
		return
	}
	// 后缀白名单：只放行图片，防有人传 .exe
	// filepath.Ext 获取后缀
	//  照片.PNG    ===>  .PNG
	// 转小写为了更好验证
	ext := strings.ToLower(filepath.Ext(file.Filename))
	allowed := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true}
	if !allowed[ext] {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "只支持 jpg/png/gif/webp 格式"})
		return
	}
	// 文件名：talk_用户ID_毫秒时间戳.后缀（和头像一个套路，防重名防缓存）
	name := fmt.Sprintf("talk_%d_%d%s", c.GetUint("user_id"), time.Now().UnixMilli(), ext)
	// 落盘到 uploads/talk/ 子目录（资源分类管理：说说配图和头像分开住）
	//  uploads/talk/talk_4_time.jpg
	// filepath.Dir 会扣出 uploads/talk
	// os.MkdirAll 递归创建
	savePath := filepath.Join("uploads", "talk", name)
	// 子目录不存在就自动创建（MkdirAll 递归建目录，已存在也不报错）
	if err := os.MkdirAll(filepath.Dir(savePath), 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "创建目录失败"})
		return
	}
	if err := c.SaveUploadedFile(file, savePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "保存文件失败"})
		return
	}
	// 只返回地址，不写数据库——图片跟说说的绑定关系由"发表说说"那条请求决定
	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "上传成功", "data": gin.H{"url": "/uploads/talk/" + name}})
}

// CreateTalk 发表说说（需要登录 + 仅博主；其他人只能看/点赞/评论）
// POST /api/talks
func CreateTalk(c *gin.Context) {
	// 发布权收紧：只有博主能发（前端藏了按钮，后端这道闸才是真防线）
	if !isOwner(c) {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "msg": "只有博主能发说说哦"})
		return
	}
	var req CreateTalkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数不合法"})
		return
	}
	// 防刷屏第一道闸：发太快直接拦（429 = Too Many Requests，标准状态码）
	if !AllowSpeak(c.GetUint("user_id")) {
		c.JSON(http.StatusTooManyRequests, gin.H{"code": 429, "msg": "发太快啦，休息一下再发"})
		return
	}
	// 自定义校验①：文字和图片至少有一样（QQ 也允许发纯图）
	if strings.TrimSpace(req.Content) == "" && len(req.Images) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "总得写点什么或传张图吧"})
		return
	}
	// 自定义校验②：最多9张（九宫格的"九"就是这么来的）
	if len(req.Images) > 9 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "配图最多9张"})
		return
	}
	// 自定义校验③：每个地址必须是本站 /uploads/ 开头的——
	// 防有人塞外链图片、或写 ../../ 去够别的目录（路径攻击）
	for _, img := range req.Images {
		if !strings.HasPrefix(img, "/uploads/") {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "图片地址不合法"})
			return
		}
	}
	// 图片列表 → JSON 字符串存库：["/uploads/a","/uploads/b"]
	// json.Marshal：把 Go 的切片转成 JSON 格式的字节串
	imagesJSON, _ := json.Marshal(req.Images)
	talk := model.Talk{
		UserID:   c.GetUint("user_id"),
		Nickname: c.GetString("nickname"), // 昵称从令牌拿，前端说了不算
		Content:  strings.TrimSpace(req.Content),
		Images:   string(imagesJSON),
	}
	if err := database.DB.Create(&talk).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "发表失败，请稍后再试"})
		return
	}
	TouchActivity(c.GetUint("user_id")) // 活跃度+1（热力图）
	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "发表成功", "data": talk})
}

// TalkWithUser 说说 + 作者头像 + 两个计数（连表临时结构，不是数据库表）
type TalkWithUser struct {
	model.Talk
	Avatar       string `json:"avatar"`        // 作者头像（users 表连出来的）
	Likes        int64  `json:"likes"`         // 点赞数（Redis Set 数出来的）
	CommentCount int64  `json:"comment_count"` // 评论数（MySQL 一次分组查完）
}

// ListTalks 说说列表（公开，游客可看）
// GET /api/talks
func ListTalks(c *gin.Context) {
	// 必须初始化成空切片！var 声明的 nil 切片转 JSON 会变成 null，
	// 前端拿到 null 调 .map 直接报错；[] 才是安全的空数组
	talks := []TalkWithUser{}
	page, size := pageParams(c) // 读 ?page=1&size=20（不传就用默认值）
	database.DB.Table("talks").
		Select("talks.*, users.avatar AS avatar").
		Joins("LEFT JOIN users ON users.id = talks.user_id").
		Order("talks.created_at DESC"). // 最新的在最上，刷动态就该这样
		Offset((page - 1) * size).      // 跳过前面几页（第1页跳0条，第2页跳20条…）
		Limit(size + 1).                // 故意多取 1 条：专门用来判断"后面还有没有"
		Scan(&talks)

	// 注意：每条 talk 里的 images 是 JSON 字符串，前端要 JSON.parse 一下才能用
	fillCounters(context.Background(), talks)

	// 多取的那条只是"探路"的：说明后面还有 → 把它切掉，不返回给前端
	hasMore := len(talks) > size
	if hasMore {
		talks = talks[:size]
	}
	pageResult(c, talks, hasMore)
}

// fillCounters 给一批说说补上点赞数（Redis）和评论数（MySQL 一条 GROUP BY）
// 列表页和详情页共用
func fillCounters(ctx context.Context, talks []TalkWithUser) {
	// 点赞数：每条说说查一次 Redis Set 的大小（本地 Redis，几十次也是瞬间的事）
	for i := range talks {
		key := "talk:likes:" + strconv.FormatUint(uint64(talks[i].ID), 10)
		n, _ := database.RDB.SCard(ctx, key).Result()
		talks[i].Likes = n
	}
	// 评论数：一条 GROUP BY 全查出来（别一条一条查，那是 N+1 问题）
	if len(talks) == 0 {
		return
	}
	ids := make([]uint, 0, len(talks))
	for _, t := range talks {
		ids = append(ids, t.ID)
	}
	type cntRow struct {
		TalkID uint
		C      int64
	}
	rows := []cntRow{}
	database.DB.Model(&model.Comment{}).
		Select("talk_id, COUNT(*) AS c").
		Where("talk_id IN ?", ids).
		Group("talk_id").
		Scan(&rows)
	m := map[uint]int64{}
	for _, r := range rows {
		m[r.TalkID] = r.C
	}
	for i := range talks {
		talks[i].CommentCount = m[talks[i].ID]
	}
}

// GetTalk 单条说说详情（公开）—— 详情页用
// GET /api/talks/:id
func GetTalk(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "说说ID不合法"})
		return
	}
	talks := []TalkWithUser{}
	database.DB.Table("talks").
		Select("talks.*, users.avatar AS avatar").
		Joins("LEFT JOIN users ON users.id = talks.user_id").
		Where("talks.id = ?", id).
		Scan(&talks)
	// 一条都没查到 = 说说不存在（或已被删）
	if len(talks) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "说说不存在"})
		return
	}
	fillCounters(context.Background(), talks)
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": talks[0]})
}

// DeleteTalk 删除说说（需要登录：自己的随便删，博主谁的都能删——"执法权"）
// DELETE /api/talks/:id
func DeleteTalk(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "说说ID不合法"})
		return
	}
	var talk model.Talk
	if err := database.DB.First(&talk, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "说说不存在"})
		return
	}
	// 归属校验：不是自己的 → 只有博主能动手（防恶意内容的最终手段）
	if talk.UserID != c.GetUint("user_id") && !isOwner(c) {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "msg": "只能删除自己的说说"})
		return
	}
	if err := database.DB.Delete(&talk).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "删除失败，请稍后再试"})
		return
	}
	// ---- 连带清理（防恶意用户用垃圾内容塞满磁盘）----
	// ① 这条说说的评论记录
	database.DB.Where("talk_id = ?", talk.ID).Delete(&model.Comment{})
	// ② 点赞的两个账本一起清（少清一个就会留下"幽灵赞"）
	ctx := context.Background()
	talkIDStr := strconv.Itoa(id) // 账本里存的成员是字符串，提前转好，下面要用三次
	likesKey := "talk:likes:" + talkIDStr
	// 趁"谁赞过这条"的总账本还在，先把名单读出来——
	// 靠这份名单就知道该去哪些人的私人账本上划账，不用去扫全站用户
	likers, _ := database.RDB.SMembers(ctx, likesKey).Result()
	// 拿着名单挨个划：把这条说说从每个点赞人的"我赞过的"里删掉
	for _, uid := range likers {
		database.RDB.SRem(ctx, "talk:liked:"+uid, talkIDStr)
	}
	// 最后把总账本整个删掉（名单已经用完，留着只会占内存）
	database.RDB.Del(ctx, likesKey)
	// ③ 配图文件（尽力而为：文件删失败也不影响记录删除）
	deleteTalkImages(talk.Images)
	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "删除成功"})
}

// deleteTalkImages 把图片地址 JSON 解析出来，挨个删本地文件
// 只认 /uploads/ 开头的地址，且带 ".." 的一律跳过（防路径穿越删到别的东西）
func deleteTalkImages(imagesJSON string) {
	var urls []string
	if err := json.Unmarshal([]byte(imagesJSON), &urls); err != nil {
		return
	}
	for _, u := range urls {
		if !strings.HasPrefix(u, "/uploads/") || strings.Contains(u, "..") {
			continue
		}
		rel := strings.TrimPrefix(u, "/uploads/")
		_ = os.Remove(filepath.Join("uploads", filepath.FromSlash(rel)))
	}
}

// ============================================================
// 点赞 —— 数据全存 Redis，双 Set 结构：
//   talk:likes:{说说ID}   成员=点过赞的用户ID（SCard = 总赞数）
//   talk:liked:{用户ID}   成员=TA赞过的说说ID（打开页面恢复小红心用）
// 点赞/取消要同时维护两个 Set，跟"互相关注"一个道理
// ============================================================

// ToggleTalkLike 点赞/取消（切换式）
// POST /api/talks/:id/like （需要登录）
func ToggleTalkLike(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "说说ID不合法"})
		return
	}
	// 写操作先验说说存在
	var talk model.Talk
	if err := database.DB.First(&talk, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "说说不存在"})
		return
	}

	ctx := context.Background()
	uid := strconv.FormatUint(uint64(c.GetUint("user_id")), 10)
	likesKey := "talk:likes:" + strconv.Itoa(id) // 总账本：这条说说谁赞了
	mineKey := "talk:liked:" + uid               // 我的账本：我赞过哪些

	// SISMEMBER：我在总账本上吗？
	isMember, _ := database.RDB.SIsMember(ctx, likesKey, uid).Result()
	if isMember {
		// 在 → 取消点赞，两个账本一起划掉
		database.RDB.SRem(ctx, likesKey, uid)
		database.RDB.SRem(ctx, mineKey, strconv.Itoa(id))
	} else {
		// 不在 → 点赞，两个账本一起记上（Set 天然去重，防刷）
		database.RDB.SAdd(ctx, likesKey, uid)
		database.RDB.SAdd(ctx, mineKey, strconv.Itoa(id))
	}
	// SCard：总账本上还有几个人 = 最新点赞数
	count, _ := database.RDB.SCard(ctx, likesKey).Result()

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "ok",
		"data": gin.H{
			"liked": !isMember, // 操作完的状态：true=现在是已赞
			"likes": count,     // 最新点赞数，前端直接用
		},
	})
}

// MyLikedTalks 我赞过的说说ID列表（前端打开页面时恢复小红心）
// GET /api/me/liked-talks （需要登录）
func MyLikedTalks(c *gin.Context) {
	ctx := context.Background()
	mineKey := "talk:liked:" + strconv.FormatUint(uint64(c.GetUint("user_id")), 10)
	members, _ := database.RDB.SMembers(ctx, mineKey).Result()
	// Set 里存的是字符串"3"，转回数字给前端好比对
	ids := []int{} // 老规矩：空也返回 []，别返回 null
	for _, s := range members {
		if n, err := strconv.Atoi(s); err == nil {
			ids = append(ids, n)
		}
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": ids})
}
