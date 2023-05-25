# BUILD PHASE
FROM golang:alpine AS builder

WORKDIR $GOPATH/src/blogotron
COPY . .
RUN go mod download

# Build
RUN GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /go/bin/blogotron

# IMAGE PHASE
FROM alpine:latest

WORKDIR /app

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /go/bin/blogotron /app/blogotron
COPY templates ./templates

EXPOSE 8666

ENTRYPOINT [ "/app/blogotron" ]