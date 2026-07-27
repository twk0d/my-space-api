# syntax=docker/dockerfile:1

ARG GO_VERSION=1.26.5

FROM golang:${GO_VERSION}-alpine AS build
WORKDIR /src

COPY go.work ./
COPY apps/api/go.mod ./apps/api/
RUN go -C apps/api mod download

COPY apps/api ./apps/api
RUN CGO_ENABLED=0 GOOS=linux go -C apps/api build -trimpath -ldflags="-s -w" -o /out/my-space-api ./cmd/api

FROM alpine:3.22
WORKDIR /app

RUN addgroup -S app && adduser -S app -G app
COPY --from=build /out/my-space-api /app/my-space-api

USER app
EXPOSE 8080

ENV API_ADDR=:8080
ENV API_DATA_PATH=/tmp/my-space-api.json

CMD ["/app/my-space-api"]
