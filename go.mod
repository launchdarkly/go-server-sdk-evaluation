module gopkg.in/launchdarkly/go-server-sdk-evaluation.v1

go 1.13

require (
	github.com/launchdarkly/go-jsonstream v0.0.1-alpha.3
	github.com/launchdarkly/go-semver v1.0.1
	github.com/stretchr/testify v1.6.1
	gopkg.in/launchdarkly/go-sdk-common.v2 v2.1.0-alpha.1
	gopkg.in/launchdarkly/go-sdk-events.v1 v1.0.1-alpha.1
)

replace gopkg.in/launchdarkly/go-sdk-common.v2 => github.com/launchdarkly/go-sdk-common-private/v2 v2.1.0-alpha.1
replace gopkg.in/launchdarkly/go-sdk-events.v1 => github.com/launchdarkly/go-sdk-events-private v1.0.1-alpha.2
