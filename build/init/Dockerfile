FROM golang:1.15.8-alpine AS builder

ARG CGO_ENABLED=0
ENV CGO_ENABLED=$CGO_ENABLED

COPY . /src
WORKDIR /src

RUN GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o k8s-pod-mutator-init cmd/init/main.go

###
FROM scratch

WORKDIR /app
COPY --from=builder /src/k8s-pod-mutator-init .

ENTRYPOINT ["/app/k8s-pod-mutator-init"]
