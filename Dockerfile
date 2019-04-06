FROM golang:alphine

COPY . /app/
WORKDIR /app

RUN go build -o main .
CMD ["/app/main"]
EXPOSE 8080
