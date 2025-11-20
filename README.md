# AWS Transit Gateway to Neo4j Graph Database

This project models AWS Transit Gateway (TGW) infrastructure in a Neo4j graph database and performs packet walks to trace network traffic flow between VPCs.

## Overview

The application demonstrates:

1. **AWS Data Modeling**: Represents AWS networking components (TGW, VPCs, Subnets, Route Tables, Routes, Attachments) as Go structs
2. **Graph Database Integration**: Loads AWS infrastructure into Neo4j as nodes and relationships
3. **Packet Walk Algorithm**: Traverses the graph to show step-by-step how packets flow through the network

## Architecture

### Mock AWS Environment

The mock environment includes:

- **1 Transit Gateway** with multiple route tables
- **4 VPCs**:
  - Production VPC (10.1.0.0/16)
  - Development VPC (10.2.0.0/16)
  - Shared Services VPC (10.3.0.0/16)
  - Firewall VPC (10.100.0.0/16) - Central inspection VPC

### Routing Scenarios

#### Scenario 1: Central Firewall VPC (Inspection VPC)

All spoke VPCs (Production, Development, Shared Services) route traffic through the central Firewall VPC for inspection before reaching the destination VPC.

**Traffic Flow**: `Spoke VPC → TGW → Firewall VPC → TGW → Destination VPC`

#### TGW Route Table Design

1. **Spoke Route Table**: Associated with all spoke VPC attachments
   - Default route (0.0.0.0/0) → Firewall VPC attachment

2. **Firewall Route Table**: Associated with Firewall VPC attachment
   - Specific routes to each spoke VPC

### Graph Model

#### Nodes

- `TransitGateway`: AWS Transit Gateway
- `TGWAttachment`: TGW attachment to VPC
- `TGWRouteTable`: TGW route table
- `TGWRoute`: Route in TGW route table
- `VPC`: Virtual Private Cloud
- `Subnet`: VPC subnet
- `VPCRouteTable`: VPC route table
- `VPCRoute`: Route in VPC route table

#### Relationships

- `HAS_ATTACHMENT`: TGW → TGWAttachment
- `HAS_ROUTE_TABLE`: TGW → TGWRouteTable, VPC → VPCRouteTable
- `HAS_ROUTE`: RouteTable → Route
- `HAS_SUBNET`: VPC → Subnet
- `USES_ROUTE_TABLE`: Attachment/Subnet → RouteTable
- `ATTACHED_VIA`: VPC → TGWAttachment

## Prerequisites

- Go 1.21 or later
- Neo4j 5.x (can be run via Docker)
- Docker and Docker Compose (optional, for running Neo4j)

## Setup

### 1. Start Neo4j

Using Docker Compose:

```bash
docker-compose up -d
```

Or manually with Docker:

```bash
docker run -d \
  --name neo4j \
  -p 7474:7474 -p 7687:7687 \
  -e NEO4J_AUTH=neo4j/password \
  neo4j:5.15
```

Access Neo4j Browser at: http://localhost:7474

### 2. Install Go Dependencies

```bash
go mod download
```

### 3. Configure Connection (Optional)

Set environment variables if using different Neo4j credentials:

```bash
export NEO4J_URI="bolt://localhost:7687"
export NEO4J_USER="neo4j"
export NEO4J_PASSWORD="password"
```

## Running the Application

```bash
go run main.go
```

The application will:

1. Connect to Neo4j
2. Load the mock AWS environment (TGW, VPCs, Subnets, Routes)
3. Populate the Neo4j graph database
4. Execute packet walk scenarios

## Packet Walk Scenarios

The application demonstrates several packet walk scenarios:

### Scenario 1: Production → Development (via Firewall)
- Source: Production VPC subnet (10.1.1.0/24)
- Destination: Development VPC subnet (10.2.1.0/24)
- Path: Prod VPC → TGW → Firewall VPC → TGW → Dev VPC

### Scenario 2: Development → Shared Services (via Firewall)
- Source: Development VPC subnet (10.2.1.0/24)
- Destination: Shared Services VPC subnet (10.3.1.0/24)
- Path: Dev VPC → TGW → Firewall VPC → TGW → Shared VPC

### Scenario 3: Production → Shared Services (via Firewall)
- Source: Production VPC subnet (10.1.1.0/24)
- Destination: Shared Services VPC subnet (10.3.1.0/24)
- Path: Prod VPC → TGW → Firewall VPC → TGW → Shared VPC

### Scenario 4: Shared Services → Production (Return Traffic)
- Source: Shared Services VPC subnet (10.3.1.0/24)
- Destination: Production VPC subnet (10.1.1.0/24)
- Path: Shared VPC → TGW → Firewall VPC → TGW → Prod VPC

## Example Output

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
  Description: Route table lookup: destination 10.2.1.0/24 matches route to tgw-0123456789abcdef0 (type: transit-gateway)

Step 3: Transit Gateway
  ID: tgw-0123456789abcdef0
  Action: Enter TGW
  Destination: 10.2.1.0/24
  Description: Packet enters Transit Gateway tgw-0123456789abcdef0

... (more steps) ...

✓ PACKET DELIVERED SUCCESSFULLY
```

## Project Structure

```
.
├── main.go                 # Main application entry point
├── models/
│   └── aws_models.go       # AWS data structures (TGW, VPC, Subnet, etc.)
├── aws/
│   └── mock_data.go        # Mock AWS environment data
├── neo4j/
│   ├── connection.go       # Neo4j client and connection management
│   └── loader.go           # Load AWS data into Neo4j
├── packetwalk/
│   └── walker.go           # Packet walk algorithm and graph traversal
├── go.mod                  # Go module dependencies
├── docker-compose.yml      # Docker Compose for Neo4j
└── README.md              # This file
```

## Querying the Graph

Once data is loaded, you can query the graph using Cypher in Neo4j Browser:

### View all nodes
```cypher
MATCH (n) RETURN n LIMIT 100
```

### View TGW and connected VPCs
```cypher
MATCH (tgw:TransitGateway)-[:HAS_ATTACHMENT]->(att:TGWAttachment)
MATCH (vpc:VPC)-[:ATTACHED_VIA]->(att)
RETURN tgw, att, vpc
```

### View routes in a specific VPC
```cypher
MATCH (vpc:VPC {name: 'production-vpc'})-[:HAS_ROUTE_TABLE]->(rt:VPCRouteTable)-[:HAS_ROUTE]->(r:VPCRoute)
RETURN vpc, rt, r
```

### View TGW routing
```cypher
MATCH (tgw:TransitGateway)-[:HAS_ROUTE_TABLE]->(rt:TGWRouteTable)-[:HAS_ROUTE]->(r:TGWRoute)
RETURN tgw, rt, r
```

## Adapting for Real AWS Environment

To use with a real AWS environment:

1. Replace `aws/mock_data.go` with real AWS SDK calls
2. Use AWS SDK v2 to query:
   - `ec2.DescribeTransitGateways`
   - `ec2.DescribeTransitGatewayAttachments`
   - `ec2.DescribeTransitGatewayRouteTables`
   - `ec2.SearchTransitGatewayRoutes`
   - `ec2.DescribeVpcs`
   - `ec2.DescribeSubnets`
   - `ec2.DescribeRouteTables`

3. Build the `models.AWSEnvironment` struct from the AWS API responses

Example:
```go
func GetRealAWSEnvironment(ctx context.Context) (*models.AWSEnvironment, error) {
    cfg, err := config.LoadDefaultConfig(ctx)
    if err != nil {
        return nil, err
    }

    ec2Client := ec2.NewFromConfig(cfg)

    // Query AWS resources...
    // Build environment struct...

    return env, nil
}
```

## License

MIT
