FROM golang:1.25.3-alpine AS build

WORKDIR /src
ENV PATH="/usr/local/go/bin:${PATH}"

COPY go.mod go.sum* ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/worker ./cmd/worker
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/vag ./cmd/vag
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/mcp ./cmd/mcp

FROM alpine:3.20

ARG AGENT_IMAGEFLOW_VERSION=""
ARG AGENT_IMAGEFLOW_COMMIT=""
ARG AGENT_IMAGEFLOW_BUILD_TIME=""
ARG AGENT_IMAGEFLOW_IMAGE_TAG=""
ENV AGENT_IMAGEFLOW_VERSION=$AGENT_IMAGEFLOW_VERSION
ENV AGENT_IMAGEFLOW_COMMIT=$AGENT_IMAGEFLOW_COMMIT
ENV AGENT_IMAGEFLOW_BUILD_TIME=$AGENT_IMAGEFLOW_BUILD_TIME
ENV AGENT_IMAGEFLOW_IMAGE_TAG=$AGENT_IMAGEFLOW_IMAGE_TAG

RUN apk add --no-cache libwebp-tools
RUN adduser -D -u 10001 appuser
WORKDIR /app
RUN mkdir -p /data/agent-imageflow && chown -R appuser:appuser /data

COPY --from=build /out/api /app/api
COPY --from=build /out/worker /app/worker
COPY --from=build /out/vag /app/vag
COPY --from=build /out/mcp /app/mcp
COPY examples /app/examples

USER appuser
EXPOSE 8081

CMD ["/app/api"]
