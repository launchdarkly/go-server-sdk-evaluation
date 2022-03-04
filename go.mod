module gopkg.in/launchdarkly/go-server-sdk-evaluation.v2

go 1.16

require (
	github.com/launchdarkly/go-semver v1.0.2
	github.com/mailru/easyjson v0.7.6
	github.com/stretchr/testify v1.6.1
	gopkg.in/launchdarkly/go-jsonstream.v1 v1.0.1
	gopkg.in/launchdarkly/go-sdk-common.v3 v3.0.0
	gopkg.in/launchdarkly/go-sdk-events.v2 v2.0.0
)

replace gopkg.in/launchdarkly/go-sdk-common.v3 => github.com/launchdarkly/go-sdk-common-private/v3 v3.0.0-alpha.3

replace gopkg.in/launchdarkly/go-sdk-events.v2 => github.com/launchdarkly/go-sdk-events-private/v2 v2.0.0-alpha.1
