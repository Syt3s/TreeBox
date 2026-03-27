FROM golang:1.20-alpine as builder

WORKDIR /app

ENV CGO_ENABLED=0

ARG GITHUB_SHA=dev

COPY . .

RUN go mod tidy
RUN go build -v -ldflags "-w -s -extldflags '-static' -X 'github.com/syt3s/TreeBox/internal/config.BuildCommit=$GITHUB_SHA'" -o TreeBox ./cmd/treebox

FROM alpine:latest

RUN apk update && apk add tzdata && cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
&& echo "Asia/Shanghai" > /etc/timezone

WORKDIR /home/app

COPY --from=builder /app/TreeBox .

RUN mkdir -p /home/app/configs
RUN chmod 777 /home/app/TreeBox

ENTRYPOINT ["./TreeBox", "web"]
EXPOSE 8080
