FROM gcr.io/distroless/base:nonroot

COPY kubestatus2cloudwatch /app/kubestatus2cloudwatch

ENTRYPOINT ["/app/kubestatus2cloudwatch"]
