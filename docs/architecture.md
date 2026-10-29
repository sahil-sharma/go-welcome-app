# Architecture

```mermaid
flowchart LR
  user[Browser] --> gw[Istio gateway] --> svc[go-welcome-app Service] --> pod[Pods x2]
```

- Stateless; no database.
- Secrets: `secret/apps/go-welcome-app` in Vault.