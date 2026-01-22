# Setup Summary - External Secrets with AWS Parameter Store

## What We've Done

✅ **Created 17 parameters in AWS Parameter Store** (under `/evite/` prefix)
✅ **Created IAM user** (`external-secrets-evite`) with read-only access to `/evite/*` parameters
✅ **Created Kubernetes manifests** for External Secrets Operator integration
✅ **Updated Dockerfile** to run as non-root user `alex`
✅ **Updated .gitignore** to prevent committing secrets
✅ **Deployed self-hosted GitHub Actions runner** in k3s cluster (no need to expose k3s API to internet)

---

## Architecture Overview

```text
┌─────────────────────────────────────────────────────────────┐
│                         GitHub                               │
│  - Code push triggers workflow                               │
│  - Sends job to self-hosted runner                           │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│           Self-Hosted Runner (in k3s cluster)                │
│  - Runs in github-runner namespace                           │
│  - Polls GitHub for jobs                                     │
│  - Executes deployment commands locally                      │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│                      AWS Parameter Store                     │
│  /evite/google-client-id, /evite/session-secret, etc.       │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       │ IAM User: external-secrets-evite
                       │ (Access Key ID + Secret Access Key)
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│              External Secrets Operator (k3s)                 │
│  - Polls AWS every 15 minutes                                │
│  - Creates/updates Kubernetes Secrets                        │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│                  Kubernetes Secrets (k3s)                    │
│  - evite-secret (app secrets)                                │
│  - postgres-secret (database credentials)                    │
│  - evite-config (configuration)                              │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│                   Application Pods                           │
│  Environment variables injected from secrets                 │
└─────────────────────────────────────────────────────────────┘
```

---

## Files Created

### External Secrets Manifests (NEW - USED IN PRODUCTION)

- `k8s/secretstore.yaml` - Connects to AWS Parameter Store
- `k8s/externalsecret-app.yaml` - Pulls app secrets from AWS
- `k8s/externalsecret-postgres.yaml` - Pulls PostgreSQL secrets from AWS
- `k8s/externalsecret-config.yaml` - Pulls configuration from AWS

### Self-Hosted GitHub Actions Runner (DEPLOYED IN K3S)

- `k8s/github-runner-namespace.yaml` - Creates `github-runner` namespace
- `k8s/github-runner-serviceaccount.yaml` - Service account with cluster-admin permissions
- `k8s/github-runner-deployment.yaml` - Runner pod deployment
- **Runner is running in your k3s cluster** (no need to expose k3s API to internet)
- **Runner name**: `k3s-runner`
- **Status**: Check at https://github.com/AlexTLDR/evite/settings/actions/runners

### Documentation
- `k8s/DEPLOYMENT-GUIDE.md` - Step-by-step deployment instructions
- `k8s/SETUP-SUMMARY.md` - This file
- `k8s/GITHUB-RUNNER-SETUP.md` - GitHub Actions runner setup guide

### Application Manifests
- `k8s/namespace.yaml` - Creates `evite` namespace
- `k8s/postgres-pvc.yaml` - PostgreSQL persistent storage
- `k8s/postgres-statefulset.yaml` - PostgreSQL deployment
- `k8s/postgres-service.yaml` - PostgreSQL service
- `k8s/app-deployment.yaml` - Application deployment
- `k8s/app-service.yaml` - Application service
- `k8s/ingress.yaml` - Traefik ingress for HTTPS

---

## AWS Parameters Created

### Secrets (SecureString - Encrypted)
1. `/evite/google-client-id`
2. `/evite/google-client-secret`
3. `/evite/session-secret`
4. `/evite/postgres-password`
5. `/evite/database-url`

### Configuration (String - Not Encrypted)
6. `/evite/postgres-user`
7. `/evite/postgres-db`
8. `/evite/admin-emails`
9. `/evite/base-url` → `https://invitatie.dotsat.work`
10. `/evite/google-redirect-url` → `https://invitatie.dotsat.work/auth/google/callback`
11. `/evite/event-date`
12. `/evite/rsvp-deadline`
13. `/evite/church-name`
14. `/evite/church-address`
15. `/evite/restaurant-name`
16. `/evite/restaurant-address`
17. `/evite/port`

---

## Security Benefits

✅ **No secrets in Git** - All sensitive data in AWS Parameter Store
✅ **Encrypted at rest** - AWS encrypts SecureString parameters with KMS
✅ **Audit trail** - AWS CloudTrail logs all parameter access
✅ **IAM access control** - Fine-grained permissions via IAM policies
✅ **Easy rotation** - Update in AWS, pods restart automatically
✅ **Separation of concerns** - Secrets managed separately from code
✅ **Non-root containers** - App runs as user `alex` (UID 1000)

---

## Cost

**FREE** ✅ - Using AWS Parameter Store Standard tier (up to 10,000 parameters free)

---

## Next Steps

1. **Create IAM user in AWS** (if not done yet)
   - See `DEPLOYMENT-GUIDE.md` Step 2

2. **Install External Secrets Operator on k3s**
   - See `DEPLOYMENT-GUIDE.md` Step 1

3. **Deploy to k3s**
   - Follow `DEPLOYMENT-GUIDE.md` from Step 3 onwards

4. **Build and push Docker image**
   - Build: `docker build -t ghcr.io/alextldr/evite:latest .`
   - Push: `docker push ghcr.io/alextldr/evite:latest`

5. **Update Google OAuth settings**
   - Add redirect URI: `https://invitatie.dotsat.work/auth/google/callback`

---

## Updating Configuration

To change any value (e.g., RSVP deadline):

```bash
# 1. Update in AWS
aws ssm put-parameter \
  --name "/evite/rsvp-deadline" \
  --value "2026-05-01T23:59:59+03:00" \
  --overwrite

# 2. Force sync (optional - otherwise waits 15 minutes)
kubectl annotate externalsecret evite-config force-sync="$(date +%s)" -n evite

# 3. Restart pods to pick up new values
kubectl rollout restart deployment/evite -n evite
```

---

## Troubleshooting

See `DEPLOYMENT-GUIDE.md` for detailed troubleshooting steps.

Quick checks:
```bash
# Check External Secrets status
kubectl get externalsecrets -n evite

# Check if secrets were created
kubectl get secrets -n evite

# Check External Secrets Operator logs
kubectl logs -n external-secrets-system -l app.kubernetes.io/name=external-secrets

# Check app logs
kubectl logs -f deployment/evite -n evite
```

---

## References

- External Secrets Operator: https://external-secrets.io/
- AWS Parameter Store: https://docs.aws.amazon.com/systems-manager/latest/userguide/systems-manager-parameter-store.html
- k3s Documentation: https://docs.k3s.io/

