# Architecture Documentation

## System Overview

This application models AWS Transit Gateway infrastructure as a graph database in Neo4j and implements a packet walk algorithm to trace network traffic flow.

## High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                     Application Layer                           │
│                        (main.go)                                 │
└────────────┬───────────────────────────────────┬────────────────┘
             │                                   │
             ▼                                   ▼
┌────────────────────────┐          ┌──────────────────────────┐
│   AWS Data Provider    │          │   Neo4j Graph Database   │
│  (aws/mock_data.go)    │          │    (Port 7687/7474)      │
└────────────┬───────────┘          └──────────┬───────────────┘
             │                                  │
             ▼                                  ▼
┌────────────────────────┐          ┌──────────────────────────┐
│   Data Models          │          │   Graph Loader           │
│ (models/aws_models.go) │◄────────►│  (neo4j/loader.go)       │
└────────────────────────┘          └──────────────────────────┘
                                               │
                                               ▼
                                    ┌──────────────────────────┐
                                    │   Packet Walk Engine     │
                                    │ (packetwalk/walker.go)   │
                                    └──────────────────────────┘
```

## Network Topology

### Mock AWS Environment

```
                          ┌─────────────────────────────┐
                          │   Transit Gateway (TGW)     │
                          │   tgw-0123456789abcdef0     │
                          │   ASN: 64512                │
                          └──────────┬──────────────────┘
                                     │
                    ┌────────────────┼────────────────┐
                    │                │                │
                    ▼                ▼                ▼
        ┌─────────────────┐  ┌─────────────┐  ┌─────────────────┐
        │  Production VPC │  │  Dev VPC    │  │ Shared Svc VPC  │
        │  10.1.0.0/16    │  │ 10.2.0.0/16 │  │  10.3.0.0/16    │
        └─────────────────┘  └─────────────┘  └─────────────────┘
                    │                │                │
                    │   All traffic routes through   │
                    │      Firewall VPC for          │
                    │         inspection             │
                    └────────────────┼────────────────┘
                                     ▼
                          ┌─────────────────────────┐
                          │   Firewall VPC          │
                          │   10.100.0.0/16         │
                          │   (Central Inspection)  │
                          └─────────────────────────┘
```

### TGW Route Table Design

#### 1. Spoke Route Table
- **Associated with**: Production, Development, Shared Services VPC attachments
- **Routes**:
  - `0.0.0.0/0` → Firewall VPC attachment

#### 2. Firewall Route Table
- **Associated with**: Firewall VPC attachment
- **Routes**:
  - `10.1.0.0/16` → Production VPC attachment
  - `10.2.0.0/16` → Development VPC attachment
  - `10.3.0.0/16` → Shared Services VPC attachment

## Traffic Flow Patterns

### Pattern 1: Spoke-to-Spoke Communication

```
Source VPC (Prod)
    │
    │ 1. Route lookup: 0.0.0.0/0 → TGW
    │
    ▼
Transit Gateway
    │
    │ 2. Spoke RT: 0.0.0.0/0 → Firewall attachment
    │
    ▼
Firewall VPC
    │
    │ 3. Inspection/filtering
    │ 4. Route lookup: 10.2.0.0/16 → TGW
    │
    ▼
Transit Gateway
    │
    │ 5. Firewall RT: 10.2.0.0/16 → Dev attachment
    │
    ▼
Destination VPC (Dev)
    │
    │ 6. Local routing to destination subnet
    │
    ▼
Destination Subnet
```

## Graph Database Schema

### Node Types

1. **TransitGateway**
   - Properties: `id`, `name`, `asn`, `state`

2. **TGWAttachment**
   - Properties: `id`, `resourceType`, `resourceId`, `state`, `routeTableId`

3. **TGWRouteTable**
   - Properties: `id`, `name`, `isDefault`

4. **TGWRoute**
   - Properties: `destination`, `attachmentId`, `routeType`, `state`

5. **VPC**
   - Properties: `id`, `name`, `cidr`, `tgwAttachmentId`

6. **Subnet**
   - Properties: `id`, `cidr`, `az`, `name`, `routeTableId`

7. **VPCRouteTable**
   - Properties: `id`, `name`, `isMain`

8. **VPCRoute**
   - Properties: `destination`, `target`, `targetType`

### Relationship Types

```
(TGW)-[:HAS_ATTACHMENT]->(TGWAttachment)
(TGW)-[:HAS_ROUTE_TABLE]->(TGWRouteTable)
(TGWRouteTable)-[:HAS_ROUTE]->(TGWRoute)
(TGWAttachment)-[:USES_ROUTE_TABLE]->(TGWRouteTable)
(VPC)-[:ATTACHED_VIA]->(TGWAttachment)
(VPC)-[:HAS_SUBNET]->(Subnet)
(VPC)-[:HAS_ROUTE_TABLE]->(VPCRouteTable)
(VPCRouteTable)-[:HAS_ROUTE]->(VPCRoute)
(Subnet)-[:USES_ROUTE_TABLE]->(VPCRouteTable)
```

### Graph Visualization

```
     ┌────────────────┐
     │ TransitGateway │
     └───┬────────┬───┘
         │        │
   [:HAS_ATTACHMENT] [:HAS_ROUTE_TABLE]
         │        │
         ▼        ▼
  ┌─────────┐  ┌──────────────┐
  │TGWAttach│  │TGWRouteTable │
  └────┬────┘  └──────┬───────┘
       │              │
 [:ATTACHED_VIA] [:HAS_ROUTE]
       │              │
       ▼              ▼
   ┌──────┐      ┌─────────┐
   │ VPC  │      │TGWRoute │
   └──┬───┘      └─────────┘
      │
[:HAS_SUBNET]
      │
      ▼
 ┌────────┐
 │Subnet  │
 └────────┘
```

## Packet Walk Algorithm

### Algorithm Steps

1. **Find Source Subnet**: Locate subnet node by CIDR
2. **Source VPC Routing**: Query VPC route table for matching route
3. **Enter TGW**: If route target is TGW, enter transit gateway
4. **TGW Attachment Lookup**: Find source VPC's TGW attachment
5. **TGW Route Table Association**: Find route table associated with attachment
6. **TGW Routing Decision**: Query TGW route table for matching route
7. **Intermediate VPC Check**: Determine if exiting to destination or intermediate VPC
8. **Firewall Inspection** (if applicable): Process through firewall VPC
9. **Return to TGW** (if firewall): Re-enter TGW with firewall route table
10. **Exit to Destination**: Route to destination VPC attachment
11. **Destination VPC Routing**: Query destination VPC route table
12. **Final Delivery**: Deliver to destination subnet

### Route Matching Logic

```go
// Pseudo-code
func findMatchingRoute(routeTableID, destCIDR):
    routes = queryRoutes(routeTableID)

    for route in routes:
        if route.destination == "0.0.0.0/0":
            defaultRoute = route

        if routeContains(route.destination, destCIDR):
            return route  // Most specific match

    return defaultRoute  // Fallback to default route
```

## Data Flow

### Loading Data into Neo4j

```
1. GetMockAWSEnvironment()
   └─> Returns *AWSEnvironment struct

2. LoadAWSEnvironment(env)
   ├─> ClearDatabase()
   ├─> For each TransitGateway:
   │   ├─> CREATE (tgw:TransitGateway)
   │   ├─> For each RouteTable:
   │   │   ├─> CREATE (rt:TGWRouteTable)
   │   │   └─> CREATE (tgw)-[:HAS_ROUTE_TABLE]->(rt)
   │   └─> For each Attachment:
   │       ├─> CREATE (att:TGWAttachment)
   │       └─> CREATE (tgw)-[:HAS_ATTACHMENT]->(att)
   └─> For each VPC:
       ├─> CREATE (vpc:VPC)
       ├─> For each Subnet:
       │   ├─> CREATE (s:Subnet)
       │   └─> CREATE (vpc)-[:HAS_SUBNET]->(s)
       └─> For each RouteTable:
           ├─> CREATE (rt:VPCRouteTable)
           └─> CREATE (vpc)-[:HAS_ROUTE_TABLE]->(rt)
```

### Packet Walk Execution

```
1. WalkFromSubnetToSubnet(sourceCIDR, destCIDR)
   ├─> findSubnetByCIDR(sourceCIDR)
   ├─> findSubnetByCIDR(destCIDR)
   ├─> findMatchingRoute(sourceSubnet.routeTableId)
   ├─> If route.targetType == "transit-gateway":
   │   ├─> walkThroughTGW()
   │   │   ├─> findAttachmentByVPC(sourceVPC)
   │   │   ├─> findMatchingRoute(tgwRouteTable)
   │   │   └─> If intermediate VPC:
   │   │       └─> walkThroughIntermediateVPC()
   │   │           ├─> Process firewall inspection
   │   │           └─> Re-enter TGW
   │   └─> finalVPCRouting()
   └─> Return PacketWalk with steps
```

## Extension Points

### Adding Real AWS Integration

Replace `aws/mock_data.go` with AWS SDK v2 calls:

```go
// Example structure
func GetRealAWSEnvironment(ctx context.Context, region string) (*models.AWSEnvironment, error) {
    cfg, _ := config.LoadDefaultConfig(ctx, config.WithRegion(region))
    ec2Client := ec2.NewFromConfig(cfg)

    // Query AWS
    tgws := queryTransitGateways(ctx, ec2Client)
    vpcs := queryVPCs(ctx, ec2Client)

    return &models.AWSEnvironment{
        TransitGateways: tgws,
        VPCs: vpcs,
    }, nil
}
```

### Adding Additional Scenarios

1. **VPN Connections**: Add VPN attachments
2. **Direct Connect**: Add DX Gateway attachments
3. **VPC Peering**: Add peering connections
4. **Multiple TGWs**: Add inter-TGW peering
5. **Advanced Routing**: Implement route prioritization

## Performance Considerations

### Graph Queries

- Indexed properties: `id`, `cidr`
- Use `MATCH` with specific labels for better performance
- Limit relationship traversal depth in path queries

### Scalability

- Current implementation: ~100 nodes, ~200 relationships
- Production scale: 1000s of nodes possible
- Consider batching for large environments
- Use Neo4j Enterprise for clustering

## Testing Strategy

### Unit Tests

- `models/aws_models_test.go`: Data structure validation
- `aws/mock_data_test.go`: Mock environment verification

### Integration Tests

- Requires running Neo4j instance
- Test data loading and querying
- Test packet walk scenarios

### Test Coverage

```bash
go test ./... -cover
```

## Deployment

### Local Development

```bash
make docker-up    # Start Neo4j
make build        # Build application
./tgw-neo4j      # Run
```

### Production Deployment

1. Deploy Neo4j cluster
2. Configure AWS credentials
3. Build application: `go build -o tgw-neo4j`
4. Set environment variables
5. Run with systemd or container orchestration

## Monitoring

### Neo4j Metrics

- Node count: `MATCH (n) RETURN count(n)`
- Relationship count: `MATCH ()-[r]->() RETURN count(r)`
- Query performance: Enable Neo4j query logging

### Application Metrics

- Packet walk execution time
- Neo4j connection pool status
- Error rates

## Security Considerations

1. **Neo4j Authentication**: Always use strong passwords
2. **AWS Credentials**: Use IAM roles, not access keys
3. **Network Security**: Restrict Neo4j port access
4. **Data Sensitivity**: Consider encryption at rest
5. **Audit Logging**: Enable Neo4j query logging

## References

- AWS Transit Gateway: https://docs.aws.amazon.com/vpc/latest/tgw/
- Neo4j Graph Database: https://neo4j.com/docs/
- Go Neo4j Driver: https://github.com/neo4j/neo4j-go-driver
- AWS SDK for Go v2: https://aws.github.io/aws-sdk-go-v2/
