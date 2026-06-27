package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// OrderType identifies the execution semantics of an order.
type OrderType string

// String returns the raw order type value.
func (o OrderType) String() string {
	return string(o)
}

const (
	// OrderTypeUnspecified is the zero-value order type.
	OrderTypeUnspecified OrderType = "Unspecified"

	// OrderTypeLimit is a limit order.
	OrderTypeLimit OrderType = "Limit"

	// OrderTypeMarket is a market order.
	OrderTypeMarket OrderType = "Market"

	// OrderTypeStopLimit is a stop-limit order.
	OrderTypeStopLimit OrderType = "StopLimit"

	// OrderTypeStopMarket is a stop-market order.
	OrderTypeStopMarket OrderType = "StopMarket"

	// OrderTypeSwap is an aggregate order that is expanded into market suborders.
	OrderTypeSwap OrderType = "Swap"
)

// OrderStatus identifies the current lifecycle state of an order.
type OrderStatus string

// String returns the raw order status value.
func (os OrderStatus) String() string {
	return string(os)
}

const (
	// OrderStatusUnspecified is the zero-value order status.
	OrderStatusUnspecified OrderStatus = "Unspecified"
	// OrderStatusNew means the order has not started execution.
	OrderStatusNew OrderStatus = "New"
	// OrderStatusTriggered means a stop condition has triggered.
	OrderStatusTriggered OrderStatus = "Triggered"
	// OrderStatusOpen means the order is active on an execution venue.
	OrderStatusOpen OrderStatus = "Open"
	// OrderStatusPartiallyCompleted means the order executed partially.
	OrderStatusPartiallyCompleted OrderStatus = "PartiallyCompleted"
	// OrderStatusCompleted means the order executed fully.
	OrderStatusCompleted OrderStatus = "Completed"
	// OrderStatusCanceled means the order was canceled.
	OrderStatusCanceled OrderStatus = "Canceled"
	// OrderStatusRejected means the order was rejected.
	OrderStatusRejected OrderStatus = "Rejected"
)

// Order represents either a user swap order or a generated market suborder.
type Order struct {
	ID      uint64 `json:"ID"`
	Account string `json:"account"`

	Symbol       Symbol       `json:"symbol"`
	Type         OrderType    `json:"type"`
	Side         Side         `json:"side"`
	Status       OrderStatus  `json:"status"`
	RejectReason RejectReason `json:"rejectReason"`

	Amount          decimal.Decimal `json:"amount"`
	AvailableAmount decimal.Decimal `json:"availableAmount"`
	ExecutedAmount  decimal.Decimal `json:"executedAmount"`
	CanceledAmount  decimal.Decimal `json:"canceledAmount"`

	Total          decimal.Decimal `json:"total"`
	AvailableTotal decimal.Decimal `json:"availableTotal"`
	ExecutedTotal  decimal.Decimal `json:"executedTotal"`
	CanceledTotal  decimal.Decimal `json:"canceledTotal"`

	Price    decimal.Decimal `json:"price"`
	AvgPrice decimal.Decimal `json:"avgPrice"`

	StrandedAmount decimal.Decimal `json:"strandedAmount"`
	StrandedAsset  string          `json:"strandedAsset"`

	CreatedAt time.Time `json:"createdAt"`
}

// Reject marks the order as rejected with the provided reason.
func (o *Order) Reject(rejectReason RejectReason) {
	o.Status = OrderStatusRejected
	o.RejectReason = rejectReason
}
