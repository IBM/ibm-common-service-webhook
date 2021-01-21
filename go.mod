module github.com/IBM/ibm-common-service-webhook

go 1.15

require (
	github.com/IBM/operand-deployment-lifecycle-manager v1.4.1
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32
	github.com/operator-framework/operator-lifecycle-manager v0.0.0-20200321030439-57b580e57e88
	k8s.io/api v0.18.6
	k8s.io/apimachinery v0.18.6
	k8s.io/klog v1.0.0
	sigs.k8s.io/controller-runtime v0.6.2
)

// fix vulnerability: CVE-2021-3121 in github.com/gogo/protobuf < v1.3.2
replace github.com/gogo/protobuf => github.com/gogo/protobuf v1.3.2
