# ──────────────────────────────
#  第一階段：編譯 Go 程式
# ──────────────────────────────
FROM golang:1.25.3 AS builder
ENV GOTOOLCHAIN=auto

# 工作目錄
WORKDIR /app

# 複製 go.mod / go.sum 並先下載依賴
COPY go.mod go.sum ./
RUN go mod download

# 複製所有程式碼
COPY . .

# 編譯主程式（cmd/server/main.go）
RUN CGO_ENABLED=0 GOOS=linux go build -o banking ./cmd/server

# ──────────────────────────────
#  第二階段：建立最小執行環境
# ──────────────────────────────
FROM gcr.io/distroless/base-debian12

WORKDIR /app

# 從 builder 複製已編譯好的執行檔
COPY --from=builder /app/banking .

# 若使用 JSON 快照，可掛載 data.json
VOLUME ["/app"]

# 開放 port
EXPOSE 8080

# 啟動指令
ENTRYPOINT ["/app/banking"]
