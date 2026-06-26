package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type OrderType string

func (o OrderType) String() string {
	return string(o)
}

const (
	OrderTypeUnspecified OrderType = "Unspecified"
	OrderTypeLimit       OrderType = "Limit"
	OrderTypeMarket      OrderType = "Market"
	OrderTypeStopLimit   OrderType = "StopLimit"
	OrderTypeStopMarket  OrderType = "StopMarket"
	OrderTypeSwap        OrderType = "Swap"
)

type OrderStatus string

func (os OrderStatus) String() string {
	return string(os)
}

const (
	OrderStatusUnspecified        OrderStatus = "Unspecified"
	OrderStatusNew                OrderStatus = "New"
	OrderStatusTriggered          OrderStatus = "Triggered"
	OrderStatusOpen               OrderStatus = "Open"
	OrderStatusPartiallyCompleted OrderStatus = "PartiallyCompleted"
	OrderStatusCompleted          OrderStatus = "Completed"
	OrderStatusCanceled           OrderStatus = "Canceled"
	OrderStatusRejected           OrderStatus = "Rejected"
)

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

	CreatedAt time.Time `json:"createdAt"`
}

func (o *Order) TotalExecuted() decimal.Decimal {
	return o.ExecutedAmount.Mul(o.AvgPrice)
}

func (o *Order) IsNewStopOrder() bool {
	return (o.Type == OrderTypeStopMarket || o.Type == OrderTypeStopLimit) && o.Status == OrderStatusNew
}

type MatchedOrder struct {
	Order         *Order          `json:"order"`
	IsDone        bool            `json:"isDone"`
	MatchedAmount decimal.Decimal `json:"matchedAmount"`
}

func (o *Order) Cancel() {
	o.CanceledAmount = o.CanceledAmount.Add(o.AvailableAmount)
	o.CanceledTotal = o.CanceledTotal.Add(o.AvailableTotal)
	o.AvailableAmount = decimal.Zero
	o.AvailableTotal = decimal.Zero
	o.Status = OrderStatusCanceled
}

func (o *Order) Reject(rejectReason RejectReason) {
	o.Status = OrderStatusRejected
	o.RejectReason = rejectReason
}
