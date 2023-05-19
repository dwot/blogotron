FROM golang:alpine

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .

RUN go build -o /blogotron

EXPOSE 8666

CMD [ "/blogotron" ]