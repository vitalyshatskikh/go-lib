FROM golang:1.26 AS builder
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /restapi ./examples/restapi/
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /loadgen ./examples/loadgen/
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /sentry-mock ./examples/sentry-mock/

FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /restapi /restapi
COPY --from=builder /loadgen /loadgen
COPY --from=builder /sentry-mock /sentry-mock
EXPOSE 8080 8081
ENTRYPOINT ["/restapi"]
