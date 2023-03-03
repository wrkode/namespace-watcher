FROM golang:1.19-alpine

RUN apk add --update --no-cache ca-certificates

WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN go build -o namespace-watcher .

CMD ["./namespace-watcher"]