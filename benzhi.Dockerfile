FROM golang:1.23.12-bookworm

ENV GOFLAGS=-mod=vendor
ENV GOPROXY=off
ENV GOSUMDB=off
ENV CGO_ENABLED=0

WORKDIR /app

COPY go.mod go.sum ./
COPY vendor ./vendor
COPY cmd ./cmd
COPY internal ./internal
COPY web ./web

RUN go build -mod=vendor -o /bin/graphstore ./cmd/graphstore

EXPOSE 8612
CMD ["/bin/graphstore", "-addr", "0.0.0.0:8612", "-data", "/tmp/graphstore-data"]
