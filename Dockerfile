FROM golang:1.19.8 as build

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY *.go ./

RUN CGO_ENABLED=0 go build -o /tendermint-exporter

FROM alpine:latest as run

WORKDIR /app

COPY --from=build /app/tendermint-exporter .

ENTRYPOINT [ "tendermint-exporter" ]