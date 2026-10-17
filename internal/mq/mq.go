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
	QueueNotify       = "notify"        // 正常队列：评论/留言 → 博主的未读提醒
	QueueNotifyFailed = "notify.failed" // 死信队列：处理失败的消息会被转到这里，方便排查
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
	// ① 先声明"死信队列"：处理失败的消息最终流到这里，方便事后排查
	if _, err = pubCh.QueueDeclare(QueueNotifyFailed, true, false, false, false, nil); err != nil {
		return err
	}
	// ② 再声明"正常队列"，并挂上死信规则。
	//    参数：(队列名, durable, autoDelete, exclusive, noWait, args)
	//      durable=true → 队列持久化：RabbitMQ 重启后队列还在
	//      x-dead-letter-exchange/routing-key → "这条消息处理失败了，请转去 notify.failed"
	//    ⚠️ 队列已存在时，参数必须和上次完全一致，否则 RabbitMQ 报 PRECONDITION_FAILED
	args := amqp.Table{
		"x-dead-letter-exchange":    "",                // 空串 = 默认交换机
		"x-dead-letter-routing-key": QueueNotifyFailed, // 转给死信队列
	}
	if _, err = pubCh.QueueDeclare(QueueNotify, true, false, false, false, args); err != nil {
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
				slog.Error("通知消息处理失败", "err", err)
				// requeue=false：不重新入队。
				// 因为重投的还是同一份数据、大概率还是失败，会变成死循环。
				// 生产上更稳的做法是配"死信队列"(DLX) 把失败消息收集起来慢慢查。
				_ = d.Nack(false, false) // (多重确认, 是否重新入队)
				continue
			}
			_ = d.Ack(false) // 处理成功，告诉 RabbitMQ "这条可以从队列里删了"
		}
	}()
	return nil
}

// Close 关闭连接（优雅关闭的最后一步调用）
func Close() {
	if conn != nil {
		_ = conn.Close()
	}
	ready = false
}
