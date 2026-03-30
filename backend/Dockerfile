ARG GO_VERSION
ARG ALPINE_VERSION

FROM golang:${GO_VERSION}-alpine AS builder
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories
RUN apk add --no-cache git ca-certificates tzdata
ENV CGO_ENABLED=0
ENV GOPROXY=https://mirrors.aliyun.com/goproxy/,direct

# 构建 context 是项目根目录，先复制 proto module（replace directive 需要）
WORKDIR /src
COPY proto/go/ /src/proto/go/
COPY backend/go.mod backend/go.sum /src/backend/

WORKDIR /src/backend
RUN go mod download

# 复制 backend 源码
COPY backend/ /src/backend/

RUN go build -ldflags="-w -s" -o /oreader ./cmd/server

FROM alpine:${ALPINE_VERSION}
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories
RUN apk add --no-cache ca-certificates wget tzdata
COPY --from=builder /oreader /oreader
RUN addgroup -g 1000 oreader && adduser -D -u 1000 -G oreader oreader
RUN mkdir -p /data && chown -R oreader:oreader /data
USER oreader
EXPOSE 8080
HEALTHCHECK CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1
ENTRYPOINT ["/oreader"]
