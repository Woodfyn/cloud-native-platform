FROM golang:1.27-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o server .


FROM scratch

WORKDIR /app

COPY --from=builder /app/server /app/server

USER 10001:10001

EXPOSE 3000

CMD ["/app/server"]