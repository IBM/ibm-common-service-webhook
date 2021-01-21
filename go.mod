module github.com/IBM/ibm-common-service-webhook

go 1.14

require (
	github.com/operator-framework/operator-lifecycle-manager v0.0.0-20200321030439-57b580e57e88
	github.com/operator-framework/operator-sdk v0.17.0
	github.com/spf13/pflag v1.0.5
	k8s.io/api v0.17.4
	k8s.io/apimachinery v0.17.4
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/klog v1.0.0
	sigs.k8s.io/controller-runtime v0.5.2
)

replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.3.2+incompatible // Required by OLM
	github.com/gogo/protobuf => github.com/gogo/protobuf v1.3.2 // fix vulnerability: CVE-2021-3121 in github.com/gogo/protobuf < v1.3.2
	k8s.io/client-go => k8s.io/client-go v0.17.4 // Required by prometheus-operator

)
