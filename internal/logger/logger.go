package logger

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// Package logger 只干一件事：把全局日志系统装好。

// Level 全局日志级别：比它低的日志直接丢掉、不输出。
// 四级顺序：Debug < Info < Warn < Error（越往后越严重）
// 阶段0先写死成 Debug（什么都打，方便你观察效果）；
// 阶段1会改成从 .env 里的 LOG_LEVEL 读，不改代码就能调。
var Level = slog.LevelDebug

// 初始化日志
func Init() {
	// ① 造 Handler：它决定「日志长什么样」+「往哪里去」。
	//    这是 slog 最核心的设计：Logger 只管收，Handler 只管处理。
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level:     Level,
		AddSource: true,

		// ReplaceAttr：日志里每个属性写出去之前的"最后一道加工工序"。
		// 参数 a 就是即将被写出的那个属性（比如 time、level、source、msg，
		// 或者你业务里 slog.Info("上传成功", "user_id", 4) 里那个 user_id）。
		// 你可以改它的值、丢掉它，或者原样返回。
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// groups 是"分组路径"：属性若被包在 slog.Group(...) 里，groups 就不为空。
			// 我们要处理的 time/level/source/msg 永远在最外层，所以组内的一律放过。
			// （slog.Group 是给日志分组用的进阶功能，现在不用管）
			if len(groups) > 0 {
				return a
			}

			switch a.Key {
			case slog.TimeKey:
				// 默认时间格式是 2026-10-15T15:28:42.632+08:00，太长。
				// 收紧成 "2026-10-15 15:28:42.632"。
				// 这个 2006-01-02 15:04:05 是 Go 特有的时间模板写法，
				// 不是占位符，就是"这一刻"的字面量，死记即可。
				return slog.String(a.Key, a.Value.Time().Format("2006-01-02 15:04:05.000"))

			case slog.SourceKey:
				// source 属性的值是一个 *slog.Source 结构体（含文件路径 + 行号），
				// 所以要先断言出来。ok 是断言是否成功的标志，false 说明不是我们要的类型。
				if src, ok := a.Value.Any().(*slog.Source); ok {
					file := filepath.ToSlash(src.File)           // Windows 的 \ 统一换成 /
					file = file[strings.LastIndex(file, "/")+1:] // 砍掉前面所有目录，只留文件名
					return slog.String(a.Key, fmt.Sprintf("%s:%d", file, src.Line))
				}
			}

			// ⚠️ 这行不能删：没匹配到的属性必须原样返回，否则会被丢掉。
			// （真正的"丢弃"是返回零值 slog.Attr{}，那是另一个用法）
			return a
		},
	})
	// ② 用 Handler 造出 Logger，再把它设为「全局默认」。
	//    设完之后，任何地方写 slog.Info("...") 都会走上面这套配置。
	slog.SetDefault(slog.New(handler))
}

// Fatal 记一条 Error 日志，然后立刻结束进程。
//
// 为什么 slog 没有 Fatal？这是作者故意的——
// 「记录日志」和「杀掉程序」是两件不同的事，标准库希望你写清楚自己在干哪件。
// 但「启动失败就别硬跑了」这个动作我们确实需要，所以自己包一层。
//
// 注意：它只该用在启动阶段（连不上数据库这种）。
// 请求处理过程中千万别调，会把整个服务干掉的。

func Fatal(msg string, args ...any) {
	slog.Error(msg, args...)
	os.Exit(1)
}
