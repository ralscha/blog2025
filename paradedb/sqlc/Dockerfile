FROM golang:1.26.1

RUN go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.30.0

WORKDIR /src/

CMD ["sh", "-lc", "/go/bin/sqlc generate -f /src/sqlc/sqlc.yaml"]