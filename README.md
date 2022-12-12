# node-tainter
Add taint to node by a result of script execution


## Usage

### Installation

```
kustomize manifests |kubectl create -f -
```

### Modify Script

Re-write configmap file following below:

https://github.com/takutakahashi/node-tainter/blob/main/manifests/scripts-configmap.yaml

and apply it.

```
kubectl apply -f scripts-configmap.yaml
```

You should manage them with kustomize.

### That's it!
