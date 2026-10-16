package handler

import (
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"blog/internal/config"

	"github.com/gin-gonic/gin"
)

// ============================================================
// 图片墙模块 —— 把本地图片文件夹里的图片列给主页用
// 文件夹路径从 .env 的 PICTURES_DIR 读（换文件夹不用改代码）
// 两步配合：本文件负责"给清单"，r.Static 负责"递文件"
// ============================================================

// Picture 一张图片的信息（返回给前端的形状）
type Picture struct {
	Name string `json:"name"` // 文件名，如 3.jpg（前端拿它当 key 和 alt）
	URL  string `json:"url"`  // 访问地址，如 /pictures/3.jpg
}

// 允许展示的图片后缀（和上传接口同一套白名单）
var pictureExts = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true,
}

// ListPictures 列出图片文件夹里的所有图片，按文件名"自然顺序"排好
// GET /api/pictures （公开接口，游客打开主页也要能看）
func ListPictures(c *gin.Context) {
	// os.ReadDir：读出目录里的所有"条目"（文件 + 子文件夹各算一条）
	// 它自带按文件名排序，但那是"字典序"，下面我们会重排成"自然序"
	entries, err := os.ReadDir(config.Cfg.PicturesDir)
	if err != nil {
		// 路径不存在 / 没权限都落到这里；只报"读不到"，不把具体路径回给前端
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "读取图片目录失败"})
		return
	}

	// 老规矩：一开始就建成空切片，空也返回 []，绝不能是 null
	pics := []Picture{}
	for _, e := range entries {
		// 只认文件，跳过子文件夹（图片墙不做"递归下钻"）
		if e.IsDir() {
			continue
		}
		// 后缀转小写再查白名单：.JPG、.Jpg 也要认
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if !pictureExts[ext] {
			continue
		}
		// 拿文件信息（要它的"最后修改时间"当版本号）
		info, err := e.Info()
		if err != nil {
			continue // 读不到信息的文件直接跳过，别让一张坏文件毁掉整个列表
		}
		pics = append(pics, Picture{
			Name: e.Name(),
			// 前端拿到的地址长这样：/pictures/3.jpg?v=1792145678
			// vite 开发服务器会把 /pictures 开头的请求转发给 Go 后端，
			// 由 router.go 里的 r.Static 把磁盘上的文件递出去
			//
			// ?v= 后面是文件的最后修改时间（秒级时间戳），作用是"破缓存"：
			// 浏览器对同名文件会一直用缓存里的老图 —— 你换了照片、名字没变，
			// 页面上显示的还是旧的那张。带上版本号后，文件一改 URL 就变，
			// 浏览器当成新地址重新下载。（没变的图照样走缓存，不浪费流量）
			URL: "/pictures/" + e.Name() + "?v=" + strconv.FormatInt(info.ModTime().Unix(), 10),
		})
	}

	// 按文件名"自然排序"：保证 2.jpg 排在 10.jpg 前面
	sort.Slice(pics, func(i, j int) bool {
		return naturalLess(pics[i].Name, pics[j].Name)
	})

	c.JSON(http.StatusOK, gin.H{"code": 200, "data": pics})
}

// naturalLess 自然序比较：a 应该排在 b 前面就返回 true
// 为什么要自己写？普通字符串比较是"字典序"：
//
//	"10.jpg" < "2.jpg"    ← 因为第一个字符 '1' 比 '2' 小
//
// 但人看照片是按数字大小排的：2.jpg 应该在 10.jpg 前面
// 所以把文件名切成"数字段"和"文字段"交替比较，数字段按数值比
func naturalLess(a, b string) bool {
	i, j := 0, 0 // 两个文件名各自的游标
	for i < len(a) && j < len(b) {
		// 两边当前位置都是数字 → 各取出一整段连续数字，按"数值"比大小
		if isDigit(a[i]) && isDigit(b[j]) {
			si, sj := i, j // 记住数字段起点
			for i < len(a) && isDigit(a[i]) {
				i++ // 游标一路扫到数字段结束
			}
			for j < len(b) && isDigit(b[j]) {
				j++
			}
			// 去掉前导零再比："007" 和 "7" 是同一个数
			na := strings.TrimLeft(a[si:i], "0")
			nb := strings.TrimLeft(b[sj:j], "0")
			// 位数多的数更大（"10" 比 "2" 长 → 更大），不用转成整数就能比
			if len(na) != len(nb) {
				return len(na) < len(nb)
			}
			// 位数一样就逐字符比（都是数字字符，字典序 = 数值序）
			if na != nb {
				return na < nb
			}
			continue // 这段数字完全相同，接着比后面的内容
		}
		// 不是"两边都数字"的情况 → 老老实实逐字符比
		if a[i] != b[j] {
			return a[i] < b[j]
		}
		i++
		j++
	}
	// 前面全部相同，谁先到头谁在前（短的排前面）
	return len(a)-i < len(b)-j
}

// isDigit 判断一个字节是不是数字字符 '0'~'9'
// 文件名都是 ASCII，按字节比较就够，不用处理 Unicode
func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}
