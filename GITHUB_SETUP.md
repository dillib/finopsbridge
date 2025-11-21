# GitHub Repository Setup Guide

## Required GitHub Permissions

### For Personal Access Token (PAT) or GitHub App

If you're using a **Personal Access Token** or **GitHub App** to connect:

**Minimum Required Scopes:**
- ✅ `repo` - Full control of private repositories
  - `repo:status` - Access commit status
  - `repo_deployment` - Access deployment status
  - `public_repo` - Access public repositories (if repo is public)

**Optional but Recommended:**
- ✅ `workflow` - Update GitHub Action workflows (if using CI/CD)
- ✅ `read:org` - Read org and team membership (if using GitHub Organizations)

### For GitHub CLI (`gh`)

If using GitHub CLI:
```bash
gh auth login
```
This will prompt for permissions interactively.

---

## Repository Setup Steps

### 1. Initialize Git Repository

```bash
# In the project root
git init
git branch -M main
```

### 2. Create .gitignore (Already exists, but verify)

Your `.gitignore` already covers:
- ✅ Environment files (`.env`, `.env.local`)
- ✅ Dependencies (`node_modules/`, `vendor/`)
- ✅ Build outputs (`.next/`, `dist/`, `build/`)
- ✅ IDE files (`.vscode/`, `.idea/`)
- ✅ OS files (`.DS_Store`, `Thumbs.db`)
- ✅ Logs and databases

### 3. Create GitHub Repository

**Option A: Via GitHub Website**
1. Go to https://github.com/new
2. Repository name: `finopsbridge` (or your preferred name)
3. Choose **Private** (recommended for production code with secrets)
4. **DO NOT** initialize with README, .gitignore, or license (we already have these)
5. Click "Create repository"

**Option B: Via GitHub CLI**
```bash
gh repo create finopsbridge --private --source=. --remote=origin --push
```

### 4. Add Remote and Push

```bash
# Add remote (replace YOUR_USERNAME with your GitHub username)
git remote add origin https://github.com/YOUR_USERNAME/finopsbridge.git

# Or using SSH (if you have SSH keys set up)
git remote add origin git@github.com:YOUR_USERNAME/finopsbridge.git

# Stage all files
git add .

# Create initial commit
git commit -m "Initial commit: FinOpsBridge platform"

# Push to GitHub
git push -u origin main
```

---

## 🔐 Secrets Management

### NEVER Commit These Files:

1. **Environment Variables:**
   - `api/.env`
   - `web/.env.local`
   - Any file containing:
     - `CLERK_SECRET_KEY`
     - `DATABASE_URL`
     - AWS/Azure/GCP credentials
     - API keys

2. **Credentials:**
   - AWS IAM role ARNs with sensitive info
   - Service principal secrets
   - Database connection strings

### Use GitHub Secrets for CI/CD

If setting up GitHub Actions, add secrets in:
**Settings → Secrets and variables → Actions**

Required secrets:
- `CLERK_SECRET_KEY`
- `DATABASE_URL`
- `AWS_ACCESS_KEY_ID` (if needed)
- `AWS_SECRET_ACCESS_KEY` (if needed)
- `NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY`
- `NEXT_PUBLIC_API_URL`

---

## 📋 Pre-Commit Checklist

Before your first commit, verify:

- [ ] `.env` files are in `.gitignore` ✅ (already done)
- [ ] No hardcoded secrets in code ✅
- [ ] `api/policies/*.rego` files are ignored ✅ (already done)
- [ ] `node_modules/` is ignored ✅ (already done)
- [ ] Build artifacts are ignored ✅ (already done)

### Verify What Will Be Committed

```bash
# Check what files will be committed
git status

# Preview what will be committed
git add -n .
```

---

## 🔒 Repository Security Settings

### Recommended Settings:

1. **Branch Protection Rules:**
   - Go to: Settings → Branches
   - Add rule for `main` branch:
     - ✅ Require pull request reviews
     - ✅ Require status checks to pass
     - ✅ Require branches to be up to date

2. **Security:**
   - Settings → Security → Code security and analysis
   - ✅ Enable Dependabot alerts
   - ✅ Enable Dependabot security updates
   - ✅ Enable Code scanning (if using GitHub Advanced Security)

3. **Secrets Scanning:**
   - GitHub automatically scans for exposed secrets
   - Will alert if secrets are detected in commits

---

## 🚀 CI/CD Integration (Optional)

### GitHub Actions Workflow Example

Create `.github/workflows/ci.yml`:

```yaml
name: CI

on:
  push:
    branches: [ main ]
  pull_request:
    branches: [ main ]

jobs:
  backend:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.23'
      - name: Build
        run: |
          cd api
          go mod download
          go build ./...

  frontend:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-node@v3
        with:
          node-version: '18'
      - name: Install and Build
        run: |
          cd web
          npm ci
          npm run build
```

---

## 📝 Environment Variables Template

Create example files (commit these, but not the actual `.env` files):

### `api/.env.example`
```env
DATABASE_URL=postgres://user:password@localhost:5432/finopsbridge?sslmode=disable
CLERK_SECRET_KEY=sk_test_...
OPA_DIR=./policies
ALLOWED_ORIGINS=http://localhost:3000
PORT=8080
AWS_REGION=us-east-1
```

### `web/.env.example`
```env
NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY=pk_test_...
CLERK_SECRET_KEY=sk_test_...
NEXT_PUBLIC_CLERK_SIGN_IN_URL=/sign-in
NEXT_PUBLIC_CLERK_SIGN_UP_URL=/sign-up
NEXT_PUBLIC_CLERK_AFTER_SIGN_IN_URL=/dashboard
NEXT_PUBLIC_CLERK_AFTER_SIGN_UP_URL=/dashboard
NEXT_PUBLIC_API_URL=http://localhost:8080
```

---

## ✅ Quick Start Commands

```bash
# 1. Initialize git (if not already done)
git init
git branch -M main

# 2. Add all files
git add .

# 3. Create initial commit
git commit -m "Initial commit: FinOpsBridge platform"

# 4. Add remote (replace YOUR_USERNAME)
git remote add origin https://github.com/YOUR_USERNAME/finopsbridge.git

# 5. Push to GitHub
git push -u origin main
```

---

## 🔍 Verify Setup

After pushing, verify:
1. ✅ All files are present on GitHub
2. ✅ No `.env` files are visible
3. ✅ No `node_modules/` directory
4. ✅ No build artifacts
5. ✅ README.md is visible

---

## 🆘 Troubleshooting

### "Permission denied" error:
- Check your GitHub credentials
- Verify PAT has `repo` scope
- Try using SSH instead of HTTPS

### "Large files" warning:
- GitHub has a 100MB file size limit
- Use Git LFS for large files if needed

### "Secrets detected" warning:
- GitHub will alert if secrets are found
- Remove secrets from commit history if needed:
  ```bash
  git filter-branch --force --index-filter \
    "git rm --cached --ignore-unmatch path/to/file" \
    --prune-empty --tag-name-filter cat -- --all
  ```

---

## 📚 Additional Resources

- [GitHub Authentication](https://docs.github.com/en/authentication)
- [Managing Secrets](https://docs.github.com/en/actions/security-guides/encrypted-secrets)
- [GitHub CLI](https://cli.github.com/)
- [Git LFS](https://git-lfs.github.com/)

