module gopkg.in/launchdarkly/go-server-sdk-evaluation.v1

go 1.13

require (
	github.com/launchdarkly/go-semver v1.0.2
	github.com/mailru/easyjson v0.7.6
	github.com/stretchr/testify v1.6.1
	gopkg.in/launchdarkly/go-jsonstream.v1 v1.0.0
	gopkg.in/launchdarkly/go-sdk-common.v2 v2.2.0
	gopkg.in/launchdarkly/go-sdk-events.v1 v1.0.1
)

replace gopkg.in/launchdarkly/go-sdk-common.v2 => github.com/launchdarkly/go-sdk-common-private/v2 v2.2.3-0.20210319203906-533467f10d12
