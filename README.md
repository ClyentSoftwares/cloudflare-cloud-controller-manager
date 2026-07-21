# Cloudflare Cloud Controller Manager

A Kubernetes [cloud controller manager](https://kubernetes.io/docs/concepts/architecture/cloud-controller/)
that provisions **Cloudflare Load Balancers** for Services of `type: LoadBalancer`.

For every Service carrying the `cloudflare-load-balancer.clyent.dev/hostname`
annotation, the controller creates and keeps in sync:

- a **health monitor** (account-scoped),
- an **origin pool** whose origins are the cluster nodes' external IPs, and
- a **load balancer** on the configured zone, proxied through Cloudflare.

Services without the hostname annotation are ignored. Deleting the Service
deletes all three resources.

## How traffic flows

Cloudflare proxies requests for the configured hostname to the node external
IPs on the Service's port (e.g. `80`/`443`). The origins carry no port mapping,
so **something on each node must listen on that port** — typically an ingress
controller using `hostPort` or `hostNetwork`. NodePorts are not used.

> **Note:** this CCM only implements the LoadBalancer interface. It does not
> initialize nodes, so kubelets must **not** run with
> `--cloud-provider=external` unless another controller removes the
> `node.cloudprovider.kubernetes.io/uninitialized` taint.

## Configuration

The controller is configured with environment variables. Every variable can
alternatively be read from a file via a `<NAME>_FILE` variable (useful for
mounted secrets).

| Variable                | Required | Description                           |
| ----------------------- | -------- | ------------------------------------- |
| `CLOUDFLARE_API_TOKEN`  | yes      | API token, see scopes below           |
| `CLOUDFLARE_ZONE_ID`    | yes      | Zone that receives the load balancers |
| `CLOUDFLARE_ACCOUNT_ID` | yes      | Account that holds pools and monitors |

The API token needs:

- **Account → Load Balancing: Monitors and Pools → Edit**
- **Account → Account Settings → Read** (used to validate credentials at startup)
- **Zone → Load Balancers → Edit** (for the configured zone)

## Service annotations

| Annotation                                     | Default      | Description                                    |
| ---------------------------------------------- | ------------ | ---------------------------------------------- |
| `cloudflare-load-balancer.clyent.dev/hostname` | — (required) | DNS hostname for the load balancer             |
| `.../monitor-type`                             | `http`       | Monitor protocol: `http`, `https`, `tcp`, ...  |
| `.../monitor-path`                             | `/`          | Path probed by the health check                |
| `.../monitor-method`                           | `GET`        | HTTP method used by the health check           |
| `.../monitor-expected-codes`                   | `2xx`        | Response codes treated as healthy              |
| `.../monitor-interval`                         | `60`         | Seconds between health checks                  |
| `.../monitor-timeout`                          | `5`          | Health check timeout in seconds                |
| `.../monitor-retries`                          | `2`          | Retries before an origin is marked unhealthy   |
| `.../monitor-allow-insecure`                   | `false`      | Skip TLS validation for HTTPS checks           |
| `.../monitor-probe-zone`                       | —            | Emulate a zone while probing (HTTP/HTTPS only) |
| `.../monitor-header`                           | —            | `Host` header sent with health checks          |

Annotation values are strings — quote booleans and numbers in YAML
(`monitor-allow-insecure: 'true'`). Only the first Service port is used.

See [sample/service.yaml](sample/service.yaml) for a complete example.

## Deployment

1. Create the secret with your Cloudflare credentials
   (see [deploy/secret.yaml](deploy/secret.yaml)).
2. Apply the manifests:

   ```
   kubectl apply -k deploy
   ```

The deployment runs the CCM with `--controllers=service` (the only controller
this provider implements), `--allow-untagged-cloud` (Cloudflare has no cluster
tagging) and a namespaced-scoped RBAC role.

## Resource naming

For a hostname `hello.test.com` the controller manages:

- load balancer: `hello.test.com`
- pool: `hello-test-com-pool`
- monitor (description): `hello-test-com-monitor`
- origins: sanitized node IPs, e.g. `203-0-113-1`

Names are sanitized to the Cloudflare-accepted character set
(`a-z`, `0-9`, `_`, `-`); the load balancer name stays a hostname because it
doubles as the DNS record.

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
