FROM golang:1.20.5-alpine

RUN mkdir /app
COPY ./ /app
WORKDIR /app

RUN go build -o envoy-sds cmd/main.go

CMD ["/app/envoy-sds"]