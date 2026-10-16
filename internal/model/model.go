package model

import "time"

// User 用户表
type User struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`  // 主键 gorm 默认自增
	Username  string    `gorm:"uniqueIndex;size:50" json:"username"` // 登录名， 唯一索引(不允许重名)
	Password  string    `gorm:"size:100" json:"-"`                   // 密码 json "-" 表示不返回给前端
	Nickname  string    `gorm:"size:50" json:"nickname"`             // 昵称 页面上展示用
	Avatar    string    `gorm:"size:255" json:"avatar"`              // 头像图片的访问地址（如 /uploads/avatar_3_xxx.png），没传过是空串
	Bio       string    `gorm:"size:100" json:"bio"`                 // 个性签名（个人空间展示），最长100字，可空
	CreatedAt time.Time `json:"created_at"`                          // 创建时间  GROM自动填充
	UpdatedAt time.Time `json:"updated_at"`                          // 更新时间
}

// Comment 评论表（挂在说说下：文章已并入说说，评论跟着搬家）
type Comment struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	TalkID    uint      `gorm:"index" json:"talk_id"`    // 属于哪条说说
	UserID    uint      `gorm:"index" json:"user_id"`    // 评论者用户ID——连表查头像全靠它
	Nickname  string    `gorm:"size:50" json:"nickname"` // 评论者昵称
	Content   string    `gorm:"size:500" json:"content"` // 评论内容
	CreatedAt time.Time `json:"created_at"`
}

// Talk 说说表（全站唯一的内容形态：短文字 + 最多9张配图 + 点赞 + 评论）
type Talk struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    uint      `gorm:"index" json:"user_id"`     // 发说说的人
	Nickname  string    `gorm:"size:50" json:"nickname"`  // 昵称（冗余一份，列表页不用连表查名字）
	Content   string    `gorm:"size:1000" json:"content"` // 文字内容，可为空（允许纯图说说）
	Images    string    `gorm:"size:2000" json:"images"`  // 图片地址列表，存JSON数组字符串，如 ["/uploads/talk/1_x.jpg"]
	CreatedAt time.Time `json:"created_at"`
}

// AlbumPost 相册内容（博主专属：一段说明 + 多个照片/视频，一次发布算一条）
type AlbumPost struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    uint      `gorm:"index" json:"user_id"`  // 发布者（永远只有博主）
	Title     string    `gorm:"size:100" json:"title"` // 说明文字（可空，最长100字）
	CreatedAt time.Time `json:"created_at"`
}

// AlbumItem 相册媒体文件（一个文件一条记录，挂在某条相册内容下）
type AlbumItem struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	PostID    uint      `gorm:"index" json:"post_id"` // 属于哪条相册内容
	MediaType string    `gorm:"size:10" json:"type"`  // "image" 或 "video"
	URL       string    `gorm:"size:500" json:"url"`  // 访问地址，如 /uploads/album/xxx.mp4
	CreatedAt time.Time `json:"created_at"`
}

// Message 留言板（弹幕式：每条留言是一道飞过樱花雨的弹幕）
type Message struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    uint      `gorm:"index" json:"user_id"`
	Nickname  string    `gorm:"size:50" json:"nickname"`
	Content   string    `gorm:"size:50" json:"content"` // 弹幕要短，50字上限
	CreatedAt time.Time `json:"created_at"`
}

// Activity 活跃度记录（LeetCode 式热力图的数据源：一人一天最多一行，越勤快 count 越大）
type Activity struct {
	ID     uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID uint   `gorm:"uniqueIndex:idx_user_date" json:"user_id"`      // 谁的活跃（和 date 组成联合唯一索引）
	Date   string `gorm:"uniqueIndex:idx_user_date;size:10" json:"date"` // 日期字符串，如 "2026-10-14"（直接存字符串，省去时区换算）
	Count  int    `json:"count"`                                         // 当天活跃次数：上线记1次，每创作一次再+1
}
