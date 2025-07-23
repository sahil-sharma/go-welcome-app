# Welcome App (Golang + Kubernetes)

A simple Golang web service with 3 endpoints, containerized with Docker, and deployed via Kubernetes.

---

## 🌐 Endpoints

| Endpoint     | Description                               |
|--------------|-------------------------------------------|
| `/`          | Returns a simple "Hello" message          |
| `/welcome`   | Returns hostname, container username, and current date/time |
| `/external`  | Fetches a response from `https://httpbin.org/get` and returns it |

---

## Build and Run with Docker
```bash
docker build -t welcome-app .
docker run -p 8080:8080 welcome-app
```