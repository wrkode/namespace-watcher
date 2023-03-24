package main

import (
	"context"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const version = "v1.0-alpha6"

type Limits struct {
	CpuLimitMax              string
	MemLimitMax              string
	EphemeralStorageLimitMax string
	CpuLimitMin              string
	MemLimitMin              string
	EphemeralStorageLimitMin string
}

var setLimits Limits

func createOrUpdateLimitRange(clientset *kubernetes.Clientset, namespaceName string, limits Limits) error {

	cpuLimitMin, err := resource.ParseQuantity(limits.CpuLimitMin)
	if err != nil {
		logrus.Fatalf("error parsing CPU_LIMIT_MIN: ", err)
	}

	cpuLimitMax, err := resource.ParseQuantity(limits.CpuLimitMax)
	if err != nil {
		logrus.Fatalf("error parsing CPU_LIMIT_MAX :", err)
	}

	memLimtMin, err := resource.ParseQuantity(limits.MemLimitMin)
	if err != nil {
		logrus.Fatalf("error parsing MEM_LIMIT_MIN: ", err)
	}

	memLimtMax, err := resource.ParseQuantity(limits.MemLimitMax)
	if err != nil {
		logrus.Fatalf("error parsing MEM_LIMIT_MAX: ", err)
	}

	ephemeralStorageMin, err := resource.ParseQuantity(limits.EphemeralStorageLimitMin)
	if err != nil {
		logrus.Fatalf("error parsing EPHEMERAL_STORAGE_MIN: ", err)
	}

	ephemeralStorageMax, err := resource.ParseQuantity(limits.EphemeralStorageLimitMax)
	if err != nil {
		logrus.Fatalf("error parsing EPHEMERAL_STORAGE_MAX: ", err)
	}

	limitRange := &corev1.LimitRange{
		ObjectMeta: metav1.ObjectMeta{
			Name: "default-limits",
		},
		Spec: corev1.LimitRangeSpec{
			Limits: []corev1.LimitRangeItem{
				{
					Type: corev1.LimitTypePod,
					Max: corev1.ResourceList{
						corev1.ResourceCPU:              cpuLimitMax,
						corev1.ResourceMemory:           memLimtMax,
						corev1.ResourceEphemeralStorage: ephemeralStorageMax,
					},
					Min: corev1.ResourceList{
						corev1.ResourceCPU:              cpuLimitMin,
						corev1.ResourceMemory:           memLimtMin,
						corev1.ResourceEphemeralStorage: ephemeralStorageMin,
					},
				},
			},
		},
	}

	// Get the existing LimitRange
	existingLimitRange, err := clientset.CoreV1().LimitRanges(namespaceName).Get(context.Background(), limitRange.ObjectMeta.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {

			//Create LimitRange if it doesn't exist
			_, err = clientset.CoreV1().LimitRanges(namespaceName).Create(context.Background(), limitRange, metav1.CreateOptions{})
			if err != nil {
				return err
			}
			logrus.Warn("Created LimitRange for namespace: ", namespaceName)
		} else {
			return err

		}
	} else {
		// Update the existing LimitRange
		existingLimitRange.Spec = limitRange.Spec
		_, err = clientset.CoreV1().LimitRanges(namespaceName).Update(context.Background(), existingLimitRange, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
		logrus.Warn("Updated LimitRange for namespace: ", namespaceName)
	}

	return nil
}

func main() {

	logrus.Info("Namespace-Watcher version ", version)

	setLimits.CpuLimitMin = os.Getenv("CPU_LIMIT_MIN")
	setLimits.CpuLimitMax = os.Getenv("CPU_LIMIT_MAX")
	setLimits.MemLimitMin = os.Getenv("MEM_LIMIT_MIN")
	setLimits.MemLimitMax = os.Getenv("MEM_LIMIT_MAX")
	setLimits.EphemeralStorageLimitMin = os.Getenv("EPHEMERAL_STORAGE_MIN")
	setLimits.EphemeralStorageLimitMax = os.Getenv("EPHEMERAL_STORAGE_MAX")

	logrus.Info("Starting Namespace-Watcher with the following parameters:")
	logrus.Info("CPU_LIMIT_MIN ", setLimits.CpuLimitMin)
	logrus.Info("CPU_LIMIT_MAX ", setLimits.CpuLimitMax)
	logrus.Info("MEM_LIMIT_MIN ", setLimits.MemLimitMin)
	logrus.Info("MEM_LIMIT_MAX ", setLimits.MemLimitMax)
	logrus.Info("EPHEMERAL_STORAGE_MIN ", setLimits.EphemeralStorageLimitMin)
	logrus.Info("EPHEMERAL_STORAGE_MAX ", setLimits.EphemeralStorageLimitMax)

	// create Kubernetes API client
	config, err := rest.InClusterConfig()
	if err != nil {
		logrus.Fatal("Failed to get Kubernetes config: ", err)
		os.Exit(1)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		logrus.Fatal("Failed to create Kubernetes client: ", err)
		os.Exit(1)
	}

	// watch for namespace creation events
	watcher, err := clientset.CoreV1().Namespaces().Watch(context.Background(), metav1.ListOptions{})
	if err != nil {
		logrus.Fatal("Failed to watch namespaces: ", err)
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
				logrus.Info("Skipping namespace ", namespaceName)
				continue
			}
			if strings.Contains(namespaceName, "cattle") || strings.Contains(namespaceName, "kube-system") || strings.Contains(namespaceName, "kube-public") {
				logrus.Info("Skipping namespace ", namespaceName)
				continue
			}
			if strings.Contains(namespaceName, "istio-system") || strings.Contains(namespaceName, "kube-node-lease") || strings.Contains(namespaceName, "kube-local") {
				logrus.Info("Skipping namespace ", namespaceName)
				continue
			}

			logrus.Info("New namespace created: ", namespaceName)

			// create a LimitRange for the new namespace
			err := createOrUpdateLimitRange(clientset, namespaceName, setLimits)
			if err != nil {
				logrus.Warn("Failed to create LimitRange for namespace: ", namespaceName, " ", err)
			}
		}
	}
}
