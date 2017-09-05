# k8nskel

TODO: Add Travis CI badge

Kubernetes Controller to distribute Secrets to new Namespace on Kubernetes.

## Requirements

- Kubernetes 1.6 or above

## Installation

### From source

```console
$ git clone git@github.com:wantedly/k8nskel.git
$ cd k8nskel
$ make deps
$ make
```

### Docker image

Docker image is available at [`quay.io/wantedly/k8nskel`](https://quay.io/repository/wantedly/k8nskel).

## Environment variables

|Name|Description|Default value|
|-|-|-|
|K8NSKEL_ORIGIN|Name of the namespace from which the secret is copied.|"k8nskel-origin"|
|K8NSKEL_IGNORE_DEST|CSV list of namespaces that does not reflect secrets in `K8NSKEL_ORIGIN` is added/modified/deleted. It is not reflected in `K8NSKEL_ORIGIN` by default.|"kube-public,kube-system"|

## Usage

k8nskel copies all secrets in `K8NSKEL_ORIGIN` namespace to the new namespace.  
Also, when secrets in `K8NSKEL_ORIGIN` is created/modified/deleted, it reflects its secrets to other namespaces than namespace set to` K8NSKEL_IGNORE_DEST`.

### Workflow example

1. Create `K8NSKEL_ORIGIN` namespace.

  ```console
  # e.g.
  $ kubectl create namespace k8nskel-origin
  ```

2. Create `k8nskel` deployment.

  ```console
  # e.g.
  $ kubectl run --rm -i k8nskel --image=quay.io/wantedly/k8nskel:latest
  ```

3. Create a secret.

  ```console
  # e.g.
  $ kubectl --namespace k8nskel-origin create secret generic secret1 --from-literal=key1=supersecret
  ```

4. Create a new namespace.

  ```console
  # e.g.
  $ kubectl create namespace new-namespace
  ```

5. Get secrets of new namespace. The secret created earlier should be displayed.

  ```console
  # e.g.
  $ kubectl --namespace k8nskel-origin get secret
  ```

6. Add a secret in `K8NSKEL_ORIGIN`. The same secret should have been added to other namespaces.

  ```console
  # e.g.
  $ kubectl --namespace k8nskel-origin create secret generic secret2 --from-literal=key2=supersecret
  $ kubectl --namespace new-namespace get secret
  ```

7. Modify a secret in `K8NSKEL_ORIGIN`. The same secret should have been modified in other namespaces.

  ```console
  # e.g.
  $ kubectl --namespace k8nskel-origin edit secret secret2
  $ kubectl --namespace new-namespace describe secret secret2
  ```

8. Delete a secret in `K8NSKEL_ORIGIN`. The same secret should have been deleted from other namespaces.

  ```console
  # e.g.
  $ kubectl --namespace k8nskel-origin delete secret secret2
  $ kubectl --namespace new-namespace get secret
  ```

### Manifest sample

- Namespace manifest sample:

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: k8nskel-origin
```

- Deployment manifest sample:

```yaml
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: k8nskel
  namespace: k8nskel-origin
  labels:
    name: k8nskel
spec:
  replicas: 1
  template:
    metadata:
      name: k8nskel
      labels:
        name: k8nskel
    spec:
      containers:
        - name: k8nskel
          image: quay.io/wantedly/k8nskel:latest
```
