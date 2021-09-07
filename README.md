# K8S Sidecar Controller

A simple kubernetes controller which ensures proper termination of sidecar containers

## Tested on

- go v1.17
- Kubernetes 1.19

## Usage

1. Deploy this controller. See [How to deploy](#how-to-deploy)
2. Add an annotation to pods which have sidecar containers. Annotation format is `limgit/sidecars: <sidecar>,<container>,<name>`
```yaml
# Example pod configuration.
# After `main` container finishes,
# the sidecar controller will send SIGTERM to `sidecar` container
apiVersion: v1
kind: Pod
metadata:
  name: busybox-sleep
  namespace: default
  annotations:
    limgit/sidecars: sidecar
spec:
  containers:
  - name: sidecar
    image: busybox
    args:
    - sleep
    - "1000000"
  - name: main
    image: busybox
    args:
    - sleep
    - "20"
```
3. Tada! This controller will clean up sidecar containers if every containers but sidecar containers are completed!

## How to deploy

1. Build docker image with `Dockefile`
2. Push docker image to image repository
3. In `deployment.yaml`, set correct image tag for `Deployment` resource
4. `kubectl apply -f deployment.yaml`
