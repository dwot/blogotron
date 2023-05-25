# BUILD PHASE
FROM golang:alpine AS builder

# Install git.
#RUN apk update && apk add --no-cache git

WORKDIR $GOPATH/src/blogotron
COPY . .
RUN go mod download

# Build
RUN GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /go/bin/blogotron

# IMAGE PHASE
FROM scratch

WORKDIR /app

COPY --from=builder /go/bin/blogotron /app/blogotron
COPY templates ./templates

EXPOSE 8666

ENTRYPOINT [ "/app/blogotron" ]



