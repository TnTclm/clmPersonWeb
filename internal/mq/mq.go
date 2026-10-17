package mq

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
)

// ============================================================
// RabbitMQ 封装 —— 连接、发消息、收消息
// 单独一个包的理由：业务代码只该说"我要发一条通知"，
// 不该关心 TCP 连接、通道、队列声明这些细节
// ============================================================

const (
	QueueNotify       = "notify"        // 主队列：评论/留言 → 博主的未读提醒
	QueueNotifyRetry  = "notify.retry"  // 重试队列：消息在这里"关禁闭"几秒，然后自动回到主队列
	QueueNotifyFailed = "notify.failed" // 最终失败队列：重试次数用尽后进这里，只记录
	maxRetry          = 3               // 最多重试 3 次
	retryDelayMs      = 10000           // 每次重试间隔 10 秒
)

var (
	conn  *amqp.Connection // 一条 TCP 连接，整个进程共用
	pubCh *amqp.Channel    // 专门用来"发消息"的通道
	pubMu sync.Mutex       // 互斥锁：amqp 的 Channel 不是并发安全的，多个请求同时发要排队
	ready bool             // 是否已经连上（没连上时发布要直接返回错误，不能空指针崩溃）
)

// Connect 连上 RabbitMQ，并声明我们需要的队列
func Connect(url string) error {
	var err error
	// Dial = 建立到 RabbitMQ 的 TCP 连接
	conn, err = amqp.Dial(url)
	if err != nil {
		return err
	}
	// 在连接上开一条"通道"（Channel）——真正的收发消息都走通道
	pubCh, err = conn.Channel()
	if err != nil {
		return err
	}
	// ① 最终失败队列：重试用尽的消息进这里（只记录，不再动它）
	if _, err = pubCh.QueueDeclare(QueueNotifyFailed, true, false, false, false, nil); err != nil {
		return err
	}
	// ② 重试队列：消息在这里"关禁闭" 10 秒。
	//    x-message-ttl = 消息存活 10 秒；到点过期后，按死信规则自动回到主队列 ——
	//    这就是 RabbitMQ 里实现"延迟重试"的经典手法（不用额外的定时器）
	if _, err = pubCh.QueueDeclare(QueueNotifyRetry, true, false, false, false, amqp.Table{
		"x-message-ttl":             int32(retryDelayMs),
		"x-dead-letter-exchange":    "",
		"x-dead-letter-routing-key": QueueNotify,
	}); err != nil {
		return err
	}
	// ③ 主队列：处理失败时不是直接丢弃，而是转去"重试队列"
	//    ⚠️ 队列已存在时参数必须和上次完全一致，否则 RabbitMQ 报 PRECONDITION_FAILED
	if _, err = pubCh.QueueDeclare(QueueNotify, true, false, false, false, amqp.Table{
		"x-dead-letter-exchange":    "",
		"x-dead-letter-routing-key": QueueNotifyRetry,
	}); err != nil {
		return err
	}
	ready = true
	return nil
}

// Publish 把一条消息丢进队列（自动转成 JSON）
func Publish(queue string, payload any) error {
	if !ready {
		return errors.New("RabbitMQ 未连接")
	}
	// 先序列化：队列里传的永远是"字节"，不是 Go 结构体
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	// 加锁：同一个 Channel 不能被多个 goroutine 同时用（amqp 库的硬性要求）
	pubMu.Lock()
	defer pubMu.Unlock()
	// PublishWithContext 参数：(上下文, 交换机, 路由键, mandatory, immediate, 消息体)
	// 这里交换机传空串 = 直接用"默认交换机"，路由键就是队列名，消息直接进队列
	// DeliveryMode: amqp.Persistent = 消息持久化（配合 durable 队列，重启也不丢）
	return pubCh.PublishWithContext(context.Background(), "", queue, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         body,
	})
}

// Consume 开一个消费者：一直从队列取消息，交给 handle 处理
//
//	handle 返回 nil  → 处理成功，ack 确认（RabbitMQ 把这条消息删掉）
//	handle 返回错误  → 处理失败，nack 丢弃并记日志
//
// 注意：它会在后台起一个 goroutine，函数本身立刻返回，不阻塞启动流程
func Consume(queue string, handle func([]byte) error) error {
	// 消费者用自己独立的 Channel（和发布者分开，一方出问题不影响另一方）
	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	// 每次只预取 1 条：上一条没处理完，不给我下一条。
	// 不做这个限制，消息会一股脑全推到消费者手里，处理失败时很难重投
	if err := ch.Qos(1, 0, false); err != nil {
		return err
	}
	// Consume 参数：(队列, 消费者名(空=自动生成), autoAck, exclusive, noLocal, noWait, args)
	//   autoAck=false → 手动确认：处理完才 ack。
	//   如果处理到一半进程崩了，没 ack 的消息会被重新投递（可靠性的关键）
	msgs, err := ch.Consume(queue, "", false, false, false, false, nil)
	if err != nil {
		return err
	}

	// 后台一直收：连接断了 msgs 通道会关闭，循环自然结束
	go func() {
		for d := range msgs {
			if err := handle(d.Body); err != nil {
				// 这条消息已经"死信"过几次了？（RabbitMQ 记在 x-death 头里）
				tries := retryCount(d.Headers)
				if tries >= maxRetry {
					slog.Error("重试次数用尽，转入失败队列", "err", err, "已重试次数", tries)
					// 转存到"最终失败队列"，再把原消息确认掉（不然它会一直循环）
					if perr := PublishRaw(QueueNotifyFailed, d.Body); perr != nil {
						slog.Error("转存失败队列也失败了", "err", perr)
					}
					_ = d.Ack(false)
					continue
				}
				slog.Warn("消息处理失败，稍后自动重试", "err", err, "已重试次数", tries)
				// 不重新入队 → 走主队列的死信规则进入"重试队列"，
				// 10 秒后 TTL 到期，自动回到主队列重新消费
				_ = d.Nack(false, false) // (多重确认, 是否重新入队)
				continue
			}
			_ = d.Ack(false) // 处理成功，告诉 RabbitMQ "这条可以从队列里删了"
		}
	}()
	return nil
}

// retryCount 从消息头里读"它已经进过几次重试队列"
// RabbitMQ 每次把消息死信出去，都会往 x-death 里记一笔（同一个队列会累计到 count）
func retryCount(headers amqp.Table) int {
	if headers == nil {
		return 0
	}
	raw, ok := headers["x-death"]
	if !ok {
		return 0 // 第一次失败，还没有 x-death
	}
	// x-death 的结构：[]interface{}{ amqp.Table{"queue":..., "count":...}, ... }
	list, ok := raw.([]interface{})
	if !ok {
		return 0
	}
	for _, it := range list {
		m, ok := it.(amqp.Table)
		if !ok {
			continue
		}
		// 只数"进过重试队列"的次数
		if q, _ := m["queue"].(string); q == QueueNotifyRetry {
			if c, ok := m["count"].(int64); ok {
				return int(c)
			}
		}
	}
	return 0
}

// PublishRaw 把一段"已经是字节"的消息原样丢进队列
// （转存失败消息时用：消息本来就是 JSON 字节，不用再序列化一遍）
func PublishRaw(queue string, body []byte) error {
	if !ready {
		return errors.New("RabbitMQ 未连接")
	}
	pubMu.Lock()
	defer pubMu.Unlock()
	return pubCh.PublishWithContext(context.Background(), "", queue, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         body,
	})
}

// Close 关闭连接（优雅关闭的最后一步调用）
func Close() {
	if conn != nil {
		_ = conn.Close()
	}
	ready = false
}
