# Kubernetes Deployment for Evite

This directory contains Kubernetes manifests for deploying the Evite application to a k3s cluster.

## Architecture

- **PostgreSQL StatefulSet**: Single instance with persistent storage (10Gi)
- **Application Deployment**: 3 replicas for high availability
- **Ingress**: Traefik ingress with TLS support

## Prerequisites

1. **k3s cluster** running
2. **kubectl** configured to access your cluster
3. **Docker image** built and pushed to a registry (GitHub Container Registry, Docker Hub, etc.)
4. **Domain name** configured with DNS pointing to your cluster

## ⚠️ Security Best Practices

**IMPORTANT**: The YAML files in this directory contain **placeholder values only**.

**DO NOT** edit these files with real secrets and commit them to Git!

### Recommended Workflow:

**Option 1: Use production files (gitignored)**

```bash
# Copy templates to production files (these are gitignored)
cp k8s/app-secret.yaml k8s/app-secret.prod.yaml
cp k8s/app-configmap.yaml k8s/app-configmap.prod.yaml
cp k8s/postgres-secret.yaml k8s/postgres-secret.prod.yaml

# Edit the .prod.yaml files with your actual values
nano k8s/app-secret.prod.yaml
nano k8s/app-configmap.prod.yaml
nano k8s/postgres-secret.prod.yaml
```

**Option 2: Create secrets directly with kubectl (most secure)**

```bash
# Create namespace first
kubectl apply -f k8s/namespace.yaml

# Create app secret from command line (secrets never touch filesystem)
kubectl create secret generic evite-secret \
  --namespace=evite \
  --from-literal=GOOGLE_CLIENT_ID="your-actual-client-id" \
  --from-literal=GOOGLE_CLIENT_SECRET="your-actual-client-secret" \
  --from-literal=SESSION_SECRET="$(openssl rand -base64 32)" \
  --from-literal=DATABASE_URL="postgres://evite:your-password@postgres-0.postgres.evite.svc.cluster.local:5432/evite?sslmode=disable"

# Create postgres secret
kubectl create secret generic postgres-secret \
  --namespace=evite \
  --from-literal=POSTGRES_USER="evite" \
  --from-literal=POSTGRES_PASSWORD="your-strong-password" \
  --from-literal=POSTGRES_DB="evite"

# Create configmap
kubectl create configmap evite-config \
  --namespace=evite \
  --from-literal=ADMIN_EMAILS="your-email@example.com" \
  --from-literal=EVENT_DATE="2026-04-19T14:00:00+03:00" \
  --from-literal=RSVP_DEADLINE="2026-04-12T23:59:59+03:00" \
  --from-literal=CHURCH_NAME="Biserica Apărătorii Patriei I" \
  --from-literal=CHURCH_ADDRESS="Your Church Address" \
  --from-literal=RESTAURANT_NAME="Noor By The Pool" \
  --from-literal=RESTAURANT_ADDRESS="Your Restaurant Address" \
  --from-literal=BASE_URL="https://your-actual-domain.com" \
  --from-literal=GOOGLE_REDIRECT_URL="https://your-actual-domain.com/auth/google/callback" \
  --from-literal=PORT="8080"
```

## Configuration Steps

### 1. Update Secrets (if using Option 1 above)

Edit `app-secret.prod.yaml` and replace:
- `GOOGLE_CLIENT_ID`: Your Google OAuth client ID
- `GOOGLE_CLIENT_SECRET`: Your Google OAuth client secret
- `SESSION_SECRET`: A random string (generate with `openssl rand -base64 32`)

Edit `postgres-secret.prod.yaml` and replace:
- `POSTGRES_PASSWORD`: A strong password for production

### 2. Update ConfigMap (if using Option 1 above)

Edit `app-configmap.prod.yaml` and replace:
- `ADMIN_EMAILS`: Comma-separated list of admin email addresses
- `BASE_URL`: Your actual domain (e.g., `https://evite.yourdomain.com`)
- `GOOGLE_REDIRECT_URL`: Your OAuth callback URL
- `CHURCH_NAME`, `CHURCH_ADDRESS`, `RESTAURANT_NAME`, `RESTAURANT_ADDRESS`: Your event details
- `EVENT_DATE`, `RSVP_DEADLINE`: Your event dates

### 3. Update Ingress

Edit `ingress.yaml` and replace:
- `evite.yourdomain.com`: Your actual domain name (appears in 2 places)

### 4. Update Deployment Image

Edit `app-deployment.yaml` and replace:
- `image: ghcr.io/alextldr/evite:latest`: Your actual Docker image location

## Deployment

### If using Option 1 (production YAML files):

```bash
# 1. Create namespace
kubectl apply -f k8s/namespace.yaml

# 2. Create secrets (use .prod versions)
kubectl apply -f k8s/postgres-secret.prod.yaml
kubectl apply -f k8s/app-secret.prod.yaml

# 3. Create ConfigMap (use .prod version)
kubectl apply -f k8s/app-configmap.prod.yaml

# 4. Deploy PostgreSQL
kubectl apply -f k8s/postgres-pvc.yaml
kubectl apply -f k8s/postgres-statefulset.yaml
kubectl apply -f k8s/postgres-service.yaml

# Wait for PostgreSQL to be ready
kubectl wait --for=condition=ready pod -l app=postgres -n evite --timeout=300s

# 5. Deploy Application
kubectl apply -f k8s/app-deployment.yaml
kubectl apply -f k8s/app-service.yaml

# 6. Deploy Ingress
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

