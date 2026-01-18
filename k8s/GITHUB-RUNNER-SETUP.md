# GitHub Actions Self-Hosted Runner Setup

This guide shows you how to set up a self-hosted GitHub Actions runner in your k3s cluster.

## Why Self-Hosted Runner?

✅ **No need to expose k3s API to the internet**
✅ **More secure** - Runner connects to GitHub (outbound only)
✅ **Faster deployments** - Direct access to cluster
✅ **No IP whitelist needed**
✅ **Free** - No GitHub Actions minutes consumed for deployment job

---

## Step 1: Create GitHub Personal Access Token (PAT)

You need a token so the runner can register with GitHub.

### Create the Token:

1. Go to: **https://github.com/settings/tokens/new**
2. **Note**: `GitHub Runner for evite`
3. **Expiration**: 90 days (or No expiration)
4. **Select scopes**:
   - ✅ `repo` (Full control of private repositories)
   - ✅ `workflow` (Update GitHub Action workflows)
5. Click **"Generate token"**
6. **⚠️ COPY THE TOKEN** - You won't see it again!

---

## Step 2: Create Kubernetes Secret with the Token

Run this command (replace `YOUR_GITHUB_TOKEN` with the token from Step 1):

```bash
kubectl create namespace github-runner

kubectl create secret generic github-runner-token \
  --namespace=github-runner \
  --from-literal=token="YOUR_GITHUB_TOKEN"
```

---

## Step 3: Deploy the GitHub Runner

```bash
# Deploy the runner
kubectl apply -f k8s/github-runner-namespace.yaml
kubectl apply -f k8s/github-runner-serviceaccount.yaml
kubectl apply -f k8s/github-runner-deployment.yaml

# Wait for runner to be ready
kubectl wait --for=condition=ready pod \
  -l app=github-runner \
  -n github-runner \
  --timeout=120s

# Check logs
kubectl logs -f deployment/github-runner -n github-runner
```

---

## Step 4: Verify Runner is Connected

1. Go to: **https://github.com/AlexTLDR/evite/settings/actions/runners**
2. You should see a runner named **"k3s-runner"** with status **"Idle"** (green)

---

## Step 5: Update GitHub Actions Workflow

The workflow has been updated to use `runs-on: [self-hosted, k3s]` instead of `ubuntu-latest`.

**Commit and push the changes:**

```bash
git add .github/workflows/deploy.yml k8s/github-runner-*.yaml
git commit -m "feat: add self-hosted GitHub Actions runner"
git push origin main
```

---

## Step 6: Remove KUBECONFIG Secret (No Longer Needed)

Since the runner is inside the cluster, you don't need the `KUBECONFIG` secret anymore.

1. Go to: **https://github.com/AlexTLDR/evite/settings/secrets/actions**
2. Delete the **`KUBECONFIG`** secret (optional, but recommended for security)

---

## Step 7: Test the Deployment

Push a change to trigger the workflow:

```bash
# Make a small change
echo "# Test" >> README.md
git add README.md
git commit -m "test: trigger self-hosted runner"
git push origin main
```

Watch the deployment at: **https://github.com/AlexTLDR/evite/actions**

---

## Troubleshooting

### Runner not showing up in GitHub

```bash
# Check runner logs
kubectl logs -f deployment/github-runner -n github-runner

# Check if pod is running
kubectl get pods -n github-runner
```

### Runner shows "Offline"

The token might be expired or invalid. Recreate the secret:

```bash
kubectl delete secret github-runner-token -n github-runner
kubectl create secret generic github-runner-token \
  --namespace=github-runner \
  --from-literal=token="YOUR_NEW_GITHUB_TOKEN"

# Restart the runner
kubectl rollout restart deployment/github-runner -n github-runner
```

### Deployment fails with permission errors

Check the service account has proper permissions:

```bash
kubectl get clusterrolebinding github-runner -o yaml
```

---

## Security Notes

✅ **Runner runs in isolated namespace** (`github-runner`)
✅ **Has cluster-admin permissions** (needed for deployments)
✅ **Only processes jobs from your repository**
✅ **Token is stored as Kubernetes secret**
⚠️ **Keep your GitHub token secure** - It has full repo access

---

## Scaling

To run multiple runners (for parallel jobs):

```bash
kubectl scale deployment github-runner --replicas=2 -n github-runner
```

---

## Cleanup

To remove the runner:

```bash
kubectl delete namespace github-runner
```

---

## Next Steps

After the runner is set up:
1. ✅ Runner appears in GitHub
2. ✅ Push code to trigger deployment
3. ✅ Deployment runs on self-hosted runner
4. ✅ No need to expose k3s API to internet

