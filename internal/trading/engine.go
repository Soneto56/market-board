package trading

import (
	"log"

	"github.com/Soneto56/market-board/pkg/model"
	"gorm.io/gorm"
)

// Engine 简单撮合引擎
type Engine struct {
	db  *gorm.DB
	hub PriceHub
}

// PriceHub 行情价格查询接口
type PriceHub interface {
	GetLatestPrice(symbol string) (price, bid, ask float64)
}

// NewEngine 创建撮合引擎
func NewEngine(db *gorm.DB, hub PriceHub) *Engine {
	return &Engine{db: db, hub: hub}
}

// MatchResult 撮合结果
type MatchResult struct {
	OrderID      uint    `json:"order_id"`
	FillPrice    float64 `json:"fill_price"`
	FillQty      float64 `json:"fill_qty"`
	Status       string  `json:"status"`
	RejectReason string  `json:"reject_reason"`
}

// Match 撮合一个订单
func (e *Engine) Match(order *model.Order) *MatchResult {
	price, bid, ask := e.hub.GetLatestPrice(order.Symbol)

	if price <= 0 {
		return &MatchResult{
			OrderID:      order.ID,
			Status:       "REJECTED",
			RejectReason: "price not available",
		}
	}

	switch order.Type {
	case "MARKET":
		return &MatchResult{
			OrderID:   order.ID,
			FillPrice: price,
			FillQty:   order.Quantity,
			Status:    "FILLED",
		}

	case "LIMIT":
		if order.Side == "BUY" && order.Price >= ask {
			return &MatchResult{
				OrderID:   order.ID,
				FillPrice: ask,
				FillQty:   order.Quantity,
				Status:    "FILLED",
			}
		}
		if order.Side == "SELL" && order.Price <= bid {
			return &MatchResult{
				OrderID:   order.ID,
				FillPrice: bid,
				FillQty:   order.Quantity,
				Status:    "FILLED",
			}
		}
		return nil

	default:
		return &MatchResult{
			OrderID:      order.ID,
			Status:       "REJECTED",
			RejectReason: "unknown order type",
		}
	}
}

// ApplyResult 应用撮合结果（更新订单状态、用户余额、持仓）
func (e *Engine) ApplyResult(order *model.Order, result *MatchResult) error {
	return e.db.Transaction(func(tx *gorm.DB) error {
		// 1. 更新订单状态
		if err := tx.Model(order).Update("status", result.Status).Error; err != nil {
			log.Printf("failed to update order status: %v", err)
			return err
		}

		if result.Status != "FILLED" {
			return nil
		}

		// 2. 获取用户信息
		var user model.User
		if err := tx.First(&user, order.UserID).Error; err != nil {
			log.Printf("failed to find user %d: %v", order.UserID, err)
			return err
		}

		// 3. 计算成交金额
		amount := result.FillPrice * result.FillQty

		if order.Side == "BUY" {
			// 检查余额
			if user.Balance < amount {
				log.Printf("user %d balance insufficient: need %.2f, have %.2f", user.ID, amount, user.Balance)
				// 更新订单状态为 REJECTED
				tx.Model(order).Update("status", "REJECTED")
				return nil
			}

			// 扣余额
			if err := tx.Model(&user).Update("balance", gorm.Expr("balance - ?", amount)).Error; err != nil {
				log.Printf("failed to deduct balance: %v", err)
				return err
			}

			// 更新持仓
			var pos model.Position
			err := tx.Where("user_id = ? AND symbol = ?", order.UserID, order.Symbol).First(&pos).Error
			if err == gorm.ErrRecordNotFound {
				// 新建持仓
				pos = model.Position{
					UserID:   order.UserID,
					Symbol:   order.Symbol,
					Quantity: result.FillQty,
					AvgPrice: result.FillPrice,
				}
				if err := tx.Create(&pos).Error; err != nil {
					log.Printf("failed to create position: %v", err)
					return err
				}
				log.Printf("created new position for user %d: %s x%.4f @%.2f",
					order.UserID, order.Symbol, result.FillQty, result.FillPrice)
			} else if err != nil {
				log.Printf("failed to query position: %v", err)
				return err
			} else {
				// 更新已有持仓（加权平均成本）
				totalCost := pos.AvgPrice*pos.Quantity + amount
				newQty := pos.Quantity + result.FillQty
				newAvgPrice := totalCost / newQty
				if err := tx.Model(&pos).Updates(map[string]interface{}{
					"quantity":  newQty,
					"avg_price": newAvgPrice,
				}).Error; err != nil {
					log.Printf("failed to update position: %v", err)
					return err
				}
				log.Printf("updated position for user %d: %s qty %.4f -> %.4f",
					order.UserID, order.Symbol, pos.Quantity, newQty)
			}

		} else {
			// 卖出：检查持仓
			var pos model.Position
			if err := tx.Where("user_id = ? AND symbol = ?", order.UserID, order.Symbol).First(&pos).Error; err != nil {
				log.Printf("user %d has no position for %s", order.UserID, order.Symbol)
				tx.Model(order).Update("status", "REJECTED")
				return nil
			}
			if pos.Quantity < result.FillQty {
				log.Printf("user %d insufficient position: have %.4f, sell %.4f",
					order.UserID, pos.Quantity, result.FillQty)
				tx.Model(order).Update("status", "REJECTED")
				return nil
			}

			// 减持仓
			newQty := pos.Quantity - result.FillQty
			if newQty <= 0 {
				// 全部卖出，删除持仓记录
				if err := tx.Delete(&pos).Error; err != nil {
					log.Printf("failed to delete position: %v", err)
					return err
				}
				log.Printf("deleted position for user %d: %s", order.UserID, order.Symbol)
			} else {
				if err := tx.Model(&pos).Update("quantity", newQty).Error; err != nil {
					log.Printf("failed to update position: %v", err)
					return err
				}
				log.Printf("reduced position for user %d: %s qty -> %.4f",
					order.UserID, order.Symbol, newQty)
			}

			// 加余额
			if err := tx.Model(&user).Update("balance", gorm.Expr("balance + ?", amount)).Error; err != nil {
				log.Printf("failed to add balance: %v", err)
				return err
			}
		}

		return nil
	})
}

// Notifier 成交通知接口
type Notifier interface {
	Notify(userID uint, data any)
}

// FillNotification 成交通知结构
type FillNotification struct {
	Type    string  `json:"type"`
	OrderID uint    `json:"order_id"`
	Symbol  string  `json:"symbol"`
	Side    string  `json:"side"`
	Price   float64 `json:"price"`
	Qty     float64 `json:"qty"`
	Status  string  `json:"status"`
}

// ProcessOrder 完整处理订单：撮合 + 应用结果 + 通知
func (e *Engine) ProcessOrder(order *model.Order, notifier Notifier) error {
	result := e.Match(order)

	if result == nil {
		log.Printf("order %d pending: limit condition not met", order.ID)
		return nil
	}

	if err := e.ApplyResult(order, result); err != nil {
		log.Printf("failed to apply result for order %d: %v", order.ID, err)
		return err
	}

	if notifier != nil {
		notifier.Notify(order.UserID, FillNotification{
			Type:    "fill",
			OrderID: order.ID,
			Symbol:  order.Symbol,
			Side:    order.Side,
			Price:   result.FillPrice,
			Qty:     result.FillQty,
			Status:  result.Status,
		})
	}

	log.Printf("order %d processed: %s at %.2f", order.ID, result.Status, result.FillPrice)
	return nil
}
