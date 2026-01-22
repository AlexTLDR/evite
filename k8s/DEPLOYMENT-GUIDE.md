# Deployment Guide - External Secrets with AWS Parameter Store

This guide walks you through deploying the Evite application to k3s using External Secrets Operator and AWS Parameter Store.

## Prerequisites

✅ All parameters created in AWS Parameter Store (under `/evite/` prefix)
✅ IAM user `external-secrets-evite` created with appropriate permissions
✅ Access Key ID and Secret Access Key saved
✅ k3s cluster running and accessible via `kubectl`
✅ Helm installed
✅ Docker image built and pushed to registry

---

## Step 1: Install External Secrets Operator

```bash
# Add helm repo
helm repo add external-secrets https://charts.external-secrets.io
helm repo update

# Install External Secrets Operator
helm install external-secrets \
  external-secrets/external-secrets \
  -n external-secrets-system \
  --create-namespace \
  --set installCRDs=true

# Verify installation
kubectl get pods -n external-secrets-system
```

Wait until all pods are `Running`.

---

## Step 2: Create Namespace

```bash
kubectl apply -f k8s/namespace.yaml
```

---

## Step 3: Create AWS Credentials Secret

**⚠️ IMPORTANT: Do this manually - DO NOT commit this file to Git!**

```bash
# Create the secret with your AWS credentials
kubectl create secret generic aws-credentials \
  --namespace=evite \
  --from-literal=access-key-id="YOUR_AWS_ACCESS_KEY_ID" \
  --from-literal=secret-access-key="YOUR_AWS_SECRET_ACCESS_KEY"
```

Replace `YOUR_AWS_ACCESS_KEY_ID` and `YOUR_AWS_SECRET_ACCESS_KEY` with the actual values from Step 2 (IAM user creation).

---

## Step 4: Update AWS Region (if needed)

If your AWS region is NOT `eu-central-1`, edit `k8s/secretstore.yaml`:

```bash
nano k8s/secretstore.yaml
```

Change the `region` field to your AWS region (e.g., `us-east-1`, `us-west-2`, etc.).

---

## Step 5: Deploy SecretStore

```bash
kubectl apply -f k8s/secretstore.yaml

# Verify it's ready
kubectl get secretstore -n evite
```

Expected output:
```
NAME                   AGE   STATUS   READY
aws-parameter-store    10s   Valid    True
```

---

## Step 6: Deploy External Secrets

```bash
# Deploy all external secrets
kubectl apply -f k8s/externalsecret-app.yaml
kubectl apply -f k8s/externalsecret-postgres.yaml
kubectl apply -f k8s/externalsecret-config.yaml

# Verify they're synced
kubectl get externalsecrets -n evite
```

Expected output:
```
NAME                    STORE                 REFRESH INTERVAL   STATUS         READY
evite-app-secret        aws-parameter-store   15m                SecretSynced   True
evite-postgres-secret   aws-parameter-store   15m                SecretSynced   True
evite-config            aws-parameter-store   15m                SecretSynced   True
```

---

## Step 7: Verify Secrets Were Created

```bash
# Check that Kubernetes secrets were created
kubectl get secrets -n evite

# You should see:
# - evite-secret (from externalsecret-app.yaml)
# - postgres-secret (from externalsecret-postgres.yaml)
# - evite-config (from externalsecret-config.yaml)
```

---

## Step 8: Update Ingress Domain

Edit `k8s/ingress.yaml` and replace `evite.yourdomain.com` with `invitatie.dotsat.work`:

```bash
nano k8s/ingress.yaml
```

Replace both occurrences of `evite.yourdomain.com` with `invitatie.dotsat.work`.

---

## Step 9: Update Docker Image

Edit `k8s/app-deployment.yaml` and update the image location:

```bash
nano k8s/app-deployment.yaml
```

Change `image: ghcr.io/alextldr/evite:latest` to your actual Docker image location.

---

## Step 10: Deploy PostgreSQL

```bash
kubectl apply -f k8s/postgres-pvc.yaml
kubectl apply -f k8s/postgres-statefulset.yaml
kubectl apply -f k8s/postgres-service.yaml

# Wait for PostgreSQL to be ready
kubectl wait --for=condition=ready pod -l app=postgres -n evite --timeout=300s
```

---

## Step 11: Deploy Application

```bash
kubectl apply -f k8s/app-deployment.yaml
kubectl apply -f k8s/app-service.yaml

# Wait for app to be ready
kubectl wait --for=condition=ready pod -l app=evite -n evite --timeout=300s
```

---

## Step 12: Deploy Ingress

```bash
kubectl apply -f k8s/ingress.yaml
```

---

## Step 13: Verify Deployment

```bash
# Check all resources
kubectl get all -n evite

# Check Traefik
kubectl get pods -n kube-system | grep traefik
kubectl logs -n kube-system -l app.kubernetes.io/name=traefik

# Check app logs
kubectl logs -f deployment/evite -n evite

# Check ingress
kubectl get ingress -n evite
kubectl describe ingress evite-ingress -n evite
```

---

## Step 14: Access Your Application

Open your browser and go to: **https://invitatie.dotsat.work**

Traefik will automatically obtain a Let's Encrypt certificate via cert-manager.

---

## Updating Configuration

To update any configuration value:

1. **Update in AWS Parameter Store**:
   ```bash
   aws ssm put-parameter \
     --name "/evite/rsvp-deadline" \
     --value "2026-05-01T23:59:59+03:00" \
     --overwrite
   ```

2. **Wait for sync** (up to 15 minutes) or force immediate sync:
   ```bash
   kubectl annotate externalsecret evite-config force-sync="$(date +%s)" -n evite
   ```

3. **Restart pods** to pick up new values:
   ```bash
   kubectl rollout restart deployment/evite -n evite
   ```

---

## Troubleshooting

### External Secrets not syncing

```bash
# Check External Secret status
kubectl describe externalsecret evite-app-secret -n evite

# Check External Secrets Operator logs
kubectl logs -n external-secrets-system -l app.kubernetes.io/name=external-secrets
```

### Pods not starting

```bash
# Check pod status
kubectl get pods -n evite

# Check pod logs
kubectl logs -l app=evite -n evite

# Describe pod for events
kubectl describe pod -l app=evite -n evite
```

### Database connection issues

```bash
# Check PostgreSQL logs
kubectl logs -l app=postgres -n evite

# Test database connectivity from app pod
kubectl exec -it deployment/evite -n evite -- sh
# Inside pod:
nc -zv postgres-0.postgres.evite.svc.cluster.local 5432
```

---

## Cleanup

To remove everything:

```bash
kubectl delete namespace evite
```

This will delete all resources in the `evite` namespace.

