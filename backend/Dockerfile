FROM golang:1.24.3-alpine

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN find . -name "main.go" -type f && ls -la

RUN go build -v -o main .

RUN chmod +x ./main && ls -la ./main

EXPOSE 8080

CMD ["./main"]