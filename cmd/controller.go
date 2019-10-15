package cmd

import (
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/labels"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	appsinformers "k8s.io/client-go/informers/apps/v1"
	corev1informers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	appslisters "k8s.io/client-go/listers/apps/v1"
	corev1listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"

	"github.com/idevz/crd-start/pkg/apis/crdstart/v1alpha1"
	crdstartclientset "github.com/idevz/crd-start/pkg/client/clientset/versioned"
	crdstartscheme "github.com/idevz/crd-start/pkg/client/clientset/versioned/scheme"
	crdstartinformers "github.com/idevz/crd-start/pkg/client/informers/externalversions/crdstart/v1alpha1"
	crdstartlisters "github.com/idevz/crd-start/pkg/client/listers/crdstart/v1alpha1"
)

const (
	controllerAgentName = "crdstart-controller"
	crQueueName         = "dcreater-queue"
)

type CrdController struct {
	kubeClientSet     kubernetes.Interface
	crdStartCleintSet crdstartclientset.Interface

	deploymentsLister appslisters.DeploymentLister
	deploymentsSynced cache.InformerSynced

	replicasetsLister appslisters.ReplicaSetLister
	replicasetsSynced cache.InformerSynced

	podsLister corev1listers.PodLister
	podsSynced cache.InformerSynced

	crdStartLister crdstartlisters.DcreaterLister
	crdStartSynced cache.InformerSynced

	workQueue workqueue.RateLimitingInterface

	recorder record.EventRecorder
}

func NewCrdController(
	kubeClientSet kubernetes.Interface,
	crdStartClientSet crdstartclientset.Interface,
	deploymentInformer appsinformers.DeploymentInformer,
	replicasetInformer appsinformers.ReplicaSetInformer,
	podInformer corev1informers.PodInformer,
	crdStartInformer crdstartinformers.DcreaterInformer,
) *CrdController {
	utilruntime.Must(crdstartscheme.AddToScheme(scheme.Scheme))
	klog.Infoln("create event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{
		Interface: kubeClientSet.CoreV1().Events(""),
	})
	recorder := eventBroadcaster.NewRecorder(
		scheme.Scheme,
		corev1.EventSource{
			Component: controllerAgentName,
		})

	crdController := &CrdController{
		kubeClientSet:     kubeClientSet,
		crdStartCleintSet: crdStartClientSet,
		deploymentsLister: deploymentInformer.Lister(),
		deploymentsSynced: deploymentInformer.Informer().HasSynced,
		replicasetsLister: replicasetInformer.Lister(),
		replicasetsSynced: replicasetInformer.Informer().HasSynced,
		podsLister:        podInformer.Lister(),
		podsSynced:        podInformer.Informer().HasSynced,
		crdStartLister:    crdStartInformer.Lister(),
		crdStartSynced:    crdStartInformer.Informer().HasSynced,
		workQueue: workqueue.NewNamedRateLimitingQueue(
			workqueue.DefaultControllerRateLimiter(),
			crQueueName),
		recorder: recorder,
	}
	klog.Infof("setting up handlers")
	crdStartInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: crdController.enQueueDcreater,
			UpdateFunc: func(oldObj, newObj interface{}) {
				old := oldObj.(*v1alpha1.Dcreater)
				new := newObj.(*v1alpha1.Dcreater)
				if new.ResourceVersion == old.ResourceVersion {
					return
				}
				crdController.enQueueDcreater(newObj)
			},
			DeleteFunc: crdController.cleanAllControl,
		},
	)
	return crdController
}

func (c *CrdController) enQueueDcreater(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workQueue.Add(key)
}

func (c *CrdController) Run(threadiness int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.workQueue.ShutDown()

	klog.Infof("start crdstart controller")

	if ok := cache.WaitForCacheSync(stopCh, c.deploymentsSynced, c.crdStartSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-stopCh
	klog.Info("shutting down workers.")
	return nil
}

func (c *CrdController) runWorker() {
	for c.processNextItem() {
	}
}

func (c *CrdController) processNextItem() bool {
	obj, shutdown := c.workQueue.Get()
	if shutdown {
		return false
	}
	err := func(obj interface{}) error {
		defer c.workQueue.Done(obj)
		key, ok := obj.(string)
		if !ok {
			c.workQueue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("exected string in workqueue but got: %#v", obj))
			return nil
		}

		if err := c.syncHandler(key); err != nil {
			c.workQueue.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s', err: %s, requeuing", key, err.Error())
		}

		c.workQueue.Forget(obj)
		klog.Infof("successfully synced '%s'", key)
		return nil
	}(obj)
	if err != nil {
		utilruntime.HandleError(err)
		return true
	}
	return true
}

func (c *CrdController) cleanAllControl(obj interface{}) {
	object, ok := obj.(metav1.Object)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			utilruntime.HandleError(fmt.Errorf(
				"error decoding object, invalid type"))
			return
		}
		object, ok = tombstone.Obj.(metav1.Object)
		if !ok {
			utilruntime.HandleError(fmt.Errorf(
				"error decoding object tombstone, invalid type"))
			return
		}
		klog.Infof("recoverd deleted object '%s' from tomstone", object.GetName())
	}
	klog.Infof("processing object:'%s'", object.GetName())

	deployments, err := c.deploymentsLister.
		Deployments(object.GetNamespace()).
		List(labels.Everything())
	if err != nil {
		utilruntime.HandleError(
			fmt.Errorf("error for list all deployment in namespace:%s",
				object.GetNamespace()))
		return
	}

	for _, deployment := range deployments {
		if ownerRef := metav1.GetControllerOf(deployment); ownerRef != nil {
			if ownerRef.UID == object.GetUID() {
				logSaveAndClean(c, deployment)
			}
		}
	}
	return
}

func (c *CrdController) syncHandler(key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}
	dCreater, err := c.crdStartLister.Dcreaters(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			utilruntime.HandleError(
				fmt.Errorf("dcreater '%s' in work queue no longer exists", key))
			return nil
		}
		return err
	}

	deploymentName := dCreater.Spec.DeploymentName
	if deploymentName == "" {
		utilruntime.HandleError(
			fmt.Errorf("%s: deployment name must be specified", key))
		return nil
	}
	deployment, err := c.deploymentsLister.
		Deployments(dCreater.Namespace).Get(deploymentName)
	if errors.IsNotFound(err) {
		newDeployment, err := newDeployment(dCreater)
		if err != nil {
			utilruntime.HandleError(
				fmt.Errorf("new deployment error, err:%s", err.Error()))
			return err
		}
		deployment, err = c.kubeClientSet.AppsV1().
			Deployments(dCreater.Namespace).Create(newDeployment)
		if err != nil {
			utilruntime.HandleError(
				fmt.Errorf("create deployment error, error: %s", err.Error()))
			return err
		}
	}

	if err != nil {
		return err
	}

	if !metav1.IsControlledBy(deployment, dCreater) {
		msg := fmt.Sprintf(
			"Resource %q already exists and is not managed by dCreate",
			deployment.Name)
		c.recorder.Event(dCreater, corev1.EventTypeWarning, "ErrResourceExists", msg)
		return fmt.Errorf(msg)
	}

	//深入理解 可调谐的字段 不是所有的都可以调谐
	if dCreater.Spec.Replicas != nil &&
		*dCreater.Spec.Replicas != *deployment.Spec.Replicas {
		klog.Infof("crdStart %s replicas:%d, deployment replicas:%d",
			name, *dCreater.Spec.Replicas, *deployment.Spec.Replicas)
		deploymentCopy := deployment.DeepCopy()
		deploymentCopy.Spec.Replicas = dCreater.Spec.Replicas
		_, err = c.kubeClientSet.AppsV1().
			Deployments(dCreater.Namespace).Update(deploymentCopy)

		if err != nil {
			utilruntime.HandleError(
				fmt.Errorf("update deployment error, err: %s", err.Error()))
			return err
		}
	}

	c.recorder.Event(dCreater,
		corev1.EventTypeNormal, "Synced",
		"dcreater synced successfully")
	return nil
}
