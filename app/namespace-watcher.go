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

//var cpuLimit string
//var memLimit string
//var ephemeralStorageLimit string

func main() {

	cpuLimit := os.Getenv("CPU_LIMIT")
	memLimit := os.Getenv("MEM_LIMIT")
	ephemeralStorageLimit := os.Getenv("EPHEMERAL_STORAGE_LIMIT")

	fmt.Printf("CPU Limit: %v\n", cpuLimit)
	fmt.Printf("MEM Limit: %v\n", memLimit)
	fmt.Printf("EPHEMERAL STORAGE Limit: %v\n", ephemeralStorageLimit)

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

			fmt.Printf("Namespace added: %s\n", namespace.Name)

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
								v1.ResourceMemory: resource.MustParse(memLimit),
								v1.ResourceCPU:    resource.MustParse(cpuLimit),
							},
							Max: v1.ResourceList{
								v1.ResourceMemory: resource.MustParse(ephemeralStorageLimit),
							},
						},
					},
				},
			}

			fmt.Printf("Creating LimitRange for Namespace %s: %+v\n", namespace.Name, limitRange)

			_, err := clientset.CoreV1().LimitRanges(namespace.Name).Create(context.Background(), limitRange, metav1.CreateOptions{})
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating LimitRange for Namespace %s: %v", namespace.Name, err)
			} else {
				fmt.Printf("LimitRange created for Namespace %s\n", namespace.Name)
			}
		},
	})

	stopCh := make(chan struct{})
	defer close(stopCh)

	go namespaceInformer.Run(stopCh)
	fmt.Printf("Namespace Informer: %+v\n", namespaceInformer)

	if !cache.WaitForCacheSync(stopCh, namespaceInformer.HasSynced) {
		panic("Timed out waiting for caches to sync")
	}

	wait.Forever(func() { time.Sleep(time.Minute) }, time.Second)
}
