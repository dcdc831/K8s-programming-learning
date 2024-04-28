package main

import (
	"context"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	// RESTClient
	// 	// config
	// 	config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	// 	// println(clientcmd.RecommendedHomeFile)
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// 	config.GroupVersion = &v1.SchemeGroupVersion
	// 	config.NegotiatedSerializer = scheme.Codecs
	// 	config.APIPath = "/api"
	// 	// client
	// 	restClient, err := rest.RESTClientFor(config)
	// 	if err != nil {
	// 		panic(err)
	// 	}

	// pod := v1.Pod{}
	// err = restClient.Get().Namespace("kube-system").Resource("pods").Name("kube-apiserver-lyl-virtual-machine").Do(context.TODO()).Into(&pod)
	//
	//	if err != nil {
	//		panic(err)
	//	} else {
	//
	//		println(pod.Name)
	//	}

	// clientSet
	config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	if err != nil {
		panic(err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}
	coreV1 := clientset.CoreV1()
	pod, err := coreV1.Pods("kube-system").Get(context.TODO(), "kube-apiserver-lyl-virtual-machine", v1.GetOptions{})
	if err != nil {
		println(err)
	} else {
		println(pod.Name)
	}
}
