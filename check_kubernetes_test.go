package main

import (
	"fmt"
	"testing"
	"time"

	"github.com/Nexinto/check_kubernetes/nrpe"
	"github.com/stretchr/testify/assert"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"

	appsv1beta2 "k8s.io/api/apps/v1beta2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func makeFakeclient() kubernetes.Interface {
	return fake.NewSimpleClientset()
}

func TestPod(t *testing.T) {
	a := assert.New(t)

	tests := []struct {
		name    string
		pod     corev1.Pod
		result  nrpe.Result
		message string
	}{
		{
			name: "running",
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mypod",
					Namespace: "default"},
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
					ContainerStatuses: []corev1.ContainerStatus{{
						Ready: true,
						State: corev1.ContainerState{
							Running: &corev1.ContainerStateRunning{
								StartedAt: metav1.Time{Time: time.Now()},
							},
						},
					},
					},
				},
			},
			result:  nrpe.OK,
			message: "running",
		},
		{
			name: "failed",
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mypod",
					Namespace: "default"},
				Status: corev1.PodStatus{
					Phase: corev1.PodFailed,
					ContainerStatuses: []corev1.ContainerStatus{{
						Ready: false,
						State: corev1.ContainerState{
							Terminated: &corev1.ContainerStateTerminated{
								StartedAt: metav1.Time{Time: time.Now()},
							},
						},
					},
					},
				},
			},
			result:  nrpe.CRITICAL,
			message: "fail",
		},
		{
			name: "failed container",
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mypod",
					Namespace: "default"},
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
					ContainerStatuses: []corev1.ContainerStatus{{
						Ready: true,
						State: corev1.ContainerState{
							Running: &corev1.ContainerStateRunning{
								StartedAt: metav1.Time{Time: time.Now()},
							},
						},
					}, {
						Ready: false,
						State: corev1.ContainerState{
							Terminated: &corev1.ContainerStateTerminated{
								StartedAt: metav1.Time{Time: time.Now()},
							},
						},
					},
					},
				},
			},
			result:  nrpe.CRITICAL,
			message: "fail",
		},
	}

	for _, test := range tests {
		c := makeFakeclient()
		c.CoreV1().Pods("default").Create(&test.pod)
		result, message := checkPod("default", test.pod.ObjectMeta.Name, c)
		a.Equal(test.result, result, fmt.Sprintf("testcase %s for pod %+v failed", test.name, test.pod))
		a.NotEmpty(message, fmt.Sprintf("testcase %s for pod %+v failed", test.name, test.pod))
		a.Regexp(test.message, message, fmt.Sprintf("testcase %s for pod %+v failed", test.name, test.pod))
	}
}

func TestReplicaSet(t *testing.T) {
	a := assert.New(t)

	tests := []struct {
		name    string
		set     appsv1beta2.ReplicaSet
		result  nrpe.Result
		message string
	}{
		{
			name: "running",
			set: appsv1beta2.ReplicaSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "myreplicaset",
					Namespace: "default"},
				Status: appsv1beta2.ReplicaSetStatus{
					Replicas:          5,
					AvailableReplicas: 5,
					ReadyReplicas:     5,
				},
			},
			result:  nrpe.OK,
			message: "available",
		},
		{
			name: "degraded",
			set: appsv1beta2.ReplicaSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "myreplicaset",
					Namespace: "default"},
				Status: appsv1beta2.ReplicaSetStatus{
					Replicas:          5,
					AvailableReplicas: 3,
					ReadyReplicas:     3,
				},
			},
			result:  nrpe.WARNING,
			message: "degraded",
		},
		{
			name: "broken",
			set: appsv1beta2.ReplicaSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "myreplicaset",
					Namespace: "default"},
				Status: appsv1beta2.ReplicaSetStatus{
					Replicas:          5,
					AvailableReplicas: 0,
					ReadyReplicas:     0,
				},
			},
			result:  nrpe.CRITICAL,
			message: "unavailable",
		},
	}

	for _, test := range tests {
		c := makeFakeclient()
		c.AppsV1beta2().ReplicaSets("default").Create(&test.set)
		result, message := checkReplicaSet("default", test.set.ObjectMeta.Name, c)
		a.Equal(test.result, result, fmt.Sprintf("testcase %s for replicaset %+v failed", test.name, test.set))
		a.NotEmpty(message, fmt.Sprintf("testcase %s for replicaset %+v failed", test.name, test.set))
		a.Regexp(test.message, message, fmt.Sprintf("testcase %s for replicaset %+v failed", test.name, test.set))
	}
}

func TestDeployment(t *testing.T) {
	a := assert.New(t)

	tests := []struct {
		name    string
		deploy  appsv1beta2.Deployment
		result  nrpe.Result
		message string
	}{
		{
			name: "running",
			deploy: appsv1beta2.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mydeployment",
					Namespace: "default"},
				Status: appsv1beta2.DeploymentStatus{
					Replicas:          5,
					AvailableReplicas: 5,
					UpdatedReplicas:   5,
				},
			},
			result:  nrpe.OK,
			message: "running",
		},
		{
			name: "degraded",
			deploy: appsv1beta2.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mydeployment",
					Namespace: "default"},
				Status: appsv1beta2.DeploymentStatus{
					Replicas:          5,
					AvailableReplicas: 3,
					UpdatedReplicas:   3,
				},
			},
			result:  nrpe.WARNING,
			message: "degraded",
		},
		{
			name: "broken",
			deploy: appsv1beta2.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mydeployment",
					Namespace: "default"},
				Status: appsv1beta2.DeploymentStatus{
					Replicas:          5,
					AvailableReplicas: 0,
					UpdatedReplicas:   0,
				},
			},
			result:  nrpe.CRITICAL,
			message: "unavailable",
		},
	}

	for _, test := range tests {
		c := makeFakeclient()
		c.AppsV1beta2().Deployments("default").Create(&test.deploy)
		result, message := checkDeployment("default", test.deploy.ObjectMeta.Name, c)
		a.Equal(test.result, result, fmt.Sprintf("testcase %s for deployment %+v failed", test.name, test.deploy))
		a.NotEmpty(message, fmt.Sprintf("testcase %s for deployment %+v failed", test.name, test.deploy))
		a.Regexp(test.message, message, fmt.Sprintf("testcase %s for deployment %+v failed", test.name, test.deploy))
	}
}

func TestDaemonSet(t *testing.T) {
	a := assert.New(t)

	tests := []struct {
		name    string
		set     appsv1beta2.DaemonSet
		result  nrpe.Result
		message string
	}{
		{
			name: "running",
			set: appsv1beta2.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "myds",
					Namespace: "default"},
				Status: appsv1beta2.DaemonSetStatus{
					DesiredNumberScheduled: 5,
					NumberReady:            5,
					NumberMisscheduled:     0,
					CurrentNumberScheduled: 5,
				},
			},
			result:  nrpe.OK,
			message: "running",
		},
		{
			name: "incomplete",
			set: appsv1beta2.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "myds",
					Namespace: "default"},
				Status: appsv1beta2.DaemonSetStatus{
					DesiredNumberScheduled: 5,
					NumberReady:            3,
					NumberMisscheduled:     0,
					CurrentNumberScheduled: 5,
				},
			},
			result:  nrpe.CRITICAL,
			message: "incomplete",
		},
		{
			name: "misscheduled",
			set: appsv1beta2.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "myds",
					Namespace: "default"},
				Status: appsv1beta2.DaemonSetStatus{
					DesiredNumberScheduled: 5,
					NumberReady:            3,
					NumberMisscheduled:     2,
					CurrentNumberScheduled: 5,
				},
			},
			result:  nrpe.CRITICAL,
			message: "misscheduled",
		},
	}

	for _, test := range tests {
		c := makeFakeclient()
		c.AppsV1beta2().DaemonSets("default").Create(&test.set)
		result, message := checkDaemonSet("default", test.set.ObjectMeta.Name, c)
		a.Equal(test.result, result, fmt.Sprintf("testcase %s for daemonset %+v failed", test.name, test.set))
		a.NotEmpty(message, fmt.Sprintf("testcase %s for daemonset %+v failed", test.name, test.set))
		a.Regexp(test.message, message, fmt.Sprintf("testcase %s for daemonset %+v failed", test.name, test.set))
	}
}

func TestStatefulSet(t *testing.T) {
	a := assert.New(t)

	tests := []struct {
		name    string
		set     appsv1beta2.StatefulSet
		result  nrpe.Result
		message string
	}{
		{
			name: "running",
			set: appsv1beta2.StatefulSet{ObjectMeta: metav1.ObjectMeta{
				Name:      "mystateful",
				Namespace: "default"},
				Status: appsv1beta2.StatefulSetStatus{
					Replicas:        3,
					ReadyReplicas:   3,
					CurrentReplicas: 3,
				},
			},
			result:  nrpe.OK,
			message: "ready",
		},
		{
			name: "incomplete",
			set: appsv1beta2.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mystateful",
					Namespace: "default"},
				Status: appsv1beta2.StatefulSetStatus{
					Replicas:        3,
					ReadyReplicas:   2,
					CurrentReplicas: 3,
				},
			},
			result:  nrpe.CRITICAL,
			message: "incomplete",
		},
	}

	for _, test := range tests {
		c := makeFakeclient()
		c.AppsV1beta2().StatefulSets("default").Create(&test.set)
		result, message := checkStatefulSet("default", test.set.ObjectMeta.Name, c)
		a.Equal(test.result, result, fmt.Sprintf("testcase %s for statefulset %+v failed", test.name, test.set))
		a.NotEmpty(message, fmt.Sprintf("testcase %s for statefulset %+v failed", test.name, test.set))
		a.Regexp(test.message, message, fmt.Sprintf("testcase %s for statefulset %+v failed", test.name, test.set))
	}
}

func TestNode(t *testing.T) {
	a := assert.New(t)

	tests := []struct {
		name    string
		node    corev1.Node
		result  nrpe.Result
		message string
	}{
		// These are the testcases from kubernetes/pkg/printers/internalversion/printers_test.go
		{
			node: corev1.Node{
				ObjectMeta: metav1.ObjectMeta{Name: "foo1"},
				Status:     corev1.NodeStatus{Conditions: []corev1.NodeCondition{{Type: corev1.NodeReady, Status: corev1.ConditionTrue}}},
			},
			name:    "Ready",
			result:  nrpe.OK,
			message: "ready",
		},
		{
			node: corev1.Node{
				ObjectMeta: metav1.ObjectMeta{Name: "foo2"},
				Spec:       corev1.NodeSpec{Unschedulable: true},
				Status:     corev1.NodeStatus{Conditions: []corev1.NodeCondition{{Type: corev1.NodeReady, Status: corev1.ConditionTrue}}},
			},
			name:    "Ready,SchedulingDisabled",
			result:  nrpe.WARNING,
			message: "no.*sched",
		},
		{
			node: corev1.Node{
				ObjectMeta: metav1.ObjectMeta{Name: "foo3"},
				Status: corev1.NodeStatus{Conditions: []corev1.NodeCondition{
					{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
					{Type: corev1.NodeReady, Status: corev1.ConditionTrue}}},
			},
			name:    "Ready",
			result:  nrpe.OK,
			message: "ready",
		},
		{
			node: corev1.Node{
				ObjectMeta: metav1.ObjectMeta{Name: "foo4"},
				Status:     corev1.NodeStatus{Conditions: []corev1.NodeCondition{{Type: corev1.NodeReady, Status: corev1.ConditionFalse}}},
			},
			name:    "NotReady",
			result:  nrpe.CRITICAL,
			message: "not.*ready",
		},
		{
			node: corev1.Node{
				ObjectMeta: metav1.ObjectMeta{Name: "foo5"},
				Spec:       corev1.NodeSpec{Unschedulable: true},
				Status:     corev1.NodeStatus{Conditions: []corev1.NodeCondition{{Type: corev1.NodeReady, Status: corev1.ConditionFalse}}},
			},
			name:    "NotReady,SchedulingDisabled",
			result:  nrpe.CRITICAL,
			message: "not.*ready",
		},
		{
			node: corev1.Node{
				ObjectMeta: metav1.ObjectMeta{Name: "foo6"},
				Status:     corev1.NodeStatus{Conditions: []corev1.NodeCondition{{Type: "InvalidValue", Status: corev1.ConditionTrue}}},
			},
			name:    "Unknown",
			result:  nrpe.UNKNOWN,
			message: "unknown",
		},
		{
			node: corev1.Node{
				ObjectMeta: metav1.ObjectMeta{Name: "foo7"},
				Status:     corev1.NodeStatus{Conditions: []corev1.NodeCondition{{}}},
			},
			name:    "Unknown",
			result:  nrpe.UNKNOWN,
			message: "unknown",
		},
		{
			node: corev1.Node{
				ObjectMeta: metav1.ObjectMeta{Name: "foo8"},
				Spec:       corev1.NodeSpec{Unschedulable: true},
				Status:     corev1.NodeStatus{Conditions: []corev1.NodeCondition{{Type: "InvalidValue", Status: corev1.ConditionTrue}}},
			},
			name:    "Unknown,SchedulingDisabled",
			result:  nrpe.UNKNOWN,
			message: "unknown",
		},
		{
			node: corev1.Node{
				ObjectMeta: metav1.ObjectMeta{Name: "foo9"},
				Spec:       corev1.NodeSpec{Unschedulable: true},
				Status:     corev1.NodeStatus{Conditions: []corev1.NodeCondition{{}}},
			},
			name:    "Unknown,SchedulingDisabled",
			result:  nrpe.UNKNOWN,
			message: "unknown",
		},
	}

	for _, test := range tests {
		c := makeFakeclient()
		c.CoreV1().Nodes().Create(&test.node)
		result, message := checkNode(test.node.ObjectMeta.Name, c)
		a.Equal(test.result, result, fmt.Sprintf("testcase %s for node %+v failed", test.name, test.node))
		a.NotEmpty(message, fmt.Sprintf("testcase %s for node %+v failed", test.name, test.node))
		a.Regexp(test.message, message, fmt.Sprintf("testcase %s for node %+v failed", test.name, test.node))
	}
}

func TestComponentStatus(t *testing.T) {
	a := assert.New(t)

	type test struct {
		name    string
		comp    string
		result  nrpe.Result
		message string
	}

	samples := []struct {
		comps []corev1.ComponentStatus
		tests []test
	}{
		{
			comps: []corev1.ComponentStatus{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "scheduler"},
					Conditions: []corev1.ComponentCondition{{Type: corev1.ComponentHealthy, Status: corev1.ConditionTrue}},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Name: "controller-manager"},
					Conditions: []corev1.ComponentCondition{{Type: corev1.ComponentHealthy, Status: corev1.ConditionTrue}},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Name: "etcd-0"},
					Conditions: []corev1.ComponentCondition{{Type: corev1.ComponentHealthy, Status: corev1.ConditionTrue}},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Name: "etcd-1"},
					Conditions: []corev1.ComponentCondition{{Type: corev1.ComponentHealthy, Status: corev1.ConditionFalse}},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Name: "etcd-2"},
					Conditions: []corev1.ComponentCondition{{Type: corev1.ComponentHealthy, Status: corev1.ConditionUnknown}},
				},
			},
			tests: []test{
				{
					name:    "scheduler healthy",
					comp:    "scheduler",
					result:  nrpe.OK,
					message: "component healthy",
				},
				{
					name:    "controller-manager healthy",
					comp:    "controller-manager",
					result:  nrpe.OK,
					message: "component healthy",
				},
				{
					name:    "etcd-0 healthy",
					comp:    "etcd-0",
					result:  nrpe.OK,
					message: "component healthy",
				},
				{
					name:    "etcd-1 healthy",
					comp:    "etcd-1",
					result:  nrpe.CRITICAL,
					message: "component unhealthy",
				},
				{
					name:    "etcd-2 healthy",
					comp:    "etcd-2",
					result:  nrpe.UNKNOWN,
					message: "component state unknown",
				},
			},
		},
	}

	for _, sample := range samples {
		c := makeFakeclient()
		for _, cs := range sample.comps {
			c.CoreV1().ComponentStatuses().Create(&cs)
		}
		for _, test := range sample.tests {
			result, message := checkComponentStatus(test.comp, c)
			a.Equal(test.result, result, fmt.Sprintf("testcase %s for componentstatus %+v failed", test.name, sample.comps))
			a.NotEmpty(message, fmt.Sprintf("testcase %s for componentstatus %+v failed", test.name, sample.comps))
			a.Regexp(test.message, message, fmt.Sprintf("testcase %s for componentstatus %+v failed", test.name, sample.comps))
		}
	}
}

func TestService(t *testing.T) {
	a := assert.New(t)

	tests := []struct {
		name    string
		service corev1.Service
		result  nrpe.Result
		message string
	}{
		{
			name: "configured ClusterIP",
			service: corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "myservice",
					Namespace: "default"},
				Spec: corev1.ServiceSpec{
					ClusterIP: "10.100.10.10",
					Type:      corev1.ServiceTypeClusterIP,
				},
			},
			result:  nrpe.OK,
			message: "configured",
		},
		{
			name: "configured LoadBalancer",
			service: corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "myservice",
					Namespace: "default"},
				Spec: corev1.ServiceSpec{
					ClusterIP: "10.100.10.10",
					Type:      corev1.ServiceTypeLoadBalancer,
				},
				Status: corev1.ServiceStatus{
					LoadBalancer: corev1.LoadBalancerStatus{
						Ingress: []corev1.LoadBalancerIngress{{IP: "192.168.1.1"}},
					},
				},
			},
			result:  nrpe.OK,
			message: "configured with IP 192.168.1.1",
		},
		{
			name: "pending LoadBalancer",
			service: corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "myservice",
					Namespace: "default"},
				Spec: corev1.ServiceSpec{
					ClusterIP: "10.100.10.10",
					Type:      corev1.ServiceTypeLoadBalancer,
				},
				Status: corev1.ServiceStatus{
					LoadBalancer: corev1.LoadBalancerStatus{},
				},
			},
			result:  nrpe.WARNING,
			message: "pending",
		},
	}

	for _, test := range tests {
		c := makeFakeclient()
		c.CoreV1().Services("default").Create(&test.service)
		result, message := checkService("default", test.service.ObjectMeta.Name, c)
		a.Equal(test.result, result, fmt.Sprintf("testcase %s for service %+v failed", test.name, test.service))
		a.NotEmpty(message, fmt.Sprintf("testcase %s for service %+v failed", test.name, test.service))
		a.Regexp(test.message, message, fmt.Sprintf("testcase %s for service %+v failed", test.name, test.service))
	}
}
