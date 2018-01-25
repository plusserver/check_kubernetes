package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/Nexinto/check_kubernetes/nrpe"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func main() {
	var kubeconfig, objectType, name, namespace string

	flag.StringVar(&kubeconfig, "kubeconfig", "", "kubeconfig location")
	flag.StringVar(&objectType, "type", "", "type of object to check")
	flag.StringVar(&name, "name", "", "name of object to check")
	flag.StringVar(&namespace, "namespace", "default", "namespace of object")

	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)

	if err != nil {
		fmt.Printf("Cannot configure kubernetes client: %s\n", err.Error())
		os.Exit(nrpe.UNKNOWN)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Printf("Cannot create kubernetes client: %s\n", err.Error())
		os.Exit(nrpe.UNKNOWN)
	}

	var level nrpe.Result
	var message string

	switch objectType {
	case "pod":
		level, message = checkPod(namespace, name, clientset)
	case "replicaset":
		level, message = checkReplicaSet(namespace, name, clientset)
	case "deployment":
		level, message = checkDeployment(namespace, name, clientset)
	case "daemonset":
		level, message = checkDaemonSet(namespace, name, clientset)
	case "statefulset":
		level, message = checkStatefulSet(namespace, name, clientset)
	case "node":
		level, message = checkNode(name, clientset)
	case "componentstatus":
		level, message = checkComponentStatus(name, clientset)
	case "service":
		level, message = checkService(namespace, name, clientset)
	default:
		level, message = nrpe.UNKNOWN, fmt.Sprintf("unsupported object type %s", objectType)
	}

	m := map[nrpe.Result]string{
		nrpe.OK:       "OK",
		nrpe.WARNING:  "WARNING",
		nrpe.CRITICAL: "CRITICAL",
		nrpe.UNKNOWN:  "UNKNOWN",
	}

	fmt.Printf("%s: %s\n", m[level], message)
	os.Exit(int(level))
}

func handleLookupError(err error) (nrpe.Result, string) {

	if errors.IsNotFound(err) {
		return nrpe.UNKNOWN, "object not found"
	} else if statusError, isStatus := err.(*errors.StatusError); isStatus {
		return nrpe.UNKNOWN, fmt.Sprintf("Error getting object: %v", statusError.ErrStatus.Message)
	} else if err != nil {
		return nrpe.UNKNOWN, fmt.Sprintf("Error getting object: %v", err.Error())
	}

	return nrpe.OK, ""

}

func checkPod(namespace, name string, kube kubernetes.Interface) (nrpe.Result, string) {

	pod, err := kube.CoreV1().Pods(namespace).Get(name, metav1.GetOptions{})

	if result, message := handleLookupError(err); result != nrpe.OK {
		return result, message
	}

	ready, running, max := 0, 0, len(pod.Status.ContainerStatuses)

	for _, c := range pod.Status.ContainerStatuses {
		if c.Ready {
			ready = ready + 1
		}
		if c.State.Running != nil {
			running = running + 1
		}
	}

	containerStatus := fmt.Sprintf("%d/%d containers running, %d/%d containers ready", running, max, ready, max)

	switch pod.Status.Phase {

	// PodRunning means the pod has been bound to a node and all of the containers have been started.
	// At least one container is still running or is in the process of being restarted.
	case corev1.PodRunning:
		if ready == max && running == max {
			return nrpe.OK, fmt.Sprintf("pod running; %s", containerStatus)
		}

		return nrpe.CRITICAL, fmt.Sprintf("pod running but containers are failing; %s", containerStatus)

	// PodPending means the pod has been accepted by the system, but one or more of the containers
	// has not been started. This includes time before being bound to a node, as well as time spent
	// pulling images onto the host.
	case corev1.PodPending:
		return nrpe.WARNING, "pod pending"

	// PodSucceeded means that all containers in the pod have voluntarily terminated
	// with a container exit code of 0, and the system is not going to restart any of these containers.
	case corev1.PodSucceeded:
		return nrpe.OK, fmt.Sprintf("pod succeeded (terminated without errors); %s", containerStatus)

	// PodFailed means that all containers in the pod have terminated, and at least one container has
	// terminated in a failure (exited with a non-zero exit code or was stopped by the system).
	case corev1.PodFailed:
		return nrpe.CRITICAL, fmt.Sprintf("pod failed (at least one container terminated with errors); %s", containerStatus)

	// PodUnknown means that for some reason the state of the pod could not be obtained, typically due
	// to an error in communicating with the host of the pod.
	case corev1.PodUnknown:
		return nrpe.UNKNOWN, "pod state could not be determined"

	}

	return nrpe.OK, containerStatus
}

func checkReplicaSet(namespace, name string, kube kubernetes.Interface) (nrpe.Result, string) {

	rs, err := kube.AppsV1beta2().ReplicaSets(namespace).Get(name, metav1.GetOptions{})

	if result, message := handleLookupError(err); result != nrpe.OK {
		return result, message
	}

	avail := rs.Status.AvailableReplicas
	ready := rs.Status.ReadyReplicas
	max := rs.Status.Replicas

	rsStatus := fmt.Sprintf("%d/%d pods available, %d/%d pods ready", avail, max, ready, max)

	if avail < max || ready < max {
		if avail != 0 && ready != 0 {
			return nrpe.WARNING, fmt.Sprintf("replicaset degraded: %s", rsStatus)
		}
		return nrpe.CRITICAL, fmt.Sprintf("replicaset unavailable: %s", rsStatus)
	}

	return nrpe.OK, rsStatus
}

func checkDeployment(namespace, name string, kube kubernetes.Interface) (nrpe.Result, string) {

	dep, err := kube.AppsV1beta2().Deployments(namespace).Get(name, metav1.GetOptions{})

	if result, message := handleLookupError(err); result != nrpe.OK {
		return result, message
	}

	avail := dep.Status.AvailableReplicas
	updated := dep.Status.UpdatedReplicas
	max := dep.Status.Replicas

	depStatus := fmt.Sprintf("%d/%d replicas available, %d/%d replicas up to spec and running", avail, max, updated, max)

	if avail < max || updated < max {
		if avail != 0 && updated != 0 {
			return nrpe.WARNING, fmt.Sprintf("deployment degraded: %s", depStatus)
		}
		return nrpe.CRITICAL, fmt.Sprintf("deployment unavailable: %s", depStatus)
	}

	return nrpe.OK, depStatus
}

func checkDaemonSet(namespace, name string, kube kubernetes.Interface) (nrpe.Result, string) {

	ds, err := kube.AppsV1beta2().DaemonSets(namespace).Get(name, metav1.GetOptions{})

	if result, message := handleLookupError(err); result != nrpe.OK {
		return result, message
	}

	current := ds.Status.CurrentNumberScheduled
	ready := ds.Status.NumberReady
	missched := ds.Status.NumberMisscheduled
	max := ds.Status.DesiredNumberScheduled

	var misStatus string
	if missched == 0 {
		misStatus = ""
	} else {
		misStatus = fmt.Sprintf(" (%d misscheduled)", missched)
	}

	dsStatus := fmt.Sprintf("%d/%d nodes with pods running, %d/%d pods available%s", current, max, ready, max, misStatus)

	if missched > 0 || current < max || ready < max {
		return nrpe.CRITICAL, fmt.Sprintf("daemonset incomplete: %s", dsStatus)
	}

	return nrpe.OK, dsStatus
}

func checkStatefulSet(namespace, name string, kube kubernetes.Interface) (nrpe.Result, string) {

	ss, err := kube.AppsV1beta2().StatefulSets(namespace).Get(name, metav1.GetOptions{})

	if result, message := handleLookupError(err); result != nrpe.OK {
		return result, message
	}

	current := ss.Status.CurrentReplicas
	max := ss.Status.Replicas
	ready := ss.Status.ReadyReplicas

	ssStatus := fmt.Sprintf("%d/%d replicas created, %d/%d ready", current, max, ready, max)

	if current < max || ready < max {
		return nrpe.CRITICAL, fmt.Sprintf("statefulset incomplete: %s", ssStatus)
	}

	return nrpe.OK, ssStatus
}

func checkNode(name string, kube kubernetes.Interface) (nrpe.Result, string) {
	node, err := kube.CoreV1().Nodes().Get(name, metav1.GetOptions{})

	if result, message := handleLookupError(err); result != nrpe.OK {
		return result, message
	}

	nosched := node.Spec.Unschedulable
	ready := false

	if len(node.Status.Conditions) < 1 {
		return nrpe.UNKNOWN, "node in unknown state"
	}

	for _, cond := range node.Status.Conditions {
		switch cond.Type {
		case corev1.NodeReady:
			if cond.Status == corev1.ConditionTrue {
				ready = true
			}
		default: // invalid value for Type
			return nrpe.UNKNOWN, "node in unknown state (invalid condition)"
		}
	}

	if ready && nosched {
		return nrpe.WARNING, "node ready but no scheduling allowed"
	}

	if ready {
		return nrpe.OK, "node ready"
	}

	return nrpe.CRITICAL, "node not ready"
}

func checkComponentStatus(name string, kube kubernetes.Interface) (nrpe.Result, string) {
	component, err := kube.CoreV1().ComponentStatuses().Get(name, metav1.GetOptions{})

	if result, message := handleLookupError(err); result != nrpe.OK {
		return result, message
	}

	healthy := false
	var message string

	for _, cond := range component.Conditions {
		message = cond.Message
		if cond.Type == corev1.ComponentHealthy {
			switch cond.Status {
			case corev1.ConditionTrue:
				healthy = true
			case corev1.ConditionFalse:
				healthy = false
				break
			case corev1.ConditionUnknown:
				return nrpe.UNKNOWN, fmt.Sprintf("component state unknown: %s", message)
			}
		}
	}

	if healthy {
		return nrpe.OK, fmt.Sprintf("component healthy: %s", message)
	}

	return nrpe.CRITICAL, fmt.Sprintf("component unhealthy: %s", message)
}

func checkService(namespace, name string, kube kubernetes.Interface) (nrpe.Result, string) {
	service, err := kube.CoreV1().Services(namespace).Get(name, metav1.GetOptions{})

	if result, message := handleLookupError(err); result != nrpe.OK {
		return result, message
	}

	switch service.Spec.Type {
	case corev1.ServiceTypeLoadBalancer:
		for _, i := range service.Status.LoadBalancer.Ingress {
			if len(i.IP) > 0 {
				return nrpe.OK, fmt.Sprintf("configured with IP %s", i.IP)
			}
			if len(i.Hostname) > 0 {
				return nrpe.OK, fmt.Sprintf("configured with hostname %s", i.Hostname)
			}
		}
		return nrpe.WARNING, "pending"
	}

	return nrpe.OK, "service configured"
}
