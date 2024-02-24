FROM golang:latest as builder
COPY . /src
RUN cd /src && \
    GO111MODULE=on CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./build/main main.go

# =====
FROM alpine:latest
COPY --from=builder /src/build/main /usr/bin/main
CMD ["/usr/bin/main","start"]

