# Agentic Factory - Web Control Panel

A comprehensive React-based web UI for managing the Agentic Software Factory control plane.

## Features

✨ **Dashboard**
- System overview with real-time stats
- Tenant, agent, service, and credential counts
- System health monitoring
- Recent activity feed

👥 **Tenant Management**
- Create, view, and delete tenants
- Monitor tenant status and quotas
- Namespace management
- Tenant-specific configurations

🤖 **Agent Management**
- Register and manage agents
- View agent status and capabilities
- Assign credentials
- Track agent activity

🔗 **Service Discovery**
- View registered services
- Monitor service health
- Track service instances
- Cross-tenant service visibility

🔐 **Credential Management**
- Issue and revoke credentials
- View credential details
- Track credential lifecycle
- Monitor expiration and status

📋 **Policy Management**
- Define access control policies
- Manage capability grants
- Configure tenant policies
- Policy validation

📊 **Monitoring & Logs**
- View system logs
- Track API calls
- Monitor performance metrics
- Audit trail

## Tech Stack

- **Frontend:** React 18 + TypeScript
- **Styling:** Tailwind CSS
- **State:** Zustand
- **HTTP:** Axios
- **Build:** Vite
- **Routing:** React Router

## Development

### Prerequisites

- Node.js 18+
- npm or yarn

### Setup

```bash
cd web
npm install
```

### Development Server

```bash
npm run dev
```

Runs on http://localhost:3000 with hot reload enabled.

The app will proxy API calls to http://localhost:8080/api.

### Build

```bash
npm run build
```

Outputs to `dist/` directory, ready for production deployment.

### Linting

```bash
npm run lint
```

## Project Structure

```
web/
├── src/
│   ├── api/
│   │   └── client.ts           # API client with type definitions
│   ├── components/
│   │   ├── Layout.tsx          # Main layout with navigation
│   │   ├── StatCard.tsx        # Stat card component
│   │   └── ActivityList.tsx    # Activity feed component
│   ├── pages/
│   │   ├── Dashboard.tsx       # Overview dashboard
│   │   ├── TenantsList.tsx     # Tenants page
│   │   ├── TenantDetail.tsx    # Tenant details
│   │   ├── Agents.tsx          # Agents management
│   │   ├── Services.tsx        # Services discovery
│   │   ├── Credentials.tsx     # Credentials management
│   │   ├── Policies.tsx        # Policy management
│   │   └── Logs.tsx            # System logs
│   ├── store/
│   │   └── index.ts            # Zustand state management
│   ├── App.tsx                 # Main app component with routing
│   ├── main.tsx                # React root entry
│   └── index.css               # Tailwind CSS
├── index.html                  # HTML entry point
├── vite.config.ts              # Vite configuration
├── tailwind.config.js          # Tailwind configuration
├── postcss.config.js           # PostCSS configuration
├── tsconfig.json               # TypeScript configuration
└── package.json                # Dependencies
```

## API Integration

The UI connects to the Go backend via REST API at `/api`. The `FactoryClient` in `src/api/client.ts` handles all API communication with:

- Automatic request tracing (X-Request-ID headers)
- Error handling
- Type-safe responses
- 10-second timeout

### Available Endpoints

- `GET /api/health` - System health check
- `GET /api/stats` - System statistics
- `GET /api/tenants` - List all tenants
- `POST /api/tenants` - Create tenant
- `GET /api/tenants/:id` - Get tenant details
- `PUT /api/tenants/:id` - Update tenant
- `DELETE /api/tenants/:id` - Delete tenant
- `GET /api/tenants/:id/agents` - List tenant agents
- `GET /api/tenants/:id/services` - List tenant services
- `GET /api/tenants/:id/policies` - Get tenant policies
- `POST /api/tenants/:id/policies` - Create policy
- `GET /api/agents/:id/credentials` - List agent credentials
- `POST /api/credentials/:id/revoke` - Revoke credential
- `PATCH /api/services/:id/health` - Update service health

## State Management

Using Zustand for simple, predictable state management:

```typescript
import { useStore } from './store';

const tenants = useStore((state) => state.tenants);
const loadTenants = useStore((state) => state.loadTenants);
```

All async operations automatically set loading and error states.

## Deployment

### Docker

```dockerfile
# In web/Dockerfile
FROM node:18-alpine as builder
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM node:18-alpine
WORKDIR /app
RUN npm install -g serve
COPY --from=builder /app/dist ./dist
CMD ["serve", "-s", "dist", "-l", "3000"]
```

Build and run:
```bash
docker build -t factory-ui .
docker run -p 3000:3000 factory-ui
```

### Kubernetes

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: factory-ui
spec:
  replicas: 2
  selector:
    matchLabels:
      app: factory-ui
  template:
    metadata:
      labels:
        app: factory-ui
    spec:
      containers:
      - name: ui
        image: factory-ui:latest
        ports:
        - containerPort: 3000
        env:
        - name: REACT_APP_API_URL
          value: http://factory-api:8080/api
```

## Features in Development

- [ ] Real-time WebSocket updates
- [ ] Advanced filtering and search
- [ ] Bulk operations
- [ ] Custom dashboards
- [ ] Dark mode support
- [ ] Advanced analytics
- [ ] API documentation browser
- [ ] Webhook management

## Troubleshooting

### API Connection Issues

If the UI can't connect to the backend:

1. Verify backend is running: `curl http://localhost:8080/api/health`
2. Check CORS configuration on backend
3. Verify proxy in `vite.config.ts` points to correct backend URL

### Build Issues

```bash
# Clear node modules and reinstall
rm -rf node_modules package-lock.json
npm install

# Clear Vite cache
rm -rf .vite
```

## Contributing

When adding new pages:

1. Create component in `src/pages/`
2. Add route in `App.tsx`
3. Add navigation link in `src/components/Layout.tsx`
4. Use Zustand store for state management
5. Use FactoryClient for API calls

## License

Part of the Agentic Software Factory project.
