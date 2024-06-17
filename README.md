# Development

## Building

Requirements

- Kubectl
- Docker
- Minikube

### Set docker env

This needs to be done every time before building, otherwise minikube will not update the image

```
eval $(minikube docker-env)                      # Unix shells
minikube docker-env | Invoke-Expression # PowerShell
```

### Build docker image

```
docker build -t clyent.dev/cloudflare-cloud-controller-manager:v0.0.1 .
```

### Apply deployment

```
kubectl apply -k deploy
```
