# goth-ipam

A simple **IP Address Management (IPAM)** tool built with the **GOTH** stack:
- **G**o (net/http, Go 1.25+)
- **O**RM-free SQL with [pgx](https://github.com/jackc/pgx)
- **T**empl for server-side HTML rendering
- **H**TMX for dynamic UI without a JS framework

![Screenshot: Subnet list](docs/screenshot.png "Subnet list")

> [!WARNING]
> **This application has NO authentication or authorization.**
> It is intended for use in **private / internal networks only**.
> **DO NOT** expose it to the public internet without adding an authentication layer (e.g., reverse proxy with basic auth, VPN, etc.).

---

## Requirements

| Tool | Version |
|------|---------|
| Go | 1.25+ |
| Node.js | 24+ |
| Docker & Docker Compose | v2+ |

## Quick Start (Docker Compose)

```bash
# 1. Clone the repository
git clone https://github.com/ttani03/goth-ipam.git
cd goth-ipam

# 2. Create your local environment file
cp .env.example .env
# Edit .env and set a secure DATABASE_URL

# 3. Start the app
docker compose up -d

# 4. Open in browser
open http://localhost:8080
```

## Local Development

```bash
# Install tools (requires mise)
mise install

# Start PostgreSQL
docker compose up -d postgres

# Set database URL
export DATABASE_URL="postgres://ipam_user:your_password@localhost:5432/ipam_db"

# Run with hot-reload (air)
make run
```

## Build

```bash
make build
# Binary is output to ./bin/ipam
```

## Features

- **Subnet management** – Add/remove IPv4 subnets (CIDR notation)
- **IP tracking** – Automatically enumerate and track all host addresses within a subnet
- **IP allocation** – Assign a hostname to any available IP with one click
- **HTMX-powered UI** – No page reloads, no separate JS framework

## License

[MIT](LICENSE)
