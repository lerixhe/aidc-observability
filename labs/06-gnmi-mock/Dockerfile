FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod ./
COPY *.go ./
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o mock-leaf .

FROM alpine:3.19
RUN apk add --no-cache curl
WORKDIR /app
COPY --from=builder /app/mock-leaf .

EXPOSE 9339
ENTRYPOINT ["/app/mock-leaf"]
CMD ["-serve", "-addr", ":9339", "-speed", "1"]
