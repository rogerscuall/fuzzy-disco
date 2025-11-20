# Quick Start Guide

## Prerequisites

- Go 1.21+
- Docker and Docker Compose
- Git

## Quick Setup (5 minutes)

### 1. Clone and Navigate
```bash
git clone <repository-url>
cd fuzzy-disco
```

### 2. Start Neo4j
```bash
make docker-up
```

Wait for Neo4j to be ready (about 10 seconds). You should see:
```
Neo4j is ready!
Neo4j Browser: http://localhost:7474
Credentials: neo4j/password
```

### 3. Build and Run
```bash
make build
./tgw-neo4j
```

Or run directly:
```bash
make run
```

### 4. View Results

The application will:
1. Connect to Neo4j
2. Load mock AWS Transit Gateway environment
3. Display environment summary
4. Run 4 packet walk scenarios showing traffic flow between VPCs

**Example output:**
```
================================================================================
PACKET WALK: 10.1.1.0/24 -> 10.2.1.0/24
================================================================================

Step 1: Source Subnet
  ID: subnet-prod-app-1a
  Action: Packet originates
  Destination: 10.2.1.0/24
  Description: Packet starts in subnet prod-app-subnet-1a (10.1.1.0/24)

Step 2: VPC Route Table
  ID: rtb-prod-private
  Action: Route lookup
  Destination: 0.0.0.0/0
  Next Hop: tgw-0123456789abcdef0
  ...

✓ PACKET DELIVERED SUCCESSFULLY
```

### 5. Explore in Neo4j Browser

1. Open http://localhost:7474 in your browser
2. Login with:
   - Username: `neo4j`
   - Password: `password`

3. Run example queries:

**View the entire topology:**
```cypher
MATCH (n) RETURN n LIMIT 100
```

**View TGW and connected VPCs:**
```cypher
MATCH (tgw:TransitGateway)-[:HAS_ATTACHMENT]->(att:TGWAttachment)
MATCH (vpc:VPC)-[:ATTACHED_VIA]->(att)
RETURN tgw, att, vpc
```

**View all routing:**
```cypher
MATCH (rt)-[:HAS_ROUTE]->(r)
RETURN rt, r
```

More queries in: `neo4j/queries.cypher`

## Mock Environment

The application creates this network topology:

```
┌─────────────────┐
│  Production VPC │
│   10.1.0.0/16   │
└────────┬────────┘
         │
    ┌────┴──────────────────┐
    │                       │
    │  Transit Gateway      │
    │  tgw-0123456789...    │
    │                       │
    └────┬──────────────────┘
         │
    ┌────┴────────┐
    │             │
    ▼             ▼
┌─────────┐  ┌──────────┐
│Dev VPC  │  │Shared VPC│
│10.2.0.0 │  │10.3.0.0  │
└─────────┘  └──────────┘
         │             │
         └──────┬──────┘
                │
         ┌──────▼───────┐
         │ Firewall VPC │
         │ 10.100.0.0   │
         │ (Inspection) │
         └──────────────┘
```

### Routing Architecture

- **Spoke VPCs** (Prod, Dev, Shared): Route all traffic (0.0.0.0/0) through Firewall VPC
- **Firewall VPC**: Routes specific traffic back to each spoke
- **Traffic Flow**: Spoke → TGW → Firewall → TGW → Destination Spoke

## Scenarios Demonstrated

1. **Prod → Dev**: Through central firewall
2. **Dev → Shared**: Through central firewall
3. **Prod → Shared**: Through central firewall
4. **Shared → Prod**: Return traffic through firewall

## Common Commands

```bash
# Start Neo4j
make docker-up

# Stop Neo4j
make docker-down

# View Neo4j logs
make docker-logs

# Build application
make build

# Run application
make run

# Clean up
make clean
make docker-clean
```

## Environment Variables

Override defaults if needed:

```bash
export NEO4J_URI="bolt://localhost:7687"
export NEO4J_USER="neo4j"
export NEO4J_PASSWORD="password"
```

## Troubleshooting

### Neo4j connection failed
- Ensure Docker is running: `docker ps`
- Check Neo4j is running: `make docker-logs`
- Verify port 7687 is not in use: `netstat -an | grep 7687`

### Build errors
- Ensure Go 1.21+: `go version`
- Download dependencies: `make deps`

### Packet walk errors
- Check Neo4j has data loaded
- Run Cypher query: `MATCH (n) RETURN count(n)`
- Should return > 0 nodes

## Next Steps

1. **Explore the code**: Start with `main.go`
2. **Query the graph**: Use `neo4j/queries.cypher` examples
3. **Modify the topology**: Edit `aws/mock_data.go`
4. **Connect to real AWS**: See `aws/real_aws_example.go`

## Documentation

- Full README: `README.md`
- Cypher query examples: `neo4j/queries.cypher`
- Project structure details: See README.md

## Support

- Issues: Open a GitHub issue
- Questions: See README.md for detailed documentation
