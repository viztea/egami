# Builder
FROM golang:1.25-alpine AS build

ARG VERSION=dev

WORKDIR /tmp/egami

COPY . .

RUN apk add --no-cache git && \
    go mod download && \
    go mod verify && \
    go build -o egami

# Runner
FROM alpine:latest

WORKDIR /opt/egami

COPY --from=build /tmp/egami/egami /opt/egami/egami

EXPOSE 3333

CMD [ "./egami" ]