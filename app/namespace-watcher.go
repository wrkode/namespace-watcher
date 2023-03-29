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

const version = "v1.0-alpha13"

type Limits struct {
	CpuLimitMax              resource.Quantity
	MemLimitMax              resource.Quantity
	EphemeralStorageLimitMax resource.Quantity
	CpuLimitMin              resource.Quantity
	MemLimitMin              resource.Quantity
	EphemeralStorageLimitMin resource.Quantity
}

var setLimits Limits

func createOrUpdateLimitRange(clientset *kubernetes.Clientset, namespaceName string, limits Limits) error {

	limitRange := &corev1.LimitRange{
		ObjectMeta: metav1.ObjectMeta{
			Name: "default-limits",
		},
		Spec: corev1.LimitRangeSpec{
			Limits: []corev1.LimitRangeItem{
				{
					Type: corev1.LimitTypePod,
					Max: corev1.ResourceList{
						corev1.ResourceCPU:              limits.CpuLimitMax,
						corev1.ResourceMemory:           limits.MemLimitMax,
						corev1.ResourceEphemeralStorage: limits.EphemeralStorageLimitMax,
					},
					Min: corev1.ResourceList{
						corev1.ResourceCPU:              limits.CpuLimitMin,
						corev1.ResourceMemory:           limits.MemLimitMin,
						corev1.ResourceEphemeralStorage: limits.EphemeralStorageLimitMin,
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
			logrus.Info("Created LimitRange for namespace: ", namespaceName)
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
		logrus.Info("Updated LimitRange for namespace: ", namespaceName)
	}

	return nil
}

func lookupEnvOrEmpty(key string) resource.Quantity {
	value, exists := os.LookupEnv(key)
	if !exists || len(strings.TrimSpace(value)) == 0 {
		logrus.Error("Environment variable ", key, " is not set or empty: ", value)
		return resource.Quantity{}
	}
	q, err := resource.ParseQuantity(value)
	if err != nil {
		logrus.Error("Error parsing ", key, ": ", err)
		return resource.Quantity{}
	}
	return q
}

func shouldExcludeNamespace(namespaceName string, excludedNamespaces map[string]struct{}) bool {
	if strings.Contains(namespaceName, "cattle") {
		return true
	}

	_, isExcluded := excludedNamespaces[namespaceName]
	return isExcluded
}

func main() {

	// Define a map for environment variables and their corresponding limits
	envVars := map[string]*resource.Quantity{
		"CPU_LIMIT_MIN":         &setLimits.CpuLimitMin,
		"CPU_LIMIT_MAX":         &setLimits.CpuLimitMax,
		"MEM_LIMIT_MIN":         &setLimits.MemLimitMin,
		"MEM_LIMIT_MAX":         &setLimits.MemLimitMax,
		"EPHEMERAL_STORAGE_MIN": &setLimits.EphemeralStorageLimitMin,
		"EPHEMERAL_STORAGE_MAX": &setLimits.EphemeralStorageLimitMax,
	}

	// Loop through the environment variables, parse and set the limits
	for key, limit := range envVars {
		parsedLimit := lookupEnvOrEmpty(key)
		if parsedLimit.IsZero() {
			logrus.Fatal("Invalid value for ", key)
		}
		*limit = parsedLimit
	}

	logrus.Info("Namespace-Watcher version ", version)
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
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		logrus.Fatal("Failed to create Kubernetes client: ", err)
	}

	// watch for namespace creation events
	watcher, err := clientset.CoreV1().Namespaces().Watch(context.Background(), metav1.ListOptions{})
	if err != nil {
		logrus.Fatal("Failed to watch namespaces: ", err)

	}

	// create a set to store excluded namespaces
	excludedNamespaces := map[string]struct{}{
		"default":         {},
		"cattle":          {},
		"kube-system":     {},
		"kube-public":     {},
		"istio-system":    {},
		"kube-node-lease": {},
		"kube-local":      {},
	}

	// read additional excluded namespaces from the environment variable
	additionalExcluded := os.Getenv("EXCLUDED_NAMESPACES")
	if additionalExcluded != "" {
		additionalExcludedList := strings.Split(additionalExcluded, ",")
		for _, ns := range additionalExcludedList {
			excludedNamespaces[strings.TrimSpace(ns)] = struct{}{}
		}
	}

	// process events
	for event := range watcher.ResultChan() {
		// check if event is a namespace creation event
		if event.Type == watch.Added {
			namespaceName := event.Object.(*corev1.Namespace).ObjectMeta.Name

			// check if namespace is in the excluded set or if it contains word "cattle"
			isExcluded := shouldExcludeNamespace(namespaceName, excludedNamespaces)
			if isExcluded {
				logrus.Info("Checking if namespace should be skipped: ", namespaceName, " - Excluded: ", isExcluded)
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
