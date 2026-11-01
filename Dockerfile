FROM gcr.io/distroless/base:nonroot@sha256:9039a88709bcc2bacf5a2faac331d65dd6840f763f225ad9594cc0c67bcd0fba

COPY kubestatus2cloudwatch /app/kubestatus2cloudwatch

ENTRYPOINT ["/app/kubestatus2cloudwatch"]
