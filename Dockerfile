FROM golang:1.22-alpine as builder
RUN apk add build-base
WORKDIR /rates-emailer
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ cmd
COPY ratesmail/ ratesmail
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags '-extldflags "-static"' -o rates-emailer ./cmd/server
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags '-extldflags "-static"' -o publish-newsletter ./cmd/mailer

FROM alpine
ENV APP_MODE=prod
WORKDIR /rates-emailer
RUN mkdir data
COPY conf/config.prod.toml conf/
COPY entrypoint.sh ./
COPY --from=builder /rates-emailer/rates-emailer /rates-emailer/publish-newsletter ./
CMD ["./entrypoint.sh"]

