# Inventory API

Real-time inventory management service for Walmart platform.

## Configuration

Required environment variables:

```bash
DB_HOST=inventory-db.walmart.internal
DB_USER=admin
DB_PASSWORD=<secure-password>
DB_NAME=inventory_prod
PORT=8080  # optional, defaults to 8080
```

## Running

```bash
export DB_HOST=inventory-db.walmart.internal
export DB_USER=admin
export DB_PASSWORD=your-password-here
export DB_NAME=inventory_prod

go run main.go
```

## API Endpoints

- `GET /health` - Health check
- `GET /ready` - Readiness check
- `GET /api/inventory` - List all products
- `GET /api/stock?sku=SKU-001` - Get stock for SKU
- `POST /api/reserve` - Reserve stock

## Security

- Database credentials must be provided via environment variables
- Never commit credentials to version control
- Use secrets management in production (Vault, AWS Secrets Manager, etc.)
