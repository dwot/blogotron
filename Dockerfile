FROM golang:alpine

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .
COPY .env.example .env
COPY config.yml.example config.yml

RUN go build -o /blogotron

EXPOSE 8666

CMD [ "/blogotron" ]