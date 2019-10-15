package cmd

import (
	"bytes"
	"flag"
	"fmt"
	"html/template"
	"io"
	"os"
	"os/signal"

	"k8s.io/apimachinery/pkg/util/jsonmergepatch"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"

	"k8s.io/apimachinery/pkg/types"

	"k8s.io/apimachinery/pkg/labels"

	"k8s.io/apimachinery/pkg/util/json"

	"github.com/idevz/crd-start/pkg/apis/crdstart/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/klog"
	"k8s.io/kubernetes/pkg/apis/apps"
	"k8s.io/kubernetes/pkg/controller"
)

var (
	onlyOneSignalHandler        = make(chan struct{})
	shutdownSignals             = []os.Signal{os.Interrupt}
	PodLogSaveAndCleanFinalizer = "apps/podlogsave.crdstart.idevz.org"
)

func (c *CrdController) getReplicaSetsForDeployment(d *appsv1.Deployment) ([]*appsv1.ReplicaSet, error) {
	// List all ReplicaSets to find those we own but that no longer match our
	// selector. They will be orphaned by ClaimReplicaSets().
	rsList, err := c.replicasetsLister.
		ReplicaSets(d.Namespace).List(labels.Everything())
	//dc.rsLister.ReplicaSets(d.Namespace).List(labels.Everything())
	if err != nil {
		return nil, err
	}
	deploymentSelector, err := metav1.LabelSelectorAsSelector(d.Spec.Selector)
	if err != nil {
		return nil, fmt.Errorf("deployment %s/%s has invalid label selector: %v", d.Namespace, d.Name, err)
	}
	// If any adoptions are attempted, we should first recheck for deletion with
	// an uncached quorum read sometime after listing ReplicaSets (see #42639).
	canAdoptFunc := controller.RecheckDeletionTimestamp(func() (metav1.Object, error) {
		fresh, err := c.kubeClientSet.AppsV1().Deployments(d.Namespace).Get(d.Name, metav1.GetOptions{})
		//dc.client.AppsV1().Deployments(d.Namespace).Get(d.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		if fresh.UID != d.UID {
			return nil, fmt.Errorf("original Deployment %v/%v is gone: got uid %v, wanted %v", d.Namespace, d.Name, fresh.UID, d.UID)
		}
		return fresh, nil
	})
	rsControl := controller.RealRSControl{
		KubeClient: c.kubeClientSet,
		Recorder:   c.recorder,
	}
	var controllerKind = apps.SchemeGroupVersion.WithKind("Deployment")
	cm := controller.NewReplicaSetControllerRefManager(rsControl, d, deploymentSelector, controllerKind, canAdoptFunc)
	return cm.ClaimReplicaSets(rsList)
}

func (c *CrdController) getPodMapForDeployment(
	d *appsv1.Deployment,
) (map[types.UID][]*corev1.Pod, error) {
	rsList, err := c.getReplicaSetsForDeployment(d)
	if err != nil {
		return nil, err
	}

	selector, err := metav1.LabelSelectorAsSelector(d.Spec.Selector)
	if err != nil {
		return nil, err
	}
	pods, err := c.podsLister.Pods(d.Namespace).List(selector)
	if err != nil {
		return nil, err
	}
	podMap := make(map[types.UID][]*corev1.Pod, len(rsList))
	for _, rs := range rsList {
		podMap[rs.UID] = []*corev1.Pod{}
	}
	for _, pod := range pods {
		// Do not ignore inactive Pods because Recreate Deployments need to verify that no
		// Pods from older versions are running before spinning up new Pods.
		controllerRef := metav1.GetControllerOf(pod)
		if controllerRef == nil {
			continue
		}
		// Only append if we care about this UID.
		if _, ok := podMap[controllerRef.UID]; ok {
			podMap[controllerRef.UID] = append(podMap[controllerRef.UID], pod)
		}
	}
	return podMap, nil
}

func (c *CrdController) getRunningPods(
	obj *appsv1.Deployment,
) ([]*corev1.Pod, error) {
	podsMap, err := c.getPodMapForDeployment(obj)
	if err != nil {
		klog.Errorf("get pod map for deployment error, err: %s", err.Error())
		return nil, err
	}
	runingPods := []*corev1.Pod{}
	for _, pods := range podsMap {
		for _, pod := range pods {
			if pod.Status.Phase == corev1.PodRunning {
				runingPods = append(runingPods, pod)
			}
		}
	}
	return runingPods, nil
}

func (c *CrdController) saveLogs(obj *appsv1.Deployment) error {
	pods, err := c.getRunningPods(obj)
	if err != nil {
		return err
	}
	podLogPathRoot := "./pod-logs"
	for _, podInfo := range pods {
		err = func(podInfo *corev1.Pod) error {
			readCloser, err := c.kubeClientSet.CoreV1().
				Pods(obj.Namespace).
				GetLogs(podInfo.Name, &corev1.PodLogOptions{
					Timestamps: true,
				}).Stream()
			defer readCloser.Close()
			if err != nil {
				return err
			}
			if err := checkPath(podLogPathRoot); err != nil {
				return err
			}
			logFile, err := os.OpenFile(podLogPathRoot+"/"+podInfo.Name+".log",
				os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
				0777)
			if err != nil {
				return err
			}

			if _, err := io.Copy(logFile, readCloser); err != nil {
				return nil
			}
			return nil
		}(podInfo)
		if err != nil {
			klog.Errorf("save pod log error, err: %s", err.Error())
		}
	}
	return nil
}

func logSaveAndClean(c *CrdController, deployment *appsv1.Deployment) {
	if err := c.saveLogs(deployment); err != nil {
		utilruntime.HandleError(
			fmt.Errorf("error for save pod logs for deployment: %s, error:%s",
				deployment.Name, err.Error()))
	}
	curJson, err := json.Marshal(deployment)
	if err != nil {
		utilruntime.HandleError(
			fmt.Errorf("marshal current deployment err, error: %s", err.Error()))
	}
	if deployment.Finalizers[0] != PodLogSaveAndCleanFinalizer {
		return
	}
	newDeployment := deployment.DeepCopy()
	newDeployment.Finalizers = nil

	newJson, err := json.Marshal(newDeployment)
	if err != nil {
		utilruntime.HandleError(
			fmt.Errorf("marshal new deployment err, error: %s", err.Error()))
	}
	patch, err := jsonmergepatch.CreateThreeWayJSONMergePatch(curJson, newJson, curJson)
	if err != nil {
		utilruntime.HandleError(
			fmt.Errorf("deployment build patch err, error: %s", err.Error()))
	}

	if len(patch) > 0 || string(patch) != "{}" {
		klog.Infoln("start patching ", deployment.Name)
		_, err := c.kubeClientSet.AppsV1().
			Deployments(deployment.Namespace).
			Patch(deployment.Name, types.MergePatchType, patch)
		if err != nil {
			utilruntime.HandleError(
				fmt.Errorf("patch deployment err, error: %s", err.Error()))
		}
	}
	return
}

func checkPath(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.MkdirAll(path, 0777); err != nil {
			return err
		}
	}
	return nil
}

func SetupSignalHandler() (stopCh <-chan struct{}) {
	close(onlyOneSignalHandler)

	stop := make(chan struct{})
	c := make(chan os.Signal, 2)
	signal.Notify(c, shutdownSignals...)
	go func() {
		<-c
		close(stop)
		<-c
		os.Exit(1)
	}()
	return stop
}

func klogInit() {
	klog.InitFlags(nil)
	flag.Set("v", klogV)
}

type deployForCrd struct {
	DeploymentName string
	Replicas       int32
}

func parseYaml(dataObj interface{}, yamlTpl string) (*[]byte, error) {
	tpl, err := template.ParseFiles(yamlTpl)
	if err != nil {
		return nil, err
	}
	var yBuf bytes.Buffer
	switch yData := dataObj.(type) {
	default:
		tpl.Execute(&yBuf, yData)
	}

	jsonByte, err := yaml.ToJSON(yBuf.Bytes())
	if err != nil {
		klog.Errorf("yamlBuf to JonsByte err: %s error, err: %s", yamlTpl, err.Error())
		return nil, err
	}
	return &jsonByte, err
}

func newDeployment(dCreater *v1alpha1.Dcreater) (*appsv1.Deployment, error) {
	deployBytes, err := parseYaml(&deployForCrd{
		DeploymentName: dCreater.Spec.DeploymentName,
		Replicas:       *dCreater.Spec.Replicas},
		deploymentTpl)
	if err != nil {
		return nil, err
	}
	deploymentObj := &appsv1.Deployment{}
	err = json.Unmarshal(*deployBytes, &deploymentObj)
	if err != nil {
		return nil, err
	}
	deploymentObj.SetOwnerReferences([]metav1.OwnerReference{
		*metav1.NewControllerRef(dCreater, v1alpha1.SchemeGroupVersion.WithKind("Dcreater")),
	})
	deploymentObj.ObjectMeta.Finalizers = append(deploymentObj.ObjectMeta.Finalizers, PodLogSaveAndCleanFinalizer)
	return deploymentObj, nil
}
