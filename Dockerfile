# ============ 阶段1：编译 ============
FROM golang:1.25-alpine AS builder

ENV GO111MODULE=on \
    GOPROXY=https://goproxy.cn,direct \
    CGO_ENABLED=0 \
    GOOS=linux

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o /app/bin/user-service ./cmd/user-service/
RUN go build -o /app/bin/market-gateway ./cmd/market-gateway/
RUN go build -o /app/bin/trading-engine ./cmd/trading-engine/
RUN go build -o /app/bin/data-service ./cmd/data-service/


# ============ 用户服务 ============
FROM alpine:3.20 AS user-service
RUN apk --no-cache add ca-certificates tzdata
ENV TZ=Asia/Shanghai
COPY --from=builder /app/bin/user-service /app/user-service
EXPOSE 8081
CMD ["/app/user-service"]


# ============ 行情网关 ============
FROM alpine:3.20 AS market-gateway
RUN apk --no-cache add ca-certificates tzdata
ENV TZ=Asia/Shanghai
WORKDIR /app
COPY --from=builder /app/bin/market-gateway /app/market-gateway
COPY web/ /app/web/
EXPOSE 8082
CMD ["/app/market-gateway"]


# ============ 交易引擎 ============
FROM alpine:3.20 AS trading-engine
RUN apk --no-cache add ca-certificates tzdata
ENV TZ=Asia/Shanghai
COPY --from=builder /app/bin/trading-engine /app/trading-engine
EXPOSE 8083
CMD ["/app/trading-engine"]


# ============ 数据服务 ============
FROM alpine:3.20 AS data-service
RUN apk --no-cache add ca-certificates tzdata
ENV TZ=Asia/Shanghai
COPY --from=builder /app/bin/data-service /app/data-service
EXPOSE 8084
CMD ["/app/data-service"]