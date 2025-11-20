// Neo4j Cypher Query Examples for AWS TGW Graph

// ============================================================================
// Basic Queries
// ============================================================================

// View all nodes in the graph
MATCH (n)
RETURN n
LIMIT 100;

// Count nodes by type
MATCH (n)
RETURN labels(n) as NodeType, count(n) as Count
ORDER BY Count DESC;

// View all relationships
MATCH ()-[r]->()
RETURN type(r) as RelationshipType, count(r) as Count
ORDER BY Count DESC;

// ============================================================================
// Transit Gateway Queries
// ============================================================================

// View Transit Gateway with all its components
MATCH (tgw:TransitGateway)
OPTIONAL MATCH (tgw)-[:HAS_ATTACHMENT]->(att:TGWAttachment)
OPTIONAL MATCH (tgw)-[:HAS_ROUTE_TABLE]->(rt:TGWRouteTable)
RETURN tgw, att, rt;

// View TGW attachments and connected VPCs
MATCH (tgw:TransitGateway)-[:HAS_ATTACHMENT]->(att:TGWAttachment)
MATCH (vpc:VPC)-[:ATTACHED_VIA]->(att)
RETURN tgw.name as TGW, vpc.name as VPC, vpc.cidr as CIDR, att.id as AttachmentID;

// View all TGW route tables and their routes
MATCH (tgw:TransitGateway)-[:HAS_ROUTE_TABLE]->(rt:TGWRouteTable)
OPTIONAL MATCH (rt)-[:HAS_ROUTE]->(r:TGWRoute)
RETURN rt.name as RouteTable,
       rt.isDefault as IsDefault,
       collect({destination: r.destination, attachment: r.attachmentId, type: r.routeType}) as Routes;

// Find which TGW route table is used by a specific VPC
MATCH (vpc:VPC {name: 'production-vpc'})-[:ATTACHED_VIA]->(att:TGWAttachment)
MATCH (att)-[:USES_ROUTE_TABLE]->(rt:TGWRouteTable)
RETURN vpc.name as VPC, rt.name as RouteTable, rt.id as RouteTableID;

// ============================================================================
// VPC Queries
// ============================================================================

// View all VPCs with their subnets
MATCH (vpc:VPC)
OPTIONAL MATCH (vpc)-[:HAS_SUBNET]->(subnet:Subnet)
RETURN vpc.name as VPC,
       vpc.cidr as VPC_CIDR,
       collect({name: subnet.name, cidr: subnet.cidr, az: subnet.az}) as Subnets;

// View specific VPC with all components
MATCH (vpc:VPC {name: 'production-vpc'})
OPTIONAL MATCH (vpc)-[:HAS_SUBNET]->(subnet:Subnet)
OPTIONAL MATCH (vpc)-[:HAS_ROUTE_TABLE]->(rt:VPCRouteTable)
OPTIONAL MATCH (rt)-[:HAS_ROUTE]->(r:VPCRoute)
RETURN vpc, subnet, rt, r;

// View VPC routing tables and routes
MATCH (vpc:VPC)-[:HAS_ROUTE_TABLE]->(rt:VPCRouteTable)
OPTIONAL MATCH (rt)-[:HAS_ROUTE]->(r:VPCRoute)
RETURN vpc.name as VPC,
       rt.name as RouteTable,
       rt.isMain as IsMain,
       collect({destination: r.destination, target: r.target, type: r.targetType}) as Routes;

// Find which subnets use which route tables
MATCH (vpc:VPC {name: 'production-vpc'})-[:HAS_SUBNET]->(subnet:Subnet)
MATCH (subnet)-[:USES_ROUTE_TABLE]->(rt:VPCRouteTable)
RETURN subnet.name as Subnet,
       subnet.cidr as CIDR,
       rt.name as RouteTable;

// ============================================================================
// Routing Path Queries
// ============================================================================

// Find path from one VPC to another through TGW
MATCH path = (sourceVpc:VPC {name: 'production-vpc'})-[:ATTACHED_VIA]->
             (sourceAtt:TGWAttachment)<-[:HAS_ATTACHMENT]-
             (tgw:TransitGateway)-[:HAS_ATTACHMENT]->
             (destAtt:TGWAttachment)<-[:ATTACHED_VIA]-
             (destVpc:VPC {name: 'development-vpc'})
RETURN path;

// Find all VPCs connected to the same TGW
MATCH (tgw:TransitGateway)-[:HAS_ATTACHMENT]->(att:TGWAttachment)
MATCH (vpc:VPC)-[:ATTACHED_VIA]->(att)
RETURN tgw.name as TransitGateway,
       collect({name: vpc.name, cidr: vpc.cidr, attachment: att.id}) as ConnectedVPCs;

// View complete routing for a specific subnet CIDR
MATCH (subnet:Subnet {cidr: '10.1.1.0/24'})
MATCH (subnet)-[:USES_ROUTE_TABLE]->(rt:VPCRouteTable)
MATCH (rt)-[:HAS_ROUTE]->(r:VPCRoute)
RETURN subnet.name as Subnet,
       subnet.cidr as SubnetCIDR,
       rt.name as RouteTable,
       collect({destination: r.destination, target: r.target, type: r.targetType}) as Routes
ORDER BY r.destination;

// ============================================================================
// Firewall VPC Inspection Flow
// ============================================================================

// View spoke VPC to firewall VPC routing
MATCH (spoke:VPC)-[:ATTACHED_VIA]->(spokeAtt:TGWAttachment)
MATCH (spokeAtt)-[:USES_ROUTE_TABLE]->(spokeRT:TGWRouteTable)
MATCH (spokeRT)-[:HAS_ROUTE]->(route:TGWRoute)
MATCH (route.attachmentId)-[:HAS_ATTACHMENT]-(tgw:TransitGateway)-[:HAS_ATTACHMENT]->(fwAtt:TGWAttachment)
MATCH (fwAtt)<-[:ATTACHED_VIA]-(fw:VPC)
WHERE spoke.name <> 'firewall-vpc' AND fw.name = 'firewall-vpc'
RETURN spoke.name as SpokeVPC,
       spoke.cidr as SpokeCIDR,
       route.destination as RouteDestination,
       fw.name as FirewallVPC;

// View firewall VPC return routing to spokes
MATCH (fw:VPC {name: 'firewall-vpc'})-[:ATTACHED_VIA]->(fwAtt:TGWAttachment)
MATCH (fwAtt)-[:USES_ROUTE_TABLE]->(fwRT:TGWRouteTable)
MATCH (fwRT)-[:HAS_ROUTE]->(route:TGWRoute)
RETURN fw.name as FirewallVPC,
       fwRT.name as RouteTable,
       collect({destination: route.destination, attachment: route.attachmentId}) as RoutesToSpokes;

// ============================================================================
// Network Analysis Queries
// ============================================================================

// Find all possible paths between two subnets (limited depth)
MATCH path = (s1:Subnet {cidr: '10.1.1.0/24'})-[*..10]-(s2:Subnet {cidr: '10.2.1.0/24'})
RETURN path
LIMIT 5;

// Count how many VPCs are in each availability zone
MATCH (subnet:Subnet)
MATCH (vpc:VPC)-[:HAS_SUBNET]->(subnet)
RETURN vpc.name as VPC,
       count(DISTINCT subnet.az) as AvailabilityZones,
       collect(DISTINCT subnet.az) as AZs;

// Find VPCs without TGW attachments
MATCH (vpc:VPC)
WHERE NOT (vpc)-[:ATTACHED_VIA]->(:TGWAttachment)
RETURN vpc.name as VPC, vpc.cidr as CIDR;

// Find subnets that route to TGW
MATCH (subnet:Subnet)-[:USES_ROUTE_TABLE]->(rt:VPCRouteTable)
MATCH (rt)-[:HAS_ROUTE]->(r:VPCRoute)
WHERE r.targetType = 'transit-gateway'
RETURN subnet.name as Subnet,
       subnet.cidr as CIDR,
       r.destination as RouteDestination,
       r.target as TGW;

// ============================================================================
// Visualization Queries
// ============================================================================

// Visualize entire TGW topology
MATCH (tgw:TransitGateway)
OPTIONAL MATCH (tgw)-[r1:HAS_ATTACHMENT]->(att:TGWAttachment)
OPTIONAL MATCH (vpc:VPC)-[r2:ATTACHED_VIA]->(att)
OPTIONAL MATCH (tgw)-[r3:HAS_ROUTE_TABLE]->(tgwRt:TGWRouteTable)
OPTIONAL MATCH (vpc)-[r4:HAS_SUBNET]->(subnet:Subnet)
RETURN tgw, r1, att, r2, vpc, r3, tgwRt, r4, subnet;

// Visualize VPC internal topology
MATCH (vpc:VPC {name: 'production-vpc'})
MATCH (vpc)-[r1:HAS_SUBNET]->(subnet:Subnet)
MATCH (vpc)-[r2:HAS_ROUTE_TABLE]->(rt:VPCRouteTable)
MATCH (subnet)-[r3:USES_ROUTE_TABLE]->(rt)
MATCH (rt)-[r4:HAS_ROUTE]->(route:VPCRoute)
RETURN vpc, r1, subnet, r2, rt, r3, r4, route;

// Visualize routing between two specific VPCs
MATCH path = (v1:VPC {name: 'production-vpc'})-[*..8]-(v2:VPC {name: 'development-vpc'})
RETURN path
LIMIT 3;

// ============================================================================
// Cleanup Queries
// ============================================================================

// Delete all nodes and relationships (use with caution!)
// MATCH (n) DETACH DELETE n;

// Delete specific VPC and related components
// MATCH (vpc:VPC {name: 'test-vpc'})
// DETACH DELETE vpc;
