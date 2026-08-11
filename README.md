# Delphi

Delphi contains the services and supporting containers used by Odysseia-Greek to provision secrets, configure Vault access, and manage Kubernetes network access.

The repository is a collection of independent Go modules. There is no root `go.mod`; run Go commands from a component directory or use the root `Makefile` to operate on every module.

## Components

| Component | Type | Responsibility |
| --- | --- | --- |
| `solon` | HTTP API | Validates workloads and brokers access to Vault and Elasticsearch. It also watches Kubernetes workloads so obsolete credentials can be cleaned up. |
| `perikles` | Kubernetes controller | Watches workloads and configuration, maintains service mappings, TLS secrets, and Cilium network policies, and exposes a small dashboard. |
| `aristides` | gRPC sidecar | Gives a workload a local interface for retrieving credentials through Solon. Its protobuf contract is in `aristides/proto`. |
| `peisistratos` | Init container | Bootstraps Vault authentication, policies, and related configuration before Solon starts. |
| `periandros` | Init container | Requests and prepares Elasticsearch credentials for a workload. |
| `ktesias` | Integration test suite | Exercises the deployed Solon, Perikles, Vault, Elasticsearch, Kubernetes, and Cilium flow. |

In broad terms, applications use Aristides or an init container to request credentials; Solon verifies the requesting Kubernetes workload and talks to the backing systems; Perikles reconciles the cluster resources and network access needed for that communication.

## Repository layout

Each component directory contains its own:

- `go.mod` and `go.sum`
- `Containerfile`
- application or test code

Cluster deployment manifests and Helm charts are maintained in the separate `mykenai` repository. The local Skaffold profile in this repository refers to that repository through a relative path.

## Development

### Prerequisites

- A Go version compatible with the `go` directive in the module being changed
- `make`
- A container builder for building images
- For cluster development: Kubernetes, Helm, Skaffold, Cilium, Vault, and a checkout of the `mykenai` repository at the relative path expected by `skaffold.yaml`

### Run unit tests

Run tests for one component from its module directory:

```sh
cd solon
go test ./...
```

To test every module from the repository root:

```sh
for module in aristides ktesias peisistratos periandros perikles solon; do
  (cd "$module" && go test ./...)
done
```

`ktesias` is an integration suite and expects a configured, running cluster. It is not a standalone unit-test module.

### Format and tidy all modules

```sh
make tidy
```

This runs `go mod tidy` and `go fmt ./...` in every module. Be aware that `go mod tidy` can change both `go.mod` and `go.sum`.

### Build a component

Build a binary locally:

```sh
cd solon
go build ./...
```

Build its production container image from the repository root:

```sh
docker build --target prod --build-arg project_name=solon -t solon:dev solon
```

Replace `solon` with the component being built. `ktesias` produces a test binary rather than a service binary.

### Local cluster workflow

The current Skaffold configuration contains the `alexandros` profile for Solon and targets the `k3d-odysseia` Kubernetes context:

```sh
skaffold dev --profile alexandros
```

Before running it, verify the chart and values paths in `skaffold.yaml`; they are relative to the layout of the wider Odysseia-Greek workspace.

## Dependency upgrades

Dependencies must be upgraded per module. A practical sequence is:

1. Upgrade one component at a time with `go get` from that component's directory.
2. Run `go mod tidy` and `go test ./...` in that module.
3. Build its container image to catch toolchain or base-image issues.
4. Upgrade shared libraries and tightly coupled Kubernetes packages together where required.
5. Run the `ktesias` suite against a deployed cluster after the individual modules pass.

Useful inspection commands:

```sh
cd solon
go list -m -u all
go mod graph
```

The Kubernetes and Cilium dependency trees are large, particularly in `perikles` and `ktesias`. Upgrade those deliberately and keep their compatible package versions aligned rather than applying a repository-wide version bump blindly.

## Configuration

The services are designed to run in Kubernetes and obtain most configuration from environment variables and mounted TLS material. Common settings include `NAMESPACE`, `POD_NAME`, `PORT`, and the TLS-related variables supplied by the shared Agora packages.

Component-specific settings visible in this repository include:

- Solon: `PORT`, `CERT_ROOT`, `SOLON_MANAGED_NAMESPACES`
- Perikles: `CRD_NAME`, `TLS_FILES`, `L7_MODE`, `CONFIGMAP_NAME`, `VAULT_NAMESPACE`, `ELASTIC_NAMESPACE`, `WATCHED_NAMESPACES`, `DASHBOARD_ADDR`
- Aristides: `PORT`
- Peisistratos: `ENV`
- Periandros: workload role, index/access, pod, namespace, and tracing settings supplied through the shared configuration package

For deployment defaults and secret mounts, treat the Helm charts in `mykenai` as the source of truth.

## API and custom resource references

- Solon's OpenAPI definition: `solon/docs/solon.yaml`
- Aristides gRPC definition: `aristides/proto/aristides.proto`
- Perikles service-mapping CRD: `perikles/pkg/service_mapping/crd/v1alpha`
- Service-mapping notes: `perikles/pkg/service_mapping/README.md`

## License

See [LICENSE](LICENSE).
