# 1. ビルド用イメージ
FROM golang:1.24-alpine AS builder

WORKDIR /app

# 依存関係のコピーとインストール
COPY go.mod go.sum ./
RUN go mod download

# ソースコードのコピー
COPY . .

# ビルド (main.go がルートにある場合)
RUN go build -o main .

# 2. 実行用イメージ
FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# ビルドしたバイナリをコピー
COPY --from=builder /app/main .

# ポート指定
EXPOSE 8080

# 実行
CMD ["./main"]