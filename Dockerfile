FROM gcr.io/distroless/base:nonroot@sha256:7088bf1cdc94a65af4e0d5ad94518ed489a372dec1ece38ff4411c6c15fb5864

COPY kubestatus2cloudwatch /app/kubestatus2cloudwatch

ENTRYPOINT ["/app/kubestatus2cloudwatch"]
