FROM golang:1.15-alpine
RUN apk --no-cache add tzdata
ENV GOPROXY=https://goproxy.cn
WORKDIR /app
COPY . /app
RUN go build -o wrg .

CMD ["/app/wrg"]
