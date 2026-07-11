FROM gcr.io/distroless/base:nonroot@sha256:b78832f41c8128046807c24840ebee4f1c18ba7870eed423d8750c272c15e147

COPY kubestatus2cloudwatch /app/kubestatus2cloudwatch

ENTRYPOINT ["/app/kubestatus2cloudwatch"]
