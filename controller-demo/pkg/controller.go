package pkg

import (
	"context"
	"fmt"
	"reflect"
	"time"

	v14 "k8s.io/api/core/v1"
	v12 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v13 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	informer "k8s.io/client-go/informers/core/v1"
	netInformer "k8s.io/client-go/informers/networking/v1"
	"k8s.io/client-go/kubernetes"
	coreLister "k8s.io/client-go/listers/core/v1"
	v1 "k8s.io/client-go/listers/networking/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

const (
	workNum    = 5
	maxRetries = 10
)

type controller struct {
	client          kubernetes.Interface
	ingressesLister v1.IngressLister
	servicesLister  coreLister.ServiceLister
	queue           workqueue.RateLimitingInterface
}

func (c *controller) addService(obj interface{}) {
	c.enqueue(obj)
}

func (c *controller) updateService(oldObj, newObj interface{}) {
	if reflect.DeepEqual(oldObj, newObj) {
		return
	}
	c.enqueue(newObj)
}

func (c *controller) enqueue(obj interface{}) {
	// Get obj's key
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		runtime.HandleError(fmt.Errorf("couldn't get key for object %+v: %v", obj, err))
		return
	}
	// Add obj's key to Workqueue
	c.queue.Add(key)
}

func (c *controller) deleteIngress(obj interface{}) {
	ingress := obj.(*v12.Ingress)
	ownerReference := v13.GetControllerOf(ingress)
	if ownerReference == nil {
		return
	}
	if ownerReference.Kind != "Service" {
		return
	}
	// When deleting ingress resource, we add the Namespace/Name of the ingress resource to the queue,
	// so that when the ingress is deleted, the controller will check the service resource and create the ingress resource again.
	c.queue.Add(ingress.Namespace + "/" + ingress.Name)
	// c.enqueue(ingress.Namespace + "/" + ingress.Name)
}

func (c *controller) Run(stopCh chan struct{}) {
	for i := 0; i < workNum; i++ {
		go wait.Until(c.worker, time.Minute, stopCh)
	}

	<-stopCh
}

func (c *controller) worker() {
	for c.processNextItem() {
	}
}

func (c *controller) processNextItem() bool {
	item, quit := c.queue.Get()
	if quit {
		return false
	}
	defer c.queue.Done(item)

	key := item.(string)

	err := c.syncService(key)
	if err != nil {
		c.handleError(key, err)
	}
	return true
}

func (c *controller) syncService(key string) error {
	namespaceKey, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}

	// delete
	service, err := c.servicesLister.Services(namespaceKey).Get(name)
	if errors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return nil
	}

	//
	_, ok := service.GetAnnotations()["ingress/http"]
	ingress, err := c.ingressesLister.Ingresses(namespaceKey).Get(name)
	if err != nil && !errors.IsNotFound(err) {
		return nil
	}

	ig := c.constructIngress(service)

	if ok && errors.IsNotFound(err) {
		// Create Ingress
		_, err = c.client.NetworkingV1().Ingresses(namespaceKey).Create(context.TODO(), ig, v13.CreateOptions{})
		if err != nil {
			return err
		}

	} else if !ok && ingress != nil {
		// Delete Ingress
		err := c.client.NetworkingV1().Ingresses(namespaceKey).Delete(context.TODO(), name, v13.DeleteOptions{})
		if err != nil {
			return err
		}

	}

	return nil
}

func (c *controller) handleError(key string, err error) {
	if c.queue.NumRequeues(key) <= maxRetries {
		c.queue.AddRateLimited(key)
		return
	}

	runtime.HandleError(err)
	c.queue.Forget(key)
}

func (c *controller) constructIngress(service *v14.Service) *v12.Ingress {
	pathType := v12.PathTypePrefix
	igClassName := "nginx"
	return &v12.Ingress{
		ObjectMeta: v13.ObjectMeta{
			Name:      service.Name,
			Namespace: service.Namespace,
			OwnerReferences: []v13.OwnerReference{
				*v13.NewControllerRef(service, v14.SchemeGroupVersion.WithKind("Service")),
			},
		},
		Spec: v12.IngressSpec{
			IngressClassName: &igClassName,
			Rules: []v12.IngressRule{
				{
					Host: "example.com",
					IngressRuleValue: v12.IngressRuleValue{
						HTTP: &v12.HTTPIngressRuleValue{
							Paths: []v12.HTTPIngressPath{
								{
									Path:     "/",
									PathType: &pathType,
									Backend: v12.IngressBackend{
										Service: &v12.IngressServiceBackend{
											Name: service.Name,
											Port: v12.ServiceBackendPort{
												Number: 80,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func NewController(client kubernetes.Interface, servicesInformer informer.ServiceInformer, ingressesInformer netInformer.IngressInformer) controller {
	c := controller{
		client:          client,
		ingressesLister: ingressesInformer.Lister(),
		servicesLister:  servicesInformer.Lister(),
		queue:           workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "ingressMananger"),
	}

	servicesInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addService,
		UpdateFunc: c.updateService,
	})

	ingressesInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		DeleteFunc: c.deleteIngress,
	})
	return c
}
