# Cloudflare Cloud Controller Manager

A Kubernetes [cloud controller manager](https://kubernetes.io/docs/concepts/architecture/cloud-controller/)
that provisions Cloudflare Load Balancers for Services of `type: LoadBalancer`.

## Configuration

The controller is configured with environment variables. Each variable can also
be read from a file via `<NAME>_FILE` (useful for mounted secrets).

| Variable                | Required | Description                           |
| ----------------------- | -------- | ------------------------------------- |
| `CLOUDFLARE_API_TOKEN`  | yes      | API token, see scopes below           |
| `CLOUDFLARE_ZONE_ID`    | yes      | Zone that receives the load balancers |
| `CLOUDFLARE_ACCOUNT_ID` | yes      | Account that holds pools and monitors |

The API token needs:

- Account / Load Balancing: Monitors and Pools / Edit
- Account / Account Settings / Read
- Zone / Load Balancers / Edit

## Service annotations

All annotations use the prefix `cloudflare-load-balancer.clyent.dev/`.

| Annotation               | Default    | Description                                    |
| ------------------------ | ---------- | ---------------------------------------------- |
| `hostname`               | (required) | DNS hostname for the load balancer             |
| `monitor-type`           | `http`     | Monitor protocol: `http`, `https`, `tcp`, ...  |
| `monitor-path`           | `/`        | Path probed by the health check                |
| `monitor-method`         | `GET`      | HTTP method used by the health check           |
| `monitor-expected-codes` | `2xx`      | Response codes treated as healthy              |
| `monitor-interval`       | `60`       | Seconds between health checks                  |
| `monitor-timeout`        | `5`        | Health check timeout in seconds                |
| `monitor-retries`        | `2`        | Retries before an origin is marked unhealthy   |
| `monitor-allow-insecure` | `false`    | Skip TLS validation for HTTPS checks           |
| `monitor-probe-zone`     |            | Emulate a zone while probing (HTTP/HTTPS only) |
| `monitor-header`         |            | `Host` header sent with health checks          |

Annotation values are strings, so quote booleans and numbers in YAML
(`monitor-allow-insecure: 'true'`). Only the first Service port is used.

See [sample/service.yaml](sample/service.yaml) for a complete example.

## Deployment

1. Create the secret with your Cloudflare credentials
   (see [deploy/secret.yaml](deploy/secret.yaml)).
2. Apply the manifests:

   ```
   kubectl apply -k deploy
   ```

## Development

```
make build   # build the binary into bin/
make test    # go test ./... -race
make vet     # go vet
make lint    # golangci-lint
make docker-build
```

### Local testing with minikube

```
# point docker at minikube (repeat per shell)
minikube docker-env | Invoke-Expression   # PowerShell
eval $(minikube docker-env)               # Unix shells

make docker-build IMAGE=clyent.dev/cloudflare-cloud-controller-manager VERSION=dev
kubectl apply -k deploy
```
