# Change log

All notable changes to the project will be documented in this file. This project adheres to [Semantic Versioning](http://semver.org).

## [1.2.2] - 2021-06-03
### Fixed:
- The 1.2.1 release updated the `go-sdk-common` dependency, but not to the latest version that actually contained the relevant bugfix. This release updates to the latest.

## [1.2.1] - 2021-06-03
### Fixed:
- Updated `go-jsonstream` to [v1.0.1](https://github.com/launchdarkly/go-jsonstream/releases/tag/1.0.1) to incorporate a bugfix in JSON number parsing.

## [1.2.0] - 2021-02-26
### Added:
- New `Generation` field in `ldmodel.Segment`.

## [1.1.2] - 2021-02-11
### Fixed:
- When deserializing feature flags from JSON, an explicit null value for the `rollout` property (as opposed to just omitting the property) was being treated as an error. The LaunchDarkly service endpoints do not ever send `rollout: null`, but it should be considered valid if encountered in JSON from some other source.

## [1.1.1] - 2021-01-20
### Fixed:
- When using semantic version operators, semantic version strings were being rejected by the parser if they contained a zero digit in any position _after_ the first character of a numeric version component. For instance, `0.1.2` and `1.2.3` were accepted, and `01.2.3` was correctly rejected (leading zeroes for nonzero values are not allowed), but `10.2.3` was incorrectly rejected.

## [1.1.0] - 2020-12-17
### Added:
- In `ldmodel`, there are now additional JSON marshaling and unmarshaling methods that can interact directly with `go-jsonstream` writers and readers (see below).
- You can now get automatic integration with EasyJSON by setting the build tag `launchdarkly_easyjson`, which causes `MarshalEasyJSON` and `UnmarshalEasyJSON` methods to be added to `FeatureFlag` and `Segment`.

### Changed:
- The internal JSON serialization logic now uses [`go-jsonstream`](https://github.com/launchdarkly/go-jsonstream) instead of the deprecated `go-sdk-common.v2/jsonstream`.
- The internal JSON deserialization logic, which previously used `encoding/json`, now uses `go-jsonstream` for a considerable increase in efficiency.

## [1.0.1] - 2020-10-08
### Fixed:
- When serializing flags and segments to JSON, properties with default values (such as false booleans or empty arrays) were being dropped entirely to save bandwidth. However, these representations may be consumed by SDKs other than the Go SDK, and some of the LaunchDarkly SDKs do not tolerate missing properties, so this has been fixed to remain consistent with the less efficient behavior of Go SDK 4.x.

## [1.0.0] - 2020-09-18
Initial release of this flag evaluation support code that will be used with versions 5.0.0 and above of the LaunchDarkly Server-Side SDK for Go.
