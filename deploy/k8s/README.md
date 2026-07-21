# Kubernetes 部署

该目录提供 `Deployment`、`Service`、`ConfigMap`、PVC 和 Kustomize 清单。容器以非 root、只读根文件系统运行；startup/liveness 使用 `/skoll/health`，readiness 使用 `/skoll/ready`。

## 准备密钥

数据库必须在集群内或可从集群访问。先通过集群密钥管理方案创建 `skoll-runtime`，不要提交 Secret YAML：

```powershell
kubectl create secret generic skoll-runtime `
  --from-literal=SKOLL_STORE_DSN='skoll:<password>@tcp(mysql.example:3306)/skoll?charset=utf8mb4&parseTime=True&loc=UTC' `
  --from-literal=SKOLL_SECURITY_JWT_SECRET='<at-least-32-random-characters>'
```

## 校验与部署

```powershell
kubectl kustomize deploy/k8s | Out-Null
kubectl apply -k deploy/k8s
kubectl rollout status deployment/skoll --timeout=120s
kubectl get pods -l app.kubernetes.io/name=skoll
kubectl logs deployment/skoll
```

发布前应在 overlay 中固定镜像 digest、StorageClass、资源配额、网络策略、Ingress 和 TLS。两个 PVC 分别保存 `/app/data` 和 `/app/plugins`；数据库、对象存储和日志仍需按实际基础设施单独备份。
