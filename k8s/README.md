# Kubernetes Deployment for Evite

This directory contains Kubernetes manifests for deploying the Evite application to a k3s cluster.

## Architecture

- **PostgreSQL StatefulSet**: Single instance with persistent storage (10Gi)
- **Application Deployment**: 3 replicas for high availability
- **Ingress Controller**: Traefik with automatic HTTPS via Let's Encrypt
- **Ingress**: Standard Kubernetes Ingress with Traefik ingress controller

## Prerequisites

1. **k3s cluster** running
2. **kubectl** configured to access your cluster
3. **Docker image** built and pushed to a registry (GitHub Container Registry)
4. **Domain name** configured with DNS pointing to your cluster
5. **External Secrets Operator** installed in the cluster
6. **AWS Parameter Store** configured with all required secrets
7. **AWS IAM credentials** for External Secrets Operator

## Secret Management

This deployment uses **External Secrets Operator** to pull secrets from **AWS Parameter Store**.

Secrets are **NOT** stored in Git or Kubernetes manifests. Instead:
- All secrets are stored in AWS Parameter Store under `/evite/` prefix
- External Secrets Operator syncs them to Kubernetes secrets automatically
- See `DEPLOYMENT-GUIDE.md` for detailed setup instructions

## Quick Deployment

**For detailed step-by-step instructions, see `DEPLOYMENT-GUIDE.md`**

```bash
# 1. Create namespace
kubectl apply -f k8s/namespace.yaml

# 2. Install External Secrets Operator (if not already installed)
helm repo add external-secrets https://charts.external-secrets.io
helm repo update
helm install external-secrets \
  external-secrets/external-secrets \
  -n external-secrets-system \
  --create-namespace \
  --set installCRDs=true

# 3. Create AWS credentials secret
kubectl create secret generic aws-credentials \
  --namespace=evite \
  --from-literal=access-key-id="YOUR_AWS_ACCESS_KEY_ID" \
  --from-literal=secret-access-key="YOUR_AWS_SECRET_ACCESS_KEY"

# 4. Deploy External Secrets
kubectl apply -f k8s/secretstore.yaml
kubectl apply -f k8s/externalsecret-app.yaml
kubectl apply -f k8s/externalsecret-postgres.yaml
kubectl apply -f k8s/externalsecret-config.yaml

# 5. Deploy PostgreSQL
kubectl apply -f k8s/postgres-pvc.yaml
kubectl apply -f k8s/postgres-statefulset.yaml
kubectl apply -f k8s/postgres-service.yaml

# 6. Deploy Application
kubectl apply -f k8s/app-deployment.yaml
kubectl apply -f k8s/app-service.yaml
kubectl apply -f k8s/ingress.yaml
```

## Building and Pushing Docker Image

### GitHub Container Registry (ghcr.io)

```bash
# Build the image
docker build -t ghcr.io/alextldr/evite:latest .

# Login to GitHub Container Registry
echo $GITHUB_TOKEN | docker login ghcr.io -u alextldr --password-stdin

# Push the image
docker push ghcr.io/alextldr/evite:latest
```

### Docker Hub

```bash
# Build the image
docker build -t alextldr/evite:latest .

# Login to Docker Hub
docker login

# Push the image
docker push alextldr/evite:latest
```

## Verification

```bash
# Check all resources
kubectl get all -n evite

# Check pods
kubectl get pods -n evite

# Check logs
kubectl logs -f deployment/evite -n evite

# Check PostgreSQL
kubectl logs -f statefulset/postgres -n evite

# Check ingress
kubectl get ingress -n evite
```

## Troubleshooting

### Pods not starting

```bash
# Describe the pod
kubectl describe pod <pod-name> -n evite

# Check events
kubectl get events -n evite --sort-by='.lastTimestamp'
```

### Database connection issues

```bash
# Check PostgreSQL is running
kubectl exec -it postgres-0 -n evite -- psql -U evite -d evite -c "SELECT 1;"

# Check database URL in app
kubectl exec -it deployment/evite -n evite -- env | grep DATABASE_URL
```

### Ingress not working

```bash
# Check Traefik is running
kubectl get pods -n kube-system | grep traefik

# Check ingress details
kubectl describe ingress evite-ingress -n evite
```

## Scaling

```bash
# Scale application replicas
kubectl scale deployment evite --replicas=5 -n evite

# Check status
kubectl get deployment evite -n evite
```

## Updates

```bash
# Update the image
kubectl set image deployment/evite evite=ghcr.io/alextldr/evite:v1.1.0 -n evite

# Or edit the deployment
kubectl edit deployment evite -n evite

# Rollout status
kubectl rollout status deployment/evite -n evite

# Rollback if needed
kubectl rollout undo deployment/evite -n evite
```

## Cleanup

```bash
# Delete all resources
kubectl delete -f k8s/

# Or delete namespace (removes everything)
kubectl delete namespace evite
```

