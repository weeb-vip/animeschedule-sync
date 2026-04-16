FROM golang:1.23 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod tidy

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/main.go

FROM gcr.io/distroless/static-debian10

WORKDIR /app

COPY --from=builder --chown=nonroot:nonroot /app/main .

ARG VERSION
ENV VERSION=$VERSION

USER nonroot

CMD ["./main", "serve"]
