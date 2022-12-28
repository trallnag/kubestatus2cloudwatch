FROM gcr.io/distroless/base-debian11:nonroot

ARG TARGETOS
ARG TARGETARCH

WORKDIR /app

COPY dist/*-${TARGETOS}-${TARGETARCH}/kubestatus2cloudwatch* /app/kubestatus2cloudwatch

ENTRYPOINT ["/app/kubestatus2cloudwatch"]
