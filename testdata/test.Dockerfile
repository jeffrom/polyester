FROM golang:1.16.6

WORKDIR /src

COPY go.mod go.sum ./
RUN set -x; go mod download

COPY . .

RUN set -x; \
    CGO_ENABLED=0 go build \
        -ldflags="-s -w" \
        -o /usr/bin/polyester \
        ./cmd/polyester

RUN set -x; for pkg in $(go list ./...); do go test -c $pkg; done
