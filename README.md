# Namespace-watcher

A Kubernetes Namespace Watcher.
namespace-watcher will watch the Kubernetes Event Stream and add LimitRange ```default-limits``` for CPU, MEM and epheremeral-storage to any new namespace created

## Requirements

kube-apiserver must have the the following admission-plugins enabled:
```LimitRanger,DefaultStorageClass,NamespaceLifecycle,LimitRanger,DefaultStorageClass,DefaultTolerationSeconds,NodeRestriction,MutatingAdmissionWebhook,ValidatingAdmissionWebhook,ResourceQuota,PodNodeSelector,PodPreset,DefaultLimitRange```

## Exclusions

namespace-watcher, will not add limits to:

- Any namespaces containing ```cattle`` in te name.
- ```kube-system```
- ```kube-public```
- ```istio-system```
- ```kube-local```
- ```default```

## Deployment

A serviceaccount, ClusterRole and ClusterRoleBinding are created to allow namespace-watcher to observe and update namespaces.
download the manifest deployment.yaml and run:
```kubectl apply -f deployment.yaml```
namespace-watcher will be deployed in the ```kube-system``` namespace

## ToDo (future release)

- Implement namespace exclusion as array in the manifest file
