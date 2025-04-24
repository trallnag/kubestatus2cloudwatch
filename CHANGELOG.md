# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0),
and adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0).

## Unreleased

### Added

- Added basic CLI to get version, set path to configuration file, and more.

### Changed

- Upgraded direct and indirect dependencies.
- **BREAKING**: Changed default log level from `debug` to `info`.
- **BREAKING**: Changed log format configuration from boolean `pretty` flag to
  `format` enum with allowed values `json` and `logfmt`.
- **BREAKING**: Changed logging from zerolog to slog using included handlers for
  JSON and logfmt depending on configuration.
- **BREAKING**: Renamed dry run flag from `dry` to `dryRun`.

### Fixed

- Fixed huge bug with the CloudWatch metric being updated using the value of
  `Success` and not `Ready`. The former attribute shows if the scan itself was
  successful, not if the scan targets are ready according to the configuration.

## [1.1.7](https://github.com/trallnag/kubestatus2cloudwatch/compare/v1.1.6...v1.1.7) / 2025-02-24

### Changed

- Added SBOMs to release artifacts. Does not cover container images.
- Added Cosign to release artifacts. Checksums, container images, and container
  manifests are signed.

## [1.1.6](https://github.com/trallnag/kubestatus2cloudwatch/compare/v1.1.5...v1.1.6) / 2025-02-22

### Changed

- Upgraded direct and indirect dependencies.
- Switched license from Apache to ISC.
- Switched container image registry from Docker Hub to GitHub Packages.

## [1.1.5](https://github.com/trallnag/kubestatus2cloudwatch/compare/v1.1.4...v1.1.5) / 2024-03-09

### Changed

- Bumped minimum required Go version to 1.22. This affects release artifacts.
- Upgraded direct and indirect dependencies.

## [1.1.4](https://github.com/trallnag/kubestatus2cloudwatch/compare/v1.1.3...v1.1.4) / 2023-03-25

### Changed

- Upgraded direct and indirect dependencies.

## [1.1.3](https://github.com/trallnag/kubestatus2cloudwatch/compare/v1.1.2...v1.1.3) / 2023-03-05

### Changed

- Bumped minimum required Go version to 1.20. This affects release artifacts.
- Switched from ISC License (ISC) to Apache License (Apache-2.0).
- Upgraded direct and indirect dependencies.

## [1.1.2](https://github.com/trallnag/kubestatus2cloudwatch/compare/v1.1.1...v1.1.2) / 2023-02-20

### Changed

- Upgraded direct and indirect dependencies.
- Switched from MIT license to functionally equivalent ISC license.
- Removed Darwin and Windows binaries from GitHub release artifacts.

## [1.1.1](https://github.com/trallnag/kubestatus2cloudwatch/compare/v1.1.0...v1.1.1) / 2023-02-05

### Changed

- Upgraded indirect dependencies.

## [1.1.0](https://github.com/trallnag/kubestatus2cloudwatch/compare/v1.0.0...v1.1.0) / 2023-01-07

### Added

- Log start and end of tick rounds including duration and identifier of
  individual ticks. Log level info is used for this.

### Changed

- Upgraded indirect dependencies.

### Fixed

- Don't wait one interval amount before starting the first tick round. Now
  Kubestatus2cloudwatch will immediately after setup start to query the
  Kubernetes API and update the CloudWatch metric.

## [1.0.0](https://github.com/trallnag/kubestatus2cloudwatch/compare/ed5965484226b6ef8b1a13de14c82c7b36d33d8d...v1.0.0) / 2022-12-28

Initial release of Kubestatus2cloudwatch. The app is ready for usage in
production environments and major breaking changes are not expected in the near
future.
