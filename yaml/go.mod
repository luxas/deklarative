module github.com/luxas/deklarative/yaml

go 1.16

replace (
	github.com/luxas/deklarative/content => ../content
	github.com/luxas/deklarative/json => ../json
	github.com/luxas/deklarative/tracing => ../tracing
)

require (
	github.com/google/go-cmp v0.5.6 // indirect
	github.com/json-iterator/go v1.1.11
	github.com/luxas/deklarative/content v0.0.0-00010101000000-000000000000
	github.com/luxas/deklarative/json v0.0.0-00010101000000-000000000000
	github.com/stretchr/testify v1.7.0
	gopkg.in/check.v1 v1.0.0-20200902074654-038fdea0a05b // indirect
	gopkg.in/yaml.v2 v2.4.0
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b
	k8s.io/apimachinery v0.22.1
	sigs.k8s.io/kustomize/kyaml v0.11.1
	sigs.k8s.io/yaml v1.2.0
)
