module playrix.com/it/nomad-cluster-scalerv2

go 1.19

replace github.com/Pramod-Devireddy/go-exprtk => github.com/tantra35/go-exprtk v0.0.0-20231123162352-b4e9b2057c6d

require (
	github.com/Pramod-Devireddy/go-exprtk v1.1.0
	github.com/armon/go-metrics v0.4.0
	github.com/aws/aws-sdk-go-v2 v1.21.0
	github.com/aws/aws-sdk-go-v2/config v1.18.26
	github.com/aws/aws-sdk-go-v2/service/autoscaling v1.28.8
	github.com/aws/aws-sdk-go-v2/service/ec2 v1.115.0
	github.com/aws/smithy-go v1.14.2
	github.com/chrusty/go-tableprinter v0.0.0-20190528113659-0de6c8f09400
	github.com/dustin/go-humanize v1.0.1
	github.com/hashicorp/go-hclog v1.5.0
	github.com/hashicorp/go-plugin v1.5.1
	github.com/hashicorp/hcl v1.0.1-0.20201016140508-a07e7d50bbee
	github.com/hashicorp/nomad v1.1.18
	github.com/hashicorp/nomad/api v0.0.0-20220614183018-69a1ba9ffcd3
	github.com/jessevdk/go-flags v1.5.0
	github.com/jinzhu/copier v0.3.5
	github.com/mitchellh/hashstructure v1.1.0
	github.com/mitchellh/mapstructure v1.4.3
	github.com/spf13/cast v1.5.1
	google.golang.org/grpc v1.53.0
	google.golang.org/protobuf v1.28.2-0.20230222093303-bc1253ad3743
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/agext/levenshtein v1.2.1 // indirect
	github.com/apparentlymart/go-cidr v1.0.1 // indirect
	github.com/apparentlymart/go-textseg/v13 v13.0.0 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.13.25 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.13.4 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.1.41 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.4.35 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.3.35 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.9.35 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.12.11 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.14.11 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.19.1 // indirect
	github.com/bmatcuk/doublestar v1.1.5 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/fatih/color v1.13.0 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/go-cmp v0.5.9 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/hashicorp/consul/api v1.9.1 // indirect
	github.com/hashicorp/cronexpr v1.1.1 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-cty-funcs v0.0.0-20200930094925-2721b1e36840 // indirect
	github.com/hashicorp/go-immutable-radix v1.3.1 // indirect
	github.com/hashicorp/go-memdb v1.3.3 // indirect
	github.com/hashicorp/go-msgpack v1.1.5 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-rootcerts v1.0.2 // indirect
	github.com/hashicorp/go-sockaddr v1.0.2 // indirect
	github.com/hashicorp/go-version v1.2.1-0.20191009193637-2046c9d0f0b0 // indirect
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/hashicorp/hcl/v2 v2.9.2-0.20210407182552-eb14f8319bdc // indirect
	github.com/hashicorp/raft v1.1.3-0.20200211192230-365023de17e6 // indirect
	github.com/hashicorp/serf v0.9.5 // indirect
	github.com/hashicorp/yamux v0.0.0-20181012175058-2f1d1f20f75d // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/mattn/go-colorable v0.1.12 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/mattn/go-runewidth v0.0.7 // indirect
	github.com/miekg/dns v1.1.26 // indirect
	github.com/mitchellh/copystructure v1.1.1 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/go-testing-interface v1.14.1 // indirect
	github.com/mitchellh/go-wordwrap v1.0.1 // indirect
	github.com/mitchellh/reflectwalk v1.0.1 // indirect
	github.com/oklog/run v1.0.1-0.20180308005104-6934b124db28 // indirect
	github.com/olekukonko/tablewriter v0.0.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/ryanuber/go-glob v1.0.0 // indirect
	github.com/stretchr/testify v1.7.2 // indirect
	github.com/zclconf/go-cty v1.13.0 // indirect
	github.com/zclconf/go-cty-yaml v1.0.2 // indirect
	golang.org/x/crypto v0.0.0-20220517005047-85d78b3ac167 // indirect
	golang.org/x/net v0.7.0 // indirect
	golang.org/x/sys v0.5.0 // indirect
	golang.org/x/text v0.11.0 // indirect
	google.golang.org/genproto v0.0.0-20230110181048-76db0878b65f // indirect
)
