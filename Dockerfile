FROM golang as builder

RUN mkdir -p /app

WORKDIR /app

COPY go.mod .
COPY go.sum .

ENV GOPROXY https://goproxy.cn,direct
ENV CGO_ENABLED=0

RUN go mod download

COPY . .

RUN GOOS=linux go build -o /bin/app .

FROM alpine
LABEL maintainer="mail@dongfg.com"

RUN sed -i "s@https://dl-cdn.alpinelinux.org/@https://repo.huaweicloud.com/@g" /etc/apk/repositories && \
    apk update && \
    apk add --no-cache tzdata

ENV TZ=Asia/Shanghai

COPY --from=builder /bin/app /bin/app

ENTRYPOINT ["/bin/app"]