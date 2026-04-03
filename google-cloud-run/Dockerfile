FROM golang:1.26.1 AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY server ./server
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/temperature-service ./server

FROM gcr.io/distroless/static-debian13:nonroot
WORKDIR /
COPY --from=build /out/temperature-service /temperature-service
EXPOSE 8080
ENTRYPOINT ["/temperature-service"]
