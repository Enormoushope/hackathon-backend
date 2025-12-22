FROM golang:1.24-alpine AS builder

# ビルド環境の作業ディレクトリ
WORKDIR /app

# Dockerfileが backend/ 内にある場合、
# COPYの起点は「ビルドコンテキスト（通常はリポジトリのルート）」になります。
COPY go.mod go.sum ./
RUN go mod download

# backend フォルダの中身をすべてコピー
COPY backend/ .

# ビルドを実行（同じ階層に main.go がある想定）
RUN go build -o main .

# --- 実行用イメージ ---
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/main .

EXPOSE 8080
CMD ["./main"]