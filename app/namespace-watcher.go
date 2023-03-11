package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func createLimitRange(clientset *kubernetes.Clientset, namespaceName string) error {

	version := "v1.0-alpha"

	cpuLimitMax := os.Getenv("CPU_LIMIT_MAX")
	memoryLimitMax := os.Getenv("MEM_LIMIT_MAX")
	ephemeralStorageLimitMax := os.Getenv("EPHEMERAL_STORAGE_MAX")

	cpuLimitMin := os.Getenv("CPU_LIMIT_MIN")
	memoryLimitMin := os.Getenv("MEM_LIMIT_MIN")
	ephemeralStorageLimitMin := os.Getenv("EPHEMERAL_STORAGE_MIN")
	log.Printf("Namespace-Watcher version %s\n", version)

	log.Println("Starting Namespace-Watcher with the folling parameters:")
	log.Printf("CPU_LIMIT_MAX %s\n", cpuLimitMax)
	log.Printf("CPU_LIMIT_MIN %s\n", cpuLimitMin)
	log.Printf("MEM_LIMIT_MAX %s\n", memoryLimitMax)
	log.Printf("MEM_LIMIT_MAX %s\n", memoryLimitMin)
	log.Printf("EPHEMERAL_STORAGE_MAX %s\n", ephemeralStorageLimitMax)
	log.Printf("EPHEMERAL_STORAGE_MAX %s\n", ephemeralStorageLimitMin)

	limitRange := &corev1.LimitRange{
		ObjectMeta: metav1.ObjectMeta{
			Name: "default-limits",
		},
		Spec: corev1.LimitRangeSpec{
			Limits: []corev1.LimitRangeItem{
				{
					Type: corev1.LimitTypePod,
					Max: corev1.ResourceList{
						corev1.ResourceCPU:              resource.MustParse(cpuLimitMax),
						corev1.ResourceMemory:           resource.MustParse(memoryLimitMax),
						corev1.ResourceEphemeralStorage: resource.MustParse(ephemeralStorageLimitMax),
					},
					Min: corev1.ResourceList{
						corev1.ResourceCPU:              resource.MustParse(cpuLimitMin),
						corev1.ResourceMemory:           resource.MustParse(memoryLimitMin),
						corev1.ResourceEphemeralStorage: resource.MustParse(ephemeralStorageLimitMin),
					},
				},
			},
		},
	}

	_, err := clientset.CoreV1().LimitRanges(namespaceName).Create(context.Background(), limitRange, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	fmt.Printf("Created LimitRange for namespace %s\n", namespaceName)
	return nil
}

func main() {

	// create Kubernetes API client
	config, err := rest.InClusterConfig()
	if err != nil {
		fmt.Println("Failed to get Kubernetes config:", err)
		os.Exit(1)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Println("Failed to create Kubernetes client:", err)
		os.Exit(1)
	}

	// watch for namespace creation events
	watcher, err := clientset.CoreV1().Namespaces().Watch(context.Background(), metav1.ListOptions{})
	if err != nil {
		fmt.Println("Failed to watch namespaces:", err)
		os.Exit(1)
	}

	// process events
	for event := range watcher.ResultChan() {
		// check if event is a namespace creation event
		if event.Type == watch.Added {
			// print namespace name to STDOUT
			namespaceName := event.Object.(*corev1.Namespace).ObjectMeta.Name

			//exclude cattle and system namespaces
			if strings.Contains(namespaceName, "default") {
				log.Printf("Skipping namespace %s\n", namespaceName)
				continue
			}
			if strings.Contains(namespaceName, "cattle") || strings.Contains(namespaceName, "kube-system") || strings.Contains(namespaceName, "kube-public") {
				log.Printf("Skipping namespace %s\n", namespaceName)
				continue
			}
			if strings.Contains(namespaceName, "istio-system") || strings.Contains(namespaceName, "kube-public") || strings.Contains(namespaceName, "kube-local") {
				log.Printf("Skipping namespace %s\n", namespaceName)
				continue
			}

			fmt.Printf("New namespace created: %s\n", namespaceName)

			// create a LimitRange for the new namespace
			err := createLimitRange(clientset, namespaceName)
			if err != nil {
				fmt.Printf("Failed to create LimitRange for namespace %s: %v\n", namespaceName, err)
			}
		}
	}
}
