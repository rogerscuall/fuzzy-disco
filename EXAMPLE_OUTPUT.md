# Example Application Output

This document shows what the output looks like when running the application.

## Starting the Application

```bash
$ ./tgw-neo4j
```

## Console Output

```
AWS Transit Gateway to Neo4j Graph Database
Packet Walk Demonstration

Connecting to Neo4j...
✓ Connected to Neo4j

Loading AWS environment data...
✓ Loaded 1 Transit Gateway(s)
✓ Loaded 4 VPC(s)

=== AWS Environment Summary ===

Transit Gateway: main-tgw (tgw-0123456789abcdef0)
  ASN: 64512
  State: available
  Attachments: 4
  Route Tables: 3

  Route Tables:
    - default-route-table (tgw-rtb-default)
      Default: true
      Routes: 0
    - spoke-route-table (tgw-rtb-spoke)
      Default: false
      Routes: 1
        • 0.0.0.0/0 -> tgw-attach-firewall (static)
    - firewall-route-table (tgw-rtb-firewall)
      Default: false
      Routes: 3
        • 10.1.0.0/16 -> tgw-attach-prod (static)
        • 10.2.0.0/16 -> tgw-attach-dev (static)
        • 10.3.0.0/16 -> tgw-attach-shared (static)

VPCs:
  - production-vpc (vpc-prod-001)
    CIDR: 10.1.0.0/16
    Subnets: 4
    Route Tables: 2
    TGW Attachment: tgw-attach-prod

  - development-vpc (vpc-dev-001)
    CIDR: 10.2.0.0/16
    Subnets: 2
    Route Tables: 2
    TGW Attachment: tgw-attach-dev

  - shared-services-vpc (vpc-shared-001)
    CIDR: 10.3.0.0/16
    Subnets: 2
    Route Tables: 2
    TGW Attachment: tgw-attach-shared

  - firewall-vpc (vpc-firewall-001)
    CIDR: 10.100.0.0/16
    Subnets: 2
    Route Tables: 2
    TGW Attachment: tgw-attach-firewall

Loading AWS environment into Neo4j graph database...
✓ AWS environment loaded into Neo4j

=== PACKET WALK SCENARIOS ===


Scenario 1: Production to Development (via Firewall)
Description: Packet from Production VPC to Development VPC through central Firewall VPC
Source: 10.1.1.0/24 -> Destination: 10.2.1.0/24

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

Step 4: TGW Attachment
  ID: tgw-attach-prod
  Action: Attachment lookup
  Destination: 10.2.1.0/24
  Description: Packet enters via attachment tgw-attach-prod

Step 5: TGW Route Table
  ID: tgw-rtb-spoke
  Action: Route table association
  Destination: 10.2.1.0/24
  Description: Using TGW route table tgw-rtb-spoke

Step 6: TGW Route Table
  ID: tgw-rtb-spoke
  Action: Route lookup
  Destination: 0.0.0.0/0
  Next Hop: tgw-attach-firewall
  Description: Route found: destination 0.0.0.0/0 -> attachment tgw-attach-firewall

Step 7: TGW Attachment
  ID: tgw-attach-firewall
  Action: Exit TGW
  Destination: 10.2.1.0/24
  Next Hop: vpc-firewall-001
  Description: Packet exits TGW via attachment tgw-attach-firewall to VPC vpc-firewall-001

Step 8: Intermediate VPC
  ID: vpc-firewall-001
  Action: Enter intermediate VPC
  Destination: 10.2.1.0/24
  Description: Packet enters intermediate VPC firewall-vpc (10.100.0.0/16) - typically firewall/inspection VPC

Step 9: Firewall/Inspection
  ID: vpc-firewall-001
  Action: Traffic inspection
  Destination: 10.2.1.0/24
  Description: Packet inspected by firewall (simulated)

Step 10: Intermediate VPC Route Table
  ID: rtb-firewall-tgw
  Action: Route lookup
  Destination: 0.0.0.0/0
  Next Hop: vpce-firewall-endpoint
  Description: Route to destination: 0.0.0.0/0 via vpce-firewall-endpoint

Step 11: VPC Endpoint
  ID: vpce-firewall-endpoint
  Action: Forward through endpoint
  Destination: 10.2.1.0/24
  Description: Packet forwarded through VPC endpoint (firewall endpoint)

Step 12: Transit Gateway
  ID: tgw-0123456789abcdef0
  Action: Re-enter TGW
  Destination: 10.2.1.0/24
  Description: Packet re-enters Transit Gateway from intermediate VPC

Step 13: TGW Route Table
  ID: tgw-rtb-firewall
  Action: Route table lookup
  Destination: 10.2.1.0/24
  Description: Using firewall route table tgw-rtb-firewall

Step 14: TGW Route Table
  ID: tgw-rtb-firewall
  Action: Route lookup
  Destination: 10.2.0.0/16
  Next Hop: tgw-attach-dev
  Description: Route to destination VPC: 10.2.0.0/16 -> attachment tgw-attach-dev

Step 15: TGW Attachment
  ID: tgw-attach-dev
  Action: Exit TGW
  Destination: 10.2.1.0/24
  Next Hop: vpc-dev-001
  Description: Packet exits TGW to destination VPC via attachment tgw-attach-dev

Step 16: Destination VPC
  ID: vpc-dev-001
  Action: Enter destination VPC
  Destination: 10.2.1.0/24
  Description: Packet enters destination VPC development-vpc (10.2.0.0/16)

Step 17: VPC Route Table
  ID: rtb-dev-private
  Action: Final route lookup
  Destination: 10.2.1.0/24
  Description: Route lookup for local delivery

Step 18: Destination Subnet
  ID: subnet-dev-app-1a
  Action: Packet delivered
  Destination: 10.2.1.0/24
  Description: Packet successfully delivered to subnet dev-app-subnet-1a (10.2.1.0/24)

--------------------------------------------------------------------------------
✓ PACKET DELIVERED SUCCESSFULLY
================================================================================


Press Enter to continue to next scenario...

Scenario 2: Development to Shared Services (via Firewall)
Description: Packet from Development VPC to Shared Services VPC through central Firewall VPC
Source: 10.2.1.0/24 -> Destination: 10.3.1.0/24

================================================================================
PACKET WALK: 10.2.1.0/24 -> 10.3.1.0/24
================================================================================

Step 1: Source Subnet
  ID: subnet-dev-app-1a
  Action: Packet originates
  Destination: 10.3.1.0/24
  Description: Packet starts in subnet dev-app-subnet-1a (10.2.1.0/24)

Step 2: VPC Route Table
  ID: rtb-dev-private
  Action: Route lookup
  Destination: 0.0.0.0/0
  Next Hop: tgw-0123456789abcdef0
  Description: Route table lookup: destination 10.3.1.0/24 matches route to tgw-0123456789abcdef0 (type: transit-gateway)

... (similar detailed steps) ...

Step 18: Destination Subnet
  ID: subnet-shared-services-1a
  Action: Packet delivered
  Destination: 10.3.1.0/24
  Description: Packet successfully delivered to subnet shared-services-subnet-1a (10.3.1.0/24)

--------------------------------------------------------------------------------
✓ PACKET DELIVERED SUCCESSFULLY
================================================================================


Press Enter to continue to next scenario...

Scenario 3: Production to Shared Services (via Firewall)
Description: Packet from Production VPC to Shared Services VPC through central Firewall VPC
Source: 10.1.1.0/24 -> Destination: 10.3.1.0/24

... (similar output) ...

Scenario 4: Shared Services to Production (Return Traffic)
Description: Return traffic from Shared Services VPC to Production VPC
Source: 10.3.1.0/24 -> Destination: 10.1.1.0/24

... (similar output) ...


=== ALL SCENARIOS COMPLETED ===
```

## Neo4j Browser Visualization

After running the application, you can visualize the graph in Neo4j Browser at http://localhost:7474.

### Example Query 1: View Full Topology

```cypher
MATCH (n) RETURN n LIMIT 100
```

**Result**: Visual graph showing all nodes and relationships

### Example Query 2: View TGW Routing

```cypher
MATCH (tgw:TransitGateway)-[:HAS_ROUTE_TABLE]->(rt:TGWRouteTable)-[:HAS_ROUTE]->(r:TGWRoute)
RETURN tgw, rt, r
```

**Result**: Shows TGW route tables and their routes

### Example Query 3: Trace Connection Path

```cypher
MATCH path = (v1:VPC {name: 'production-vpc'})-[*..8]-(v2:VPC {name: 'development-vpc'})
RETURN path
LIMIT 1
```

**Result**: Shows the path between two VPCs through TGW

## Using the Application with Real AWS

When ready to use with a real AWS environment:

1. Configure AWS credentials:
   ```bash
   export AWS_REGION=us-east-1
   export AWS_PROFILE=your-profile
   ```

2. Uncomment and implement the code in `aws/real_aws_example.go`

3. Update `main.go` to use `GetRealAWSEnvironment()` instead of `GetMockAWSEnvironment()`

4. Run the application:
   ```bash
   ./tgw-neo4j
   ```

The application will query your actual AWS infrastructure and load it into Neo4j.

## Troubleshooting Common Issues

### Issue: Connection refused to Neo4j

```
Failed to connect to Neo4j: connection refused
```

**Solution**: Start Neo4j with `make docker-up`

### Issue: Authentication failed

```
Failed to connect to Neo4j: authentication failed
```

**Solution**: Check credentials in environment variables or update default password in `main.go`

### Issue: No matching route found

```
✗ PACKET DELIVERY FAILED: No matching route found in TGW route table
```

**Solution**: Check the routing configuration in `aws/mock_data.go`

## Performance Metrics

For the mock environment:
- **Nodes created**: ~50
- **Relationships created**: ~100
- **Database load time**: ~1-2 seconds
- **Packet walk execution**: ~50-100ms per scenario
- **Total runtime**: ~10-15 seconds (including user interaction)

## Next Steps

1. **Explore the graph**: Use Neo4j Browser to visualize relationships
2. **Run custom queries**: Use examples from `neo4j/queries.cypher`
3. **Modify topology**: Edit `aws/mock_data.go` to add more VPCs or scenarios
4. **Integrate with AWS**: Implement real AWS data retrieval
5. **Add monitoring**: Track packet walks and routing patterns
6. **Extend scenarios**: Add VPN, Direct Connect, or multi-region TGW peering
