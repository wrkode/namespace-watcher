package main

import (
	"context"
	"fmt"
	"os"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

func main() {
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	namespaceInformer := cache.NewSharedIndexInformer(
		cache.NewListWatchFromClient(clientset.CoreV1().RESTClient(), "namespaces", metav1.NamespaceAll, nil),
		&v1.Namespace{},
		0,
		cache.Indexers{},
	)

	namespaceInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			namespace := obj.(*v1.Namespace)

			limitRange := &v1.LimitRange{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "default-limits",
					Namespace: namespace.Name,
				},
				Spec: v1.LimitRangeSpec{
					Limits: []v1.LimitRangeItem{
						{
							Type: v1.LimitTypeContainer,
							Default: v1.ResourceList{
								v1.ResourceMemory: resource.MustParse("512Mi"),
							},
							DefaultRequest: v1.ResourceList{
								v1.ResourceMemory: resource.MustParse("256Mi"),
							},
							Max: v1.ResourceList{
								v1.ResourceMemory: resource.MustParse("1Gi"),
							},
						},
					},
				},
			}

			_, err := clientset.CoreV1().LimitRanges(namespace.Name).Create(context.Background(), limitRange, metav1.CreateOptions{})
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating LimitRange for Namespace %s: %v", namespace.Name, err)
			}
		},
	})

	stopCh := make(chan struct{})
	defer close(stopCh)

	go namespaceInformer.Run(stopCh)

	if !cache.WaitForCacheSync(stopCh, namespaceInformer.HasSynced) {
		panic("Timed out waiting for caches to sync")
	}

	wait.Forever(func() { time.Sleep(time.Minute) }, time.Second)
}
