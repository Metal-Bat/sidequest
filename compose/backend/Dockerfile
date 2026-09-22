FROM golang:1.27.1-bookworm AS build
WORKDIR /app
COPY src/go.mod src/go.sum ./
RUN go mod download
COPY src/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /sidequest .

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*
COPY --from=build /sidequest /sidequest
COPY compose/backend/start /start
RUN chmod +x /start
USER 65532:65532
EXPOSE 8000
ENTRYPOINT ["/start"]
