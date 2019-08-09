module github.com/cuijxin/kooper-atom

go 1.12

require (
	github.com/cuijxin/redis-operator-atom v0.0.0-20190809064942-b584645cd32c
	github.com/opentracing/opentracing-go v1.1.0
	github.com/prometheus/client_golang v1.1.0
	github.com/spotahome/kooper v0.6.0
	github.com/stretchr/testify v1.3.0
	golang.org/x/sync v0.0.0-20190227155943-e225da77a7e6
	k8s.io/api v0.0.0-20190802060718-d0d4f3afa3ab
	k8s.io/apiextensions-apiserver v0.0.0-20190330190201-4cac3cbacb4e
	k8s.io/apimachinery v0.0.0-20190802060556-6fa4771c83b3
	k8s.io/client-go v11.0.0+incompatible
	k8s.io/utils v0.0.0-20190809000727-6c36bc71fc4a // indirect
)

replace (
	k8s.io/api => k8s.io/api v0.0.0-20190313235455-40a48860b5ab
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190404173353-6a84e37a896d
	k8s.io/client-go => k8s.io/client-go v11.0.0+incompatible
)
