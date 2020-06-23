package myfilter

import (
	corev1 "k8s.io/api/core/v1"
	framework "k8s.io/kubernetes/pkg/scheduler/framework/v1alpha1"


)


func filterHandler( pod *corev1.Pod, node *corev1.Node, f framework.FrameworkHandle ) error {


	return nil

}


