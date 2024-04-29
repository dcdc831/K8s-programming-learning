package main

import (
	"log"

	"lyl/controller-demo/pkg"

	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	if err != nil {
		inClusterConfig, err := rest.InClusterConfig()
		if err != nil {
			log.Fatalln("Failed to get k8s config")
		}
		config = inClusterConfig
	}

	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalln("Failed to create k8s client")
	}

	factory := informers.NewSharedInformerFactory(clientSet, 0)
	servicesInformer := factory.Core().V1().Services()
	ingressesInformer := factory.Networking().V1().Ingresses()

	controller := pkg.NewController(clientSet, servicesInformer, ingressesInformer)

	stopCh := make(chan struct{})
	factory.Start(stopCh)
	factory.WaitForCacheSync(stopCh)

	controller.Run(stopCh)
}
