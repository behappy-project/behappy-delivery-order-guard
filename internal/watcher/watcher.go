package watcher

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"behappy-delivery-order-guard/internal/config"
	"behappy-delivery-order-guard/internal/ingestion"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

// TimeoutWatcher 超时监控器
type TimeoutWatcher struct {
	cfg    *config.Config
	logger *zap.Logger
	redis  *redis.Client
}

// NewTimeoutWatcher 创建超时监控器
func NewTimeoutWatcher(cfg *config.Config, logger *zap.Logger, redis *redis.Client) *TimeoutWatcher {
	return &TimeoutWatcher{
		cfg:    cfg,
		logger: logger,
		redis:  redis,
	}
}

// Start 启动超时监控器
func (w *TimeoutWatcher) Start(ctx context.Context) {
	w.logger.Info("Timeout watcher started",
		zap.Duration("timeout", w.cfg.OrderTimeout),
	)

	ticker := time.NewTicker(30 * time.Second) // 每30秒检查一次
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.checkTimeouts(ctx)
		}
	}
}

// checkTimeouts 检查所有订单的超时状态
func (w *TimeoutWatcher) checkTimeouts(ctx context.Context) {
	// 获取所有订单键
	keys, err := w.redis.Keys(ctx, "order:*").Result()
	if err != nil {
		w.logger.Error("Failed to get order keys", zap.Error(err))
		return
	}

	now := time.Now()
	for _, key := range keys {
		// 只处理订单数据键（不是状态键或时间键）
		if !w.isOrderDataKey(key) {
			continue
		}

		orderID := w.extractOrderID(key)
		if orderID == "" {
			continue
		}

		// 获取订单数据
		orderJSON, err := w.redis.Get(ctx, key).Result()
		if err != nil {
			if err == redis.Nil {
				continue
			}
			w.logger.Error("Failed to get order", zap.String("order_id", orderID), zap.Error(err))
			continue
		}

		var order ingestion.Order
		if err := json.Unmarshal([]byte(orderJSON), &order); err != nil {
			w.logger.Error("Failed to unmarshal order", zap.String("order_id", orderID), zap.Error(err))
			continue
		}

		// 检查订单状态
		statusKey := fmt.Sprintf("order:status:%s", orderID)
		status, err := w.redis.Get(ctx, statusKey).Result()
		if err != nil {
			if err == redis.Nil {
				continue
			}
			w.logger.Error("Failed to get order status", zap.String("order_id", orderID), zap.Error(err))
			continue
		}

		// 如果订单已经被接单或出餐，跳过
		if status == "accepted" || status == "ready" {
			continue
		}

		// 检查是否超时
		elapsed := now.Sub(order.CreatedAt)
		if elapsed > w.cfg.OrderTimeout {
			w.handleTimeout(ctx, &order, elapsed)
		}
	}
}

// handleTimeout 处理超时订单
func (w *TimeoutWatcher) handleTimeout(ctx context.Context, order *ingestion.Order, elapsed time.Duration) {
	w.logger.Warn("Order timeout detected",
		zap.String("order_id", order.ID),
		zap.String("order_no", order.OrderNo),
		zap.String("platform", order.Platform),
		zap.String("status", order.Status),
		zap.Duration("elapsed", elapsed),
		zap.Duration("timeout", w.cfg.OrderTimeout),
	)

	// 发送报警（这里可以扩展为发送到钉钉、企业微信、邮件等）
	w.sendAlert(order, elapsed)

	// 更新订单状态为超时
	statusKey := fmt.Sprintf("order:status:%s", order.ID)
	if err := w.redis.Set(ctx, statusKey, "timeout", w.cfg.OrderTimeout).Err(); err != nil {
		w.logger.Error("Failed to update order status to timeout", zap.Error(err))
	}
}

// sendAlert 发送报警（模拟）
func (w *TimeoutWatcher) sendAlert(order *ingestion.Order, elapsed time.Duration) {
	alertMsg := fmt.Sprintf(
		"⚠️ 订单超时报警\n"+
			"订单ID: %s\n"+
			"订单号: %s\n"+
			"平台: %s\n"+
			"客户: %s (%s)\n"+
			"金额: ¥%.2f\n"+
			"已等待: %s\n"+
			"超时阈值: %s\n"+
			"请及时处理！",
		order.ID,
		order.OrderNo,
		order.Platform,
		order.Customer.Name,
		order.Customer.Phone,
		order.TotalAmount,
		elapsed.Round(time.Second),
		w.cfg.OrderTimeout,
	)

	w.logger.Error("ALERT: Order timeout",
		zap.String("order_id", order.ID),
		zap.String("message", alertMsg),
	)

	// TODO: 这里可以集成实际的报警系统
	// - 发送到钉钉/企业微信
	// - 发送邮件
	// - 发送短信
	// - 推送到 Prometheus
}

// isOrderDataKey 判断是否是订单数据键（不是状态键或时间键）
func (w *TimeoutWatcher) isOrderDataKey(key string) bool {
	// 订单数据键格式: order:ORDER_ID
	// 状态键格式: order:status:ORDER_ID
	// 时间键格式: order:created:ORDER_ID
	if len(key) <= 6 || key[:6] != "order:" {
		return false
	}
	// 如果包含 "status" 或 "created"，则不是订单数据键
	return key[6:12] != "status" && key[6:13] != "created"
}

// extractOrderID 从键中提取订单ID
func (w *TimeoutWatcher) extractOrderID(key string) string {
	// key 格式: order:ORDER_ID
	if len(key) <= 6 {
		return ""
	}
	return key[6:]
}
