FROM golang:1.23

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod tidy

COPY . .

RUN go build -o scroll-api main.go

EXPOSE 8080
CMD ["./scroll-api"]