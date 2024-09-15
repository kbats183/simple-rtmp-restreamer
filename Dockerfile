FROM golang:1.21-alpine AS build
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o simple-rtmp-restreamer github.com/kbats183/simple-rtmp-restreamer/cmd/server

# Stage 2: Final stage
FROM alpine:3.14

WORKDIR /app

COPY --from=build /app/simple-rtmp-restreamer .

RUN apk --no-cache add ca-certificates tzdata

ENTRYPOINT ["/app/simple-rtmp-restreamer"]
