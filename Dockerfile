FROM golang:1.17 AS builder

WORKDIR /go/src/github.com/limgit/k8s-sidecar-controller
COPY . ./
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o main ./cmd

FROM scratch

COPY --from=builder /go/src/github.com/limgit/k8s-sidecar-controller/main ./
CMD ["./main"]
