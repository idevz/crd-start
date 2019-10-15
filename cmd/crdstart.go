package cmd

import (
	"time"

	"net/http"
	_ "net/http/pprof"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"

	kubeinformers "k8s.io/client-go/informers"

	crdstartclientset "github.com/idevz/crd-start/pkg/client/clientset/versioned"
	crdstartinformers "github.com/idevz/crd-start/pkg/client/informers/externalversions"
)

const DefauleResyncDuration = time.Second * 30

var (
	masterURL     string
	kubeconfig    string
	klogV         string
	deploymentTpl string
)

func Run() {
	klogInit()

	stopCh := SetupSignalHandler()

	cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		klog.Fatalf("error build kubeConfig:%s", err.Error())
	}

	kubeClientSet, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("error build kubeClientSet: %s", err.Error())
	}

	crdStartClientSet, err := crdstartclientset.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("error build crdStartClientSet: %s", err.Error())
	}

	go func() {
		http.ListenAndServe(":6060", nil)
	}()

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(
		kubeClientSet,
		DefauleResyncDuration)
	crdStartInformerFactory := crdstartinformers.NewSharedInformerFactory(
		crdStartClientSet,
		DefauleResyncDuration)

	crdController := NewCrdController(
		kubeClientSet, crdStartClientSet,
		kubeInformerFactory.Apps().V1().Deployments(),
		kubeInformerFactory.Apps().V1().ReplicaSets(),
		kubeInformerFactory.Core().V1().Pods(),
		crdStartInformerFactory.Crdstart().V1alpha1().Dcreaters())

	kubeInformerFactory.Start(stopCh)
	crdStartInformerFactory.Start(stopCh)

	if err = crdController.Run(2, stopCh); err != nil {
		klog.Fatalf("error running controller: %s", err.Error())
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&kubeconfig, "kubeConfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	rootCmd.PersistentFlags().StringVar(&masterURL, "masterURL", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	rootCmd.PersistentFlags().StringVar(&klogV, "v", "1", "klog log level setting.")
	rootCmd.PersistentFlags().StringVar(&deploymentTpl, "deploymentTpl", "./build/artifacts/deployment-tpl.yaml", "deployment template")
}
