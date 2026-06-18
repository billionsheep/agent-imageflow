FROM golang:1.25.3-alpine AS build

WORKDIR /src
ENV PATH="/usr/local/go/bin:${PATH}"

COPY go.mod go.sum* ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/worker ./cmd/worker
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/vag ./cmd/vag

FROM alpine:3.20

RUN adduser -D -u 10001 appuser
WORKDIR /app
RUN mkdir -p /data/agent-imageflow && chown -R appuser:appuser /data

COPY --from=build /out/api /app/api
COPY --from=build /out/worker /app/worker
COPY --from=build /out/vag /app/vag
COPY examples /app/examples

USER appuser
EXPOSE 8081

CMD ["/app/api"]
