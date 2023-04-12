FROM golang:1.20-alpine3.17 AS builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 go build -trimpath

FROM alpine:3.17
WORKDIR /app
COPY --from=builder /app/PAL9000 .
ENTRYPOINT ["/app/PAL9000"]
