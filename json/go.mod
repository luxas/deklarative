module github.com/luxas/deklarative/json

go 1.16

replace github.com/luxas/deklarative/content => ../content

require (
	github.com/json-iterator/go v1.1.11
	github.com/luxas/deklarative/content v0.0.0-00010101000000-000000000000
	github.com/modern-go/reflect2 v1.0.1
	github.com/stretchr/testify v1.7.0
	k8s.io/apimachinery v0.22.1
)
