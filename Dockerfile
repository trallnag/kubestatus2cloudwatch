FROM gcr.io/distroless/base:nonroot@sha256:0a0dc2036b7c56d1a9b6b3eed67a974b6d5410187b88cbd6f1ef305697210ee2

COPY kubestatus2cloudwatch /app/kubestatus2cloudwatch

ENTRYPOINT ["/app/kubestatus2cloudwatch"]
