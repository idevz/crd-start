module github.com/idevz/crd-start

go 1.12

require (
	github.com/imdario/mergo v0.3.7 // indirect
	github.com/kisielk/errcheck v1.2.0 // indirect
	github.com/mitchellh/go-homedir v1.1.0
	github.com/spf13/cobra v0.0.5
	github.com/spf13/viper v1.4.0
	k8s.io/api v0.0.0
	k8s.io/apimachinery v0.0.0
	k8s.io/client-go v0.0.0
	k8s.io/klog v0.4.0
	k8s.io/kubernetes v1.16.0-alpha.2.0.20190730231111-75d51896125b
	k8s.io/utils v0.0.0-20190829053155-3a4a5477acf8 // indirect
)

replace (
	k8s.io/api => k8s.io/kubernetes/staging/src/k8s.io/api v1.16.0-alpha.2.0.20190730231111-75d51896125b
	k8s.io/apiextensions-apiserver => k8s.io/kubernetes/staging/src/k8s.io/apiextensions-apiserver v1.16.0-alpha.2.0.20190730231111-75d51896125b
	k8s.io/apimachinery => k8s.io/kubernetes/staging/src/k8s.io/apimachinery v1.16.0-alpha.2.0.20190730231111-75d51896125b
	k8s.io/apiserver => k8s.io/kubernetes/staging/src/k8s.io/apiserver v1.16.0-alpha.2.0.20190730231111-75d51896125b
	k8s.io/cli-runtime => k8s.io/kubernetes/staging/src/k8s.io/cli-runtime v1.16.0-alpha.2.0.20190730231111-75d51896125b
	k8s.io/client-go => k8s.io/kubernetes/staging/src/k8s.io/client-go v1.16.0-alpha.2.0.20190730231111-75d51896125b
	k8s.io/cloud-provider => k8s.io/kubernetes/staging/src/k8s.io/cloud-provider v1.16.0-alpha.2.0.20190730231111-75d51896125b
	k8s.io/cluster-bootstrap => k8s.io/kubernetes/staging/src/k8s.io/cluster-bootstrap v1.16.0-alpha.2.0.20190730231111-75d51896125b
	k8s.io/code-generator => k8s.io/kubernetes/staging/src/k8s.io/code-generator v1.16.0-alpha.2.0.20190730231111-75d51896125b
	k8s.io/component-base => k8s.io/kubernetes/staging/src/k8s.io/component-base v1.16.0-alpha.2.0.20190730231111-75d51896125b
	k8s.io/cri-api => k8s.io/kubernetes/staging/src/k8s.io/cri-api v1.16.0-alpha.2.0.20190730231111-75d51896125b
	k8s.io/csi-translation-lib => k8s.io/kubernetes/staging/src/k8s.io/csi-translation-lib v1.16.0-alpha.2.0.20190730231111-75d51896125b
	k8s.io/kube-aggregator => k8s.io/kubernetes/staging/src/k8s.io/kube-aggregator v1.16.0-alpha.2.0.20190730231111-75d51896125b
	k8s.io/kube-controller-manager => k8s.io/kubernetes/staging/src/k8s.io/kube-controller-manager v1.16.0-alpha.2.0.20190730231111-75d51896125b
	k8s.io/kube-proxy => k8s.io/kubernetes/staging/src/k8s.io/kube-proxy v1.16.0-alpha.2.0.20190730231111-75d51896125b
	k8s.io/kube-scheduler => k8s.io/kubernetes/staging/src/k8s.io/kube-scheduler v1.16.0-alpha.2.0.20190730231111-75d51896125b
	k8s.io/kubectl => k8s.io/kubernetes/staging/src/k8s.io/kubectl v1.16.0-alpha.2.0.20190730231111-75d51896125b
	k8s.io/kubelet => k8s.io/kubernetes/staging/src/k8s.io/kubelet v1.16.0-alpha.2.0.20190730231111-75d51896125b
	k8s.io/legacy-cloud-providers => k8s.io/kubernetes/staging/src/k8s.io/legacy-cloud-providers v1.16.0-alpha.2.0.20190730231111-75d51896125b
	k8s.io/metrics => k8s.io/kubernetes/staging/src/k8s.io/metrics v1.16.0-alpha.2.0.20190730231111-75d51896125b
	k8s.io/sample-apiserver => k8s.io/kubernetes/staging/src/k8s.io/sample-apiserver v1.16.0-alpha.2.0.20190730231111-75d51896125b
)
