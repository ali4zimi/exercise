FROM golang:1.22

WORKDIR /app

COPY go.mod go.sum ./

COPY . .

RUN go mod tidy

RUN go build -o main ./cmd/main.go

CMD ["/app/main"]
