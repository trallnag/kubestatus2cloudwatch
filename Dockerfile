FROM gcr.io/distroless/base:nonroot@sha256:e00da4d3bd422820880b080115b3bad24349bef37ed46d68ed0d13e150dc8d67

COPY kubestatus2cloudwatch /app/kubestatus2cloudwatch

ENTRYPOINT ["/app/kubestatus2cloudwatch"]
