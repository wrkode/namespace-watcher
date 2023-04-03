# Namespace-watcher

A Kubernetes Namespace Watcher.
namespace-watcher will watch the Kubernetes Event Stream and will add a LimitRange named ```default-limits``` for CPU, MEM and epheremeral-storage to any new namespace created. If the ```default-limits``` exists, it will be updated with the parameters defined in the manifest.
All limits (MIN, MAX) are required fields.

## Requirements

kube-apiserver must have the the following admission-plugins enabled:
```LimitRanger,DefaultStorageClass,NamespaceLifecycle,LimitRanger,DefaultStorageClass,DefaultTolerationSeconds,NodeRestriction,MutatingAdmissionWebhook,ValidatingAdmissionWebhook,ResourceQuota,PodNodeSelector,PodPreset,DefaultLimitRange```

## Exclusions

namespace-watcher, will not add limits to:

- Any namespaces containing ```cattle``` in the name.
- ```kube-system```
- ```kube-public```
- ```istio-system```
- ```kube-local```
- ```kube-node-lease```
- ```default```
- ```local```

Additional namespaces can be exluded setting the env var ```EXCLUDED_NAMESPACES``` as shown in the ```deployment.yaml```. Thie is a list of string comma (```,```) separated.

## Deployment

The necessary serviceaccount, ClusterRole and ClusterRoleBinding are created in the manifest ```deployment.yaml``` to allow namespace-watcher to observe and update namespaces.
download the manifest deployment.yaml and run:
```kubectl apply -f deployment.yaml```
namespace-watcher will be deployed in the ```kube-system``` namespace.
