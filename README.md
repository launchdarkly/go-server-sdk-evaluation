# LaunchDarkly Go SDK Evaluation Engine

[![Circle CI](https://circleci.com/gh/launchdarkly/go-server-sdk-evaluation.svg?style=shield)](https://circleci.com/gh/launchdarkly/go-server-sdk-evaluation) [![Documentation](https://img.shields.io/static/v1?label=go.dev&message=reference&color=00add8)](https://pkg.go.dev/gopkg.in/launchdarkly/go-server-sdk-evaluation.v1)

## This is a prerelease repository

This project currently contains prerelease code to support development of Go SDK 5.0.0.

## Overview

This repository contains the internal feature flag evaluation logic and data model used by the [LaunchDarkly Go SDK](https://github.com/launchdarkly/go-server-sdk). It is packaged separately because it is also used by internal LaunchDarkly components. Applications using the LaunchDarkly Go SDK should not need to reference this package directly.

Note that the base import path is `gopkg.in/launchdarkly/go-server-sdk-evaluation.v1`, not `github.com/launchdarkly/go-server-sdk-evaluation`. This ensures that the package can be referenced not only as a Go module, but also by projects that use older tools like `dep` and `govendor`, because the 5.x release of the Go SDK supports either module or non-module usage. Future releases of this package, and of the Go SDK, may drop support for non-module usage.

## Supported Go versions

This version of the project has been tested with Go 1.13 through 1.14.

## Learn more

Check out our [documentation](http://docs.launchdarkly.com) for in-depth instructions on configuring and using LaunchDarkly. You can also head straight to the [complete reference guide for the Go SDK](http://docs.launchdarkly.com/docs/go-sdk-reference), or the [generated API documentation](https://pkg.go.dev/gopkg.in/launchdarkly/go-server-sdk-evaluation.v1) for this project.

## Contributing

We encourage pull requests and other contributions from the community. Check out our [contributing guidelines](CONTRIBUTING.md) for instructions on how to contribute to this SDK.

## About LaunchDarkly

* LaunchDarkly is a continuous delivery platform that provides feature flags as a service and allows developers to iterate quickly and safely. We allow you to easily flag your features and manage them from the LaunchDarkly dashboard.  With LaunchDarkly, you can:
    * Roll out a new feature to a subset of your users (like a group of users who opt-in to a beta tester group), gathering feedback and bug reports from real-world use cases.
    * Gradually roll out a feature to an increasing percentage of users, and track the effect that the feature has on key metrics (for instance, how likely is a user to complete a purchase if they have feature A versus feature B?).
    * Turn off a feature that you realize is causing performance problems in production, without needing to re-deploy, or even restart the application with a changed configuration file.
    * Grant access to certain features based on user attributes, like payment plan (eg: users on the ‘gold’ plan get access to more features than users in the ‘silver’ plan). Disable parts of your application to facilitate maintenance, without taking everything offline.
* LaunchDarkly provides feature flag SDKs for a wide variety of languages and technologies. Check out [our documentation](https://docs.launchdarkly.com/docs) for a complete list.
* Explore LaunchDarkly
    * [launchdarkly.com](https://www.launchdarkly.com/ "LaunchDarkly Main Website") for more information
    * [docs.launchdarkly.com](https://docs.launchdarkly.com/  "LaunchDarkly Documentation") for our documentation and SDK reference guides
    * [apidocs.launchdarkly.com](https://apidocs.launchdarkly.com/  "LaunchDarkly API Documentation") for our API documentation
    * [blog.launchdarkly.com](https://blog.launchdarkly.com/  "LaunchDarkly Blog Documentation") for the latest product updates
    * [Feature Flagging Guide](https://github.com/launchdarkly/featureflags/  "Feature Flagging Guide") for best practices and strategies
