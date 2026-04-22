FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /snsd ./cmd/snsd

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata
COPY --from=build /snsd /usr/local/bin/snsd
EXPOSE 9090
ENTRYPOINT ["snsd"]
