
FROM golang:1.21

WORKDIR /app

COPY . .

RUN go build -o scroll-api main.go

CMD ["./scroll-api"]
