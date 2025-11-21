# FinOpsBridge

Policy-governance-first cloud spend control platform.

## Architecture

- **Frontend**: Next.js 15 App Router + TypeScript + Tailwind CSS + shadcn/ui + TanStack Query + Zod
- **Backend**: Go 1.23 with Fiber framework + GORM + PostgreSQL
- **Policy Engine**: Open Policy Agent (OPA) with Go bindings
- **Auth**: Clerk (React + Go middleware)
- **Database**: PostgreSQL
- **Real-time**: WebSocket via Fiber WebSocket

## Project Structure

```
finopsbridge/
├── web/          # Next.js frontend
├── api/          # Go backend
└── README.md
```

## Prerequisites

- Node.js 18+
- Go 1.23+
- PostgreSQL 14+
- Clerk account (for authentication)

## Environment Variables

### Frontend (`web/.env.local`)

```env
NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY=pk_test_...
CLERK_SECRET_KEY=sk_test_...
NEXT_PUBLIC_CLERK_SIGN_IN_URL=/sign-in
NEXT_PUBLIC_CLERK_SIGN_UP_URL=/sign-up
NEXT_PUBLIC_CLERK_AFTER_SIGN_IN_URL=/dashboard
NEXT_PUBLIC_CLERK_AFTER_SIGN_UP_URL=/dashboard
NEXT_PUBLIC_API_URL=http://localhost:8080
```

### Backend (`api/.env`)

```env
DATABASE_URL=postgres://user:password@localhost:5432/finopsbridge?sslmode=disable
CLERK_SECRET_KEY=sk_test_...
OPA_DIR=./policies
ALLOWED_ORIGINS=http://localhost:3000
PORT=8080
AWS_REGION=us-east-1
```

## Local Development

### 1. Setup Database

```bash
createdb finopsbridge
```

### 2. Start Backend

```bash
cd api
go mod download
go run main.go
```

The API will be available at `http://localhost:8080`

### 3. Start Frontend

```bash
cd web
npm install
npm run dev
```

The frontend will be available at `http://localhost:3000`

### 4. Seed Data

```bash
cd api
go run scripts/seed.go
```

This creates 3 example policies:
1. Max Monthly Spend - Production ($5000 limit)
2. Block X-Large Instances
3. Auto-Stop Idle Resources (24 hours)

## Deployment

### Deploy Backend to Fly.io

1. Install Fly CLI: `curl -L https://fly.io/install.sh | sh`

2. Login: `fly auth login`

3. Create app: `fly apps create finopsbridge-api`

4. Set secrets:
```bash
fly secrets set DATABASE_URL=postgres://...
fly secrets set CLERK_SECRET_KEY=sk_...
fly secrets set ALLOWED_ORIGINS=https://your-domain.vercel.app
```

5. Deploy:
```bash
cd api
fly deploy
```

### Deploy Frontend to Vercel

1. Install Vercel CLI: `npm i -g vercel`

2. Deploy:
```bash
cd web
vercel
```

3. Set environment variables in Vercel dashboard:
   - `NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY`
   - `CLERK_SECRET_KEY`
   - `NEXT_PUBLIC_API_URL` (your Fly.io API URL)

## Features

### ✅ Implemented

- [x] Marketing site with waitlist
- [x] Clerk authentication (sign up/login/orgs)
- [x] Dashboard with connected clouds, spend, and active policies
- [x] No-code Policy Builder (React form)
- [x] Policy engine with OPA integration
- [x] Enforcement worker (runs every 5 minutes)
- [x] Activity log
- [x] Settings page for cloud provider connections
- [x] Seed data with 3 example policies

### Policy Types Supported

1. **Max Monthly Spend**: Limit spending per account/project
2. **Block Instance Type**: Prevent deployment of oversized instances
3. **Auto-Stop Idle**: Automatically stop resources idle for X hours
4. **Require Tags**: Enforce mandatory tags on resources

### Cloud Provider Integrations

- **AWS**: Cost Explorer API, EC2 instance management
- **Azure**: Cost Management API (placeholder)
- **GCP**: Billing API (placeholder)

## API Endpoints

### Public
- `POST /api/waitlist` - Join waitlist

### Authenticated (requires Clerk token)
- `GET /api/dashboard/stats` - Get dashboard statistics
- `GET /api/policies` - List policies
- `POST /api/policies` - Create policy
- `PATCH /api/policies/:id` - Update policy
- `DELETE /api/policies/:id` - Delete policy
- `GET /api/cloud-providers` - List cloud providers
- `POST /api/cloud-providers` - Connect cloud provider
- `GET /api/activity` - List activity logs
- `GET /api/webhooks` - List webhooks
- `POST /api/webhooks` - Create webhook

## Enforcement Worker

The enforcement worker runs every 5 minutes and:

1. Fetches billing data from connected cloud providers
2. Evaluates all enabled policies using OPA
3. Creates violations when policies are breached
4. Automatically remediates violations (stops/terminates resources)
5. Sends webhook notifications

## Webhook Integrations

Supported webhook types:
- Slack
- Discord
- Microsoft Teams

Configure webhooks in the Settings page.

## License

MIT

