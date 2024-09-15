FROM golang:1.22.0-alpine3.18 as builder

WORKDIR /app

COPY go.mod .

RUN go mod download

COPY . .

RUN go build -o /web_notes

COPY docker.env .env

FROM alpine:latest

COPY --from=builder /web_notes /web_notes
COPY --from=builder /app/.env ./.env
COPY public/ /public

EXPOSE 8080

CMD ["/web_notes"]