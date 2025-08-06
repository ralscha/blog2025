FROM golang:1.24-alpine
RUN apk add --no-cache protobuf protobuf-dev
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
WORKDIR /app

# Create entrypoint script
RUN echo '#!/bin/sh' > /entrypoint.sh && \
    echo 'mkdir -p /app/publisher/mqttdemo' >> /entrypoint.sh && \
    echo 'protoc --proto_path=/app/schema --go_out=/app/publisher /app/schema/*.proto' >> /entrypoint.sh && \
    chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
