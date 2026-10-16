package router

import (
	"blog/internal/handler"
	"blog/internal/middleware"

	"github.com/gin-gonic/gin"
)

// Setup 全站路由登记处。
// 原来这些路由全挤在 cmd/server/main.go 里，现在拆出来单独住一个包：
//
//	main.go  负责「启动流程」——装日志、连数据库、起服务
//	router   负责「地址登记」——哪个 URL 交给哪个处理函数
//
// 好处：加接口只动这个文件，main.go 永远那么短
func Setup(r *gin.Engine) {
	// 静态文件服务：浏览器访问 http://localhost:8080/uploads/xxx.png
	// Gin 就把 ./uploads/ 目录里对应的文件直接递出去
	r.Static("/uploads", "./uploads")

	// 根路径：一个最简单的"我还活着"信号，浏览器直接打开 8080 就能看到
	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"code": 200,
			"msg":  "lets go!",
		})
	})

	// ---------- 公开接口组 ----------
	// Group("/api") 只是给组内路由统一加前缀，本身不挂任何中间件 → 游客也能访问
	pub := r.Group("/api")
	{
		pub.GET("/captcha", handler.GenerateCaptcha)             // 验证码
		pub.POST("/register", handler.Register)                  // 注册
		pub.POST("/login", handler.Login)                        // 登录
		pub.GET("/users/:id", handler.GetUserProfile)            // 个人空间（公开）
		pub.GET("/users/:id/activities", handler.UserActivities) // 活跃度热力图（公开）
		pub.GET("/talks", handler.ListTalks)                     // 说说列表（公开，说说=全站内容形态）
		pub.GET("/talks/:id", handler.GetTalk)                   // 单条说说详情（公开）
		pub.GET("/talks/:id/comments", handler.ListTalkComments) // 说说评论列表（公开）
		pub.GET("/album", handler.ListAlbum)                     // 相册列表（公开）
		pub.GET("/guestbook", handler.ListMessages)              // 留言板（公开）
	}

	// ---------- 需要登录的接口组 ----------
	authed := r.Group("/api")
	authed.Use(middleware.AuthMiddleware()) // 组级中间件：组内所有接口都要验卡
	{
		authed.GET("/me", handler.Me)
		authed.PUT("/me", handler.UpdateProfile)                             // 修改自己的资料（昵称/个性签名）
		authed.GET("/me/liked-talks", handler.MyLikedTalks)                  // 我赞过的说说ID（恢复小红心）
		authed.POST("/talks/:id/like", handler.ToggleTalkLike)               // 说说点赞/取消（切换式）
		authed.POST("/talks/:id/comments", handler.CreateTalkComment)        // 发表说说评论
		authed.DELETE("/talks/:id/comments/:cid", handler.DeleteTalkComment) // 删评论（自己的/博主的）
		authed.POST("/avatar", handler.UploadAvatar)                         // 上传头像
		authed.POST("/images", handler.UploadImage)                          // 通用图片上传（说说配图）
		authed.POST("/talks", handler.CreateTalk)                            // 发表说说
		authed.DELETE("/talks/:id", handler.DeleteTalk)                      // 删除说说（博主可删任何人的）
		authed.POST("/album", handler.CreateAlbumPost)                       // 博主发布相册内容
		authed.PUT("/album/posts/:id", handler.UpdateAlbumPost)              // 博主编辑相册说明
		authed.DELETE("/album/posts/:id", handler.DeleteAlbumPost)           // 博主删除相册内容
		authed.POST("/guestbook", handler.CreateMessage)                     // 发表留言
		authed.DELETE("/guestbook/:id", handler.DeleteMessage)               // 删除留言
	}
}
