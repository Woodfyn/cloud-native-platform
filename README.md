### `Mini Production Platform`

Само приложение максимально простое:

```text
Go API
 ├── POST /orders
 ├── GET /orders/:id
 ├── GET /health
 ├── GET /ready
 └── GET /metrics
```

А основной фокус вообще не на Go, а на инфраструктуре:

```text
GitHub
   │
   ▼
GitHub Actions
   │
   ├── tests
   ├── golangci-lint
   ├── docker build
   └── push image
          │
          ▼
        GHCR
          │
          ▼
       ArgoCD
          │
          ▼
     Kubernetes
    ┌──────────────┐
    │ Go API x3    │
    │ PostgreSQL   │
    │ Redis        │
    │ Prometheus   │
    │ Grafana      │
    │ Loki         │
    │ Jaeger/Tempo │
    └──────────────┘
```

Тут ты за один проект потрогаешь почти всё, что реально нужно DevOps-инженеру.

Я бы разбил обучение на этапы:

1. **Docker.** Напиши Go API + PostgreSQL, сделай нормальный multi-stage `Dockerfile`, non-root user, healthcheck, `.dockerignore`, docker-compose для локального запуска.

2. **Terraform.** Сначала можешь даже использовать Terraform Docker provider и поднимать локальные контейнеры через Terraform. Потом перейти на облако: network, cluster, database, buckets, IAM/service accounts.

3. **Kubernetes.** Задеплой приложение в `kind`, `k3d` или Docker Desktop Kubernetes. Сделай `Deployment`, `Service`, `ConfigMap`, `Secret`, readiness/liveness probes, resources requests/limits, `HorizontalPodAutoscaler`.

4. **Helm.** Убери голые YAML в Helm chart:

```text
charts/api/
├── Chart.yaml
├── values.yaml
├── values-dev.yaml
├── values-prod.yaml
└── templates/
    ├── deployment.yaml
    ├── service.yaml
    ├── ingress.yaml
    └── hpa.yaml
```

5. **CI/CD.** GitHub Actions должен тестировать приложение, собирать Docker image и публиковать его, например:

```text
ghcr.io/your-name/devops-lab:<git-sha>
```

Не делай сначала `kubectl apply` прямо из GitHub Actions. Лучше следующим этапом изучить GitOps.

6. **ArgoCD.** Сделай второй repository:

```text
app-repository
    Go source code
    Dockerfile

infra-repository
    terraform/
    helm/
    environments/
        dev/
        staging/
        prod/
```

CI собирает image → меняется image tag в infra repo → ArgoCD видит изменение → синхронизирует Kubernetes.

Вот тут уже начинается действительно интересный DevOps.

7. **Observability.** Добавь три столпа:

```text
Metrics → Prometheus → Grafana
Logs    → Loki       → Grafana
Traces  → OpenTelemetry → Tempo/Jaeger
```

Например, в Grafana сделай dashboard:

```text
Requests/sec
Latency p50/p95/p99
5xx rate
CPU
Memory
Running pods
DB connections
```

А потом специально ломай систему.

Например:

```bash
while true; do
    curl http://api/orders
done
```

Смотри, как растёт нагрузка, запускается HPA и Kubernetes создаёт дополнительные pods.

Ещё интереснее — сделать endpoint:

```text
GET /debug/slow
```

который иногда отвечает 2–5 секунд. Потом настроить alert:

```text
p95 latency > 1 sec for 5 minutes
```

И отправлять alert в Slack/Telegram.

После этого добавь **Chaos Engineering**: удаляй pod, убивай PostgreSQL connection, создавай CPU load, выпускай intentionally broken release и наблюдай, восстанавливается ли система.

Например:

```bash
kubectl delete pod api-7d8f5c9f4-x8zqp
```

И твоя задача — добиться, чтобы пользователь практически ничего не заметил.

---

Есть ещё один вариант, который я бы тебе особенно рекомендовал: сделать не просто приложение, а **маленькую internal developer platform**.

Чтобы разработчик мог написать:

```yaml
service:
  name: payment-service
  port: 8080
  replicas: 3

database:
  postgres: true

monitoring:
  enabled: true
```

а твоя платформа автоматически генерировала/создавала:

```text
Deployment
Service
Ingress
HPA
ServiceMonitor
Grafana dashboard
ArgoCD Application
```

То есть условный:

```bash
platform create service payment-service
```

Это очень хороший проект именно для backend-разработчика, потому что ты сможешь использовать Go для написания CLI/control-plane, но параллельно реально изучишь:

**Linux → Docker → networking → Terraform → Kubernetes → Helm → GitHub Actions → ArgoCD → Prometheus → Grafana → OpenTelemetry → cloud IAM → secrets → GitOps.**

Если выбирать **один** проект, я бы делал именно такой:

> **Production-like Kubernetes platform для нескольких Go microservices с Terraform + Helm + ArgoCD + GitHub Actions + Prometheus/Grafana + OpenTelemetry.**

Причём начинать с трёх микросервисов не надо. Сначала один тупой Go API. В этом pet project **90% сложности должно быть в инфраструктуре, а не в бизнес-логике**.

Ты уже знаешь backend, поэтому это даст намного больше, чем очередной CRUD.
