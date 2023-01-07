# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0),
and adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0),

## Unreleased

### Added

- Log start and end of tick rounds including duration and identifier of
  individual ticks. Log level info is used for this.

### Changed

- Updated indirect dependencies.

### Fixed

- Don't wait one interval amount before starting the first tick round. Now
  KubeStatus2CloudWatch will immediately after setup start to query the
  Kubernetes API and update the CloudWatch metric.

## [1.0.0](https://github.com/trallnag/kubestatus2cloudwatch/compare/ed5965484226b6ef8b1a13de14c82c7b36d33d8d...v1.0.0) / 2022-12-28

Initial release of KubeStatus2CloudWatch. The app is ready for usage in
production environments and major breaking changes are not expected in the near
future.
