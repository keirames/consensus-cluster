# Build Stage

FROM golang:1.21.3-alpine3.18 AS BuildStage

WORKDIR /app

COPY . .

RUN go mod download

RUN go build -o /build main.go

# Deploy State

FROM alpine:latest

WORKDIR /

COPY --from=BuildStage /build /build

EXPOSE 8000

USER nonroot:nonroot

ENTRYPOINT ["/build"]