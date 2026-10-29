# Runbook

## Pods not ready
Kubernetes tab → pod status and events.
- `ImagePullBackOff`: the tag in k8s-stuff does not exist on Docker Hub; check the last CI run.
- `CrashLoopBackOff`: open the container logs in the Kubernetes tab.

## Deploy looks stuck
Argo CD card: `OutOfSync` means the latest commit isn't applied yet; `Degraded` means the new pods are failing.

## Roll back
Revert the last commit in `k8s-stuff/apps/go-welcome-app`. Argo CD deploys the previous tag.

## Who to ask
go-app-team (see the owner on this page).