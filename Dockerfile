FROM golang:1.21.1 AS builder

COPY ./ /app
WORKDIR /app

RUN go build -o gemini main.go
RUN mkdir -p /tmp/app
RUN cp gemini /tmp/app && chmod +x /tmp/app/gemini

FROM alpine:latest
COPY --from=builder /tmp/app /app

CMD ["/app/gemini"]