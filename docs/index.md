# go-welcome-app

Returns a welcome message. Used as the reference Go service on our platform.

| | |
|---|---|
| Owner | go-app-team |
| Language | Go |
| Image | `docker.io/<dockerhub-user>/go-welcome-app` |
| Deployed by | Argo CD, from `k8s-stuff/apps/go-welcome-app` |
| URL | https://welcome.local.io |

## Endpoints

| Path | Purpose |
|---|---|
| `/` | Returns a simple "Hello" message |
| `/welcome` | Returns hostname, container username, and current date/time |
| `/external` | Fetches a response from https://httpbin.org/get and returns it |

## How a change reaches the cluster

1. Merge a PR to `main`.
2. GitHub Actions builds the image, tagged with the commit SHA.
3. CI updates the tag in `k8s-stuff/apps/go-welcome-app/kustomization.yaml`.
4. Argo CD syncs it to the cluster.