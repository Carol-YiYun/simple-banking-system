// Package bank 定義核心領域模型與業務規則。
// 本檔定義 Account 與交易 Log 結構，不含任何 HTTP 或儲存細節。

package bank

import "time"

// Account represents a bank account.
type Account struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Balance int64  `json:"balance"`
	Logs    []Log  `json:"-"`
}

// Log represents a transaction record.
type Log struct {
	Time      time.Time `json:"time"`
	Amount    int64     `json:"amount"`
	Direction string    `json:"direction"`
	CounterID string    `json:"counter_account"`
	Note      string    `json:"note"`
}
