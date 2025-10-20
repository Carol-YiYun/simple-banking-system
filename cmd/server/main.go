// cmd/server/main.go

// 本服務提供帳戶建立、存提款、轉帳等 RESTful API。
// 此檔案負責初始化模組（bank, server, storage），
// 並啟動 HTTP 伺服器；同時支援啟動時載入與結束時保存 JSON 快照。

package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"banking/internal/bank"
	"banking/internal/server"
	"banking/internal/storage"
)

func main() {
	const dataFile = "data.json"

	// 初始化銀行核心模組
	b := bank.NewBank()

	// 嘗試從上次的 JSON 快照載入資料，若不存在則以空銀行啟動
	if snap, err := storage.LoadSnapshot(dataFile); err == nil {
		b.Restore(snap)
	}

	// persist 函式：將當前銀行狀態快照存入 data.json
	persist := func() error {
		return storage.SaveSnapshot(dataFile, b.Snapshot())
	}

	// 初始化伺服器並注入 persist 回呼，以便在每次成功變更後自動儲存
	s := server.NewServer(b, persist)

	// 啟動背景 goroutine 監聽 SIGINT/SIGTERM 訊號，安全結束前保存狀態
	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
		<-ch
		_ = persist()
		os.Exit(0)
	}()

	log.Println("Bank server running at :8080")
	// 啟動 HTTP 伺服器；使用自定義 router 提供所有 API
	log.Fatal(http.ListenAndServe(":8080", s.Router()))
}
