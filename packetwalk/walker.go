package packetwalk

import (
	"fmt"
	"net"
	"strings"

	"github.com/rogerscuall/fuzzy-disco/neo4j"
)

// Step represents a single step in the packet walk
type Step struct {
	StepNumber  int
	Location    string
	LocationID  string
	Action      string
	Destination string
	NextHop     string
	Description string
}

// PacketWalk represents the complete path of a packet
type PacketWalk struct {
	SourceSubnet      string
	DestinationSubnet string
	Steps             []Step
	Success           bool
	FailureReason     string
}

// Walker performs packet walks through the network graph
type Walker struct {
	client *neo4j.Neo4jClient
}

// NewWalker creates a new packet walker
func NewWalker(client *neo4j.Neo4jClient) *Walker {
	return &Walker{client: client}
}

// WalkFromSubnetToSubnet performs a packet walk from source subnet CIDR to destination subnet CIDR
func (w *Walker) WalkFromSubnetToSubnet(sourceCIDR, destCIDR string) (*PacketWalk, error) {
	walk := &PacketWalk{
		SourceSubnet:      sourceCIDR,
		DestinationSubnet: destCIDR,
		Steps:             []Step{},
		Success:           false,
	}

	// Find source subnet
	sourceSubnet, err := w.findSubnetByCIDR(sourceCIDR)
	if err != nil {
		walk.FailureReason = fmt.Sprintf("Source subnet not found: %s", err)
		return walk, nil
	}

	// Find destination subnet
	destSubnet, err := w.findSubnetByCIDR(destCIDR)
	if err != nil {
		walk.FailureReason = fmt.Sprintf("Destination subnet not found: %s", err)
		return walk, nil
	}

	stepNum := 1

	// Step 1: Starting from source subnet
	walk.Steps = append(walk.Steps, Step{
		StepNumber:  stepNum,
		Location:    "Source Subnet",
		LocationID:  sourceSubnet["id"].(string),
		Action:      "Packet originates",
		Destination: destCIDR,
		Description: fmt.Sprintf("Packet starts in subnet %s (%s)", sourceSubnet["name"], sourceSubnet["cidr"]),
	})
	stepNum++

	// Step 2: Lookup route in source subnet's route table
	routeTableID := sourceSubnet["routeTableId"].(string)
	route, err := w.findMatchingRoute(routeTableID, destCIDR, "VPCRouteTable")
	if err != nil {
		walk.FailureReason = fmt.Sprintf("No route found in source route table: %s", err)
		return walk, nil
	}

	walk.Steps = append(walk.Steps, Step{
		StepNumber:  stepNum,
		Location:    "VPC Route Table",
		LocationID:  routeTableID,
		Action:      "Route lookup",
		Destination: route["destination"].(string),
		NextHop:     route["target"].(string),
		Description: fmt.Sprintf("Route table lookup: destination %s matches route to %s (type: %s)",
			destCIDR, route["target"], route["targetType"]),
	})
	stepNum++

	targetType := route["targetType"].(string)
	target := route["target"].(string)

	// Check if target is local (same VPC)
	if targetType == "local" {
		walk.Steps = append(walk.Steps, Step{
			StepNumber:  stepNum,
			Location:    "Destination Subnet",
			LocationID:  destSubnet["id"].(string),
			Action:      "Local delivery",
			Destination: destCIDR,
			Description: fmt.Sprintf("Packet delivered locally to subnet %s (%s)",
				destSubnet["name"], destSubnet["cidr"]),
		})
		walk.Success = true
		return walk, nil
	}

	// If target is TGW, proceed with TGW routing
	if targetType == "transit-gateway" {
		tgwSteps, success, failureReason := w.walkThroughTGW(target, sourceSubnet, destSubnet, destCIDR, stepNum)
		walk.Steps = append(walk.Steps, tgwSteps...)
		walk.Success = success
		walk.FailureReason = failureReason
		return walk, nil
	}

	walk.FailureReason = fmt.Sprintf("Unsupported target type: %s", targetType)
	return walk, nil
}

func (w *Walker) walkThroughTGW(tgwID string, sourceSubnet, destSubnet map[string]any, destCIDR string, stepNum int) ([]Step, bool, string) {
	var steps []Step

	// Step: Packet enters TGW
	steps = append(steps, Step{
		StepNumber:  stepNum,
		Location:    "Transit Gateway",
		LocationID:  tgwID,
		Action:      "Enter TGW",
		Destination: destCIDR,
		Description: fmt.Sprintf("Packet enters Transit Gateway %s", tgwID),
	})
	stepNum++

	// Find source VPC's TGW attachment
	sourceVPCID := sourceSubnet["vpcId"].(string)
	sourceAttachment, err := w.findAttachmentByVPC(sourceVPCID)
	if err != nil {
		return steps, false, fmt.Sprintf("Source VPC attachment not found: %s", err)
	}

	steps = append(steps, Step{
		StepNumber:  stepNum,
		Location:    "TGW Attachment",
		LocationID:  sourceAttachment["id"].(string),
		Action:      "Attachment lookup",
		Destination: destCIDR,
		Description: fmt.Sprintf("Packet enters via attachment %s", sourceAttachment["id"]),
	})
	stepNum++

	// Find which TGW route table is associated with this attachment
	tgwRouteTableID := sourceAttachment["routeTableId"].(string)

	steps = append(steps, Step{
		StepNumber:  stepNum,
		Location:    "TGW Route Table",
		LocationID:  tgwRouteTableID,
		Action:      "Route table association",
		Destination: destCIDR,
		Description: fmt.Sprintf("Using TGW route table %s", tgwRouteTableID),
	})
	stepNum++

	// Lookup route in TGW route table
	tgwRoute, err := w.findMatchingRoute(tgwRouteTableID, destCIDR, "TGWRouteTable")
	if err != nil {
		return steps, false, fmt.Sprintf("No route found in TGW route table: %s", err)
	}

	targetAttachmentID := tgwRoute["attachmentId"].(string)

	steps = append(steps, Step{
		StepNumber:  stepNum,
		Location:    "TGW Route Table",
		LocationID:  tgwRouteTableID,
		Action:      "Route lookup",
		Destination: tgwRoute["destination"].(string),
		NextHop:     targetAttachmentID,
		Description: fmt.Sprintf("Route found: destination %s -> attachment %s",
			tgwRoute["destination"], targetAttachmentID),
	})
	stepNum++

	// Check if target attachment is the destination VPC or an intermediate VPC (like firewall)
	targetAttachment, err := w.getAttachment(targetAttachmentID)
	if err != nil {
		return steps, false, fmt.Sprintf("Target attachment not found: %s", err)
	}

	targetVPCID := targetAttachment["resourceId"].(string)

	steps = append(steps, Step{
		StepNumber:  stepNum,
		Location:    "TGW Attachment",
		LocationID:  targetAttachmentID,
		Action:      "Exit TGW",
		Destination: destCIDR,
		NextHop:     targetVPCID,
		Description: fmt.Sprintf("Packet exits TGW via attachment %s to VPC %s",
			targetAttachmentID, targetVPCID),
	})
	stepNum++

	// Get destination VPC ID from destination subnet
	destVPCID := destSubnet["vpcId"].(string)

	// Check if we arrived at the destination VPC
	if targetVPCID == destVPCID {
		// We're at the destination VPC, do final routing
		return w.finalVPCRouting(targetVPCID, destSubnet, destCIDR, stepNum, steps)
	}

	// We're at an intermediate VPC (e.g., firewall VPC)
	intermediateSteps, success, failureReason := w.walkThroughIntermediateVPC(
		targetVPCID, destVPCID, destSubnet, destCIDR, stepNum)
	steps = append(steps, intermediateSteps...)

	return steps, success, failureReason
}

func (w *Walker) walkThroughIntermediateVPC(intermediateVPCID, destVPCID string,
	destSubnet map[string]any, destCIDR string, stepNum int) ([]Step, bool, string) {
	var steps []Step

	intermediateVPC, err := w.getVPC(intermediateVPCID)
	if err != nil {
		return steps, false, fmt.Sprintf("Intermediate VPC not found: %s", err)
	}

	steps = append(steps, Step{
		StepNumber:  stepNum,
		Location:    "Intermediate VPC",
		LocationID:  intermediateVPCID,
		Action:      "Enter intermediate VPC",
		Destination: destCIDR,
		Description: fmt.Sprintf("Packet enters intermediate VPC %s (%s) - typically firewall/inspection VPC",
			intermediateVPC["name"], intermediateVPC["cidr"]),
	})
	stepNum++

	// In a firewall VPC, packet would go through inspection
	steps = append(steps, Step{
		StepNumber:  stepNum,
		Location:    "Firewall/Inspection",
		LocationID:  intermediateVPCID,
		Action:      "Traffic inspection",
		Destination: destCIDR,
		Description: "Packet inspected by firewall (simulated)",
	})
	stepNum++

	// Find route back to TGW for the destination
	// Get the TGW subnet's route table in the intermediate VPC
	tgwSubnets, err := w.getTGWSubnetsForVPC(intermediateVPCID)
	if err != nil || len(tgwSubnets) == 0 {
		return steps, false, "No TGW subnets found in intermediate VPC"
	}

	tgwSubnet := tgwSubnets[0]
	tgwRTID := tgwSubnet["routeTableId"].(string)

	route, err := w.findMatchingRoute(tgwRTID, destCIDR, "VPCRouteTable")
	if err != nil {
		return steps, false, fmt.Sprintf("No route found in intermediate VPC: %s", err)
	}

	steps = append(steps, Step{
		StepNumber:  stepNum,
		Location:    "Intermediate VPC Route Table",
		LocationID:  tgwRTID,
		Action:      "Route lookup",
		Destination: route["destination"].(string),
		NextHop:     route["target"].(string),
		Description: fmt.Sprintf("Route to destination: %s via %s",
			route["destination"], route["target"]),
	})
	stepNum++

	// Packet goes back to TGW
	tgwID := route["target"].(string)
	if route["targetType"].(string) != "transit-gateway" {
		// Check if it's a VPC endpoint (firewall endpoint)
		if route["targetType"].(string) == "vpc-endpoint" {
			steps = append(steps, Step{
				StepNumber:  stepNum,
				Location:    "VPC Endpoint",
				LocationID:  route["target"].(string),
				Action:      "Forward through endpoint",
				Destination: destCIDR,
				Description: "Packet forwarded through VPC endpoint (firewall endpoint)",
			})
			stepNum++
			// After endpoint, it goes back to TGW
			tgwID = "tgw-0123456789abcdef0" // This would be looked up in real scenario
		} else {
			return steps, false, fmt.Sprintf("Expected TGW or endpoint, got: %s", route["targetType"])
		}
	}

	steps = append(steps, Step{
		StepNumber:  stepNum,
		Location:    "Transit Gateway",
		LocationID:  tgwID,
		Action:      "Re-enter TGW",
		Destination: destCIDR,
		Description: "Packet re-enters Transit Gateway from intermediate VPC",
	})
	stepNum++

	// Find intermediate VPC's attachment
	intermediateAttachment, err := w.findAttachmentByVPC(intermediateVPCID)
	if err != nil {
		return steps, false, fmt.Sprintf("Intermediate VPC attachment not found: %s", err)
	}

	// Get the route table associated with firewall attachment
	firewallRTID := intermediateAttachment["routeTableId"].(string)

	steps = append(steps, Step{
		StepNumber:  stepNum,
		Location:    "TGW Route Table",
		LocationID:  firewallRTID,
		Action:      "Route table lookup",
		Destination: destCIDR,
		Description: fmt.Sprintf("Using firewall route table %s", firewallRTID),
	})
	stepNum++

	// Lookup route in firewall TGW route table
	tgwRoute, err := w.findMatchingRoute(firewallRTID, destCIDR, "TGWRouteTable")
	if err != nil {
		return steps, false, fmt.Sprintf("No route found in firewall TGW route table: %s", err)
	}

	targetAttachmentID := tgwRoute["attachmentId"].(string)

	steps = append(steps, Step{
		StepNumber:  stepNum,
		Location:    "TGW Route Table",
		LocationID:  firewallRTID,
		Action:      "Route lookup",
		Destination: tgwRoute["destination"].(string),
		NextHop:     targetAttachmentID,
		Description: fmt.Sprintf("Route to destination VPC: %s -> attachment %s",
			tgwRoute["destination"], targetAttachmentID),
	})
	stepNum++

	// Exit to destination VPC
	steps = append(steps, Step{
		StepNumber:  stepNum,
		Location:    "TGW Attachment",
		LocationID:  targetAttachmentID,
		Action:      "Exit TGW",
		Destination: destCIDR,
		NextHop:     destVPCID,
		Description: fmt.Sprintf("Packet exits TGW to destination VPC via attachment %s",
			targetAttachmentID),
	})
	stepNum++

	// Final routing in destination VPC
	finalSteps, success, failureReason := w.finalVPCRouting(destVPCID, destSubnet, destCIDR, stepNum, steps)
	return finalSteps, success, failureReason
}

func (w *Walker) finalVPCRouting(vpcID string, destSubnet map[string]any, destCIDR string,
	stepNum int, existingSteps []Step) ([]Step, bool, string) {
	steps := existingSteps

	vpc, err := w.getVPC(vpcID)
	if err != nil {
		return steps, false, fmt.Sprintf("Destination VPC not found: %s", err)
	}

	steps = append(steps, Step{
		StepNumber:  stepNum,
		Location:    "Destination VPC",
		LocationID:  vpcID,
		Action:      "Enter destination VPC",
		Destination: destCIDR,
		Description: fmt.Sprintf("Packet enters destination VPC %s (%s)",
			vpc["name"], vpc["cidr"]),
	})
	stepNum++

	// Route to final subnet
	destRouteTableID := destSubnet["routeTableId"].(string)

	steps = append(steps, Step{
		StepNumber:  stepNum,
		Location:    "VPC Route Table",
		LocationID:  destRouteTableID,
		Action:      "Final route lookup",
		Destination: destCIDR,
		Description: "Route lookup for local delivery",
	})
	stepNum++

	steps = append(steps, Step{
		StepNumber:  stepNum,
		Location:    "Destination Subnet",
		LocationID:  destSubnet["id"].(string),
		Action:      "Packet delivered",
		Destination: destCIDR,
		Description: fmt.Sprintf("Packet successfully delivered to subnet %s (%s)",
			destSubnet["name"], destSubnet["cidr"]),
	})

	return steps, true, ""
}

// Helper functions to query Neo4j

func (w *Walker) findSubnetByCIDR(cidr string) (map[string]any, error) {
	query := `
		MATCH (s:Subnet {cidr: $cidr})
		MATCH (v:VPC)-[:HAS_SUBNET]->(s)
		RETURN s.id as id, s.cidr as cidr, s.name as name,
		       s.routeTableId as routeTableId, v.id as vpcId
	`
	params := map[string]any{"cidr": cidr}

	records, err := w.client.ExecuteReadQuery(query, params)
	if err != nil {
		return nil, err
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("subnet not found")
	}

	result := make(map[string]any)
	for k, v := range records[0].AsMap() {
		result[k] = v
	}

	return result, nil
}

func (w *Walker) findMatchingRoute(routeTableID, destCIDR, tableType string) (map[string]any, error) {
	var query string

	if tableType == "VPCRouteTable" {
		query = `
			MATCH (rt:VPCRouteTable {id: $rtId})-[:HAS_ROUTE]->(r:VPCRoute)
			RETURN r.destination as destination, r.target as target,
			       r.targetType as targetType
			ORDER BY r.destination DESC
		`
	} else {
		query = `
			MATCH (rt:TGWRouteTable {id: $rtId})-[:HAS_ROUTE]->(r:TGWRoute)
			RETURN r.destination as destination, r.attachmentId as attachmentId,
			       r.routeType as routeType
			ORDER BY r.destination DESC
		`
	}

	params := map[string]any{"rtId": routeTableID}

	records, err := w.client.ExecuteReadQuery(query, params)
	if err != nil {
		return nil, err
	}

	// Find the most specific matching route
	destIP, destNet, err := net.ParseCIDR(destCIDR)
	if err != nil {
		return nil, fmt.Errorf("invalid destination CIDR: %w", err)
	}

	for _, record := range records {
		recordMap := record.AsMap()
		routeDest := recordMap["destination"].(string)

		// Check for default route
		if routeDest == "0.0.0.0/0" {
			return recordMap, nil
		}

		// Check if destination matches route
		_, routeNet, err := net.ParseCIDR(routeDest)
		if err != nil {
			continue
		}

		if routeNet.Contains(destIP) || destNet.String() == routeDest {
			return recordMap, nil
		}
	}

	return nil, fmt.Errorf("no matching route found")
}

func (w *Walker) findAttachmentByVPC(vpcID string) (map[string]any, error) {
	query := `
		MATCH (v:VPC {id: $vpcId})-[:ATTACHED_VIA]->(att:TGWAttachment)
		RETURN att.id as id, att.resourceId as resourceId,
		       att.routeTableId as routeTableId
	`
	params := map[string]any{"vpcId": vpcID}

	records, err := w.client.ExecuteReadQuery(query, params)
	if err != nil {
		return nil, err
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("attachment not found")
	}

	return records[0].AsMap(), nil
}

func (w *Walker) getAttachment(attachmentID string) (map[string]any, error) {
	query := `
		MATCH (att:TGWAttachment {id: $attId})
		RETURN att.id as id, att.resourceId as resourceId,
		       att.routeTableId as routeTableId
	`
	params := map[string]any{"attId": attachmentID}

	records, err := w.client.ExecuteReadQuery(query, params)
	if err != nil {
		return nil, err
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("attachment not found")
	}

	return records[0].AsMap(), nil
}

func (w *Walker) getVPC(vpcID string) (map[string]any, error) {
	query := `
		MATCH (v:VPC {id: $vpcId})
		RETURN v.id as id, v.name as name, v.cidr as cidr
	`
	params := map[string]any{"vpcId": vpcID}

	records, err := w.client.ExecuteReadQuery(query, params)
	if err != nil {
		return nil, err
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("VPC not found")
	}

	return records[0].AsMap(), nil
}

func (w *Walker) getTGWSubnetsForVPC(vpcID string) ([]map[string]any, error) {
	query := `
		MATCH (v:VPC {id: $vpcId})-[:HAS_SUBNET]->(s:Subnet)
		WHERE s.name CONTAINS 'tgw'
		RETURN s.id as id, s.cidr as cidr, s.routeTableId as routeTableId
	`
	params := map[string]any{"vpcId": vpcID}

	records, err := w.client.ExecuteReadQuery(query, params)
	if err != nil {
		return nil, err
	}

	var results []map[string]any
	for _, record := range records {
		results = append(results, record.AsMap())
	}

	return results, nil
}

// PrintWalk prints the packet walk in a readable format
func PrintWalk(walk *PacketWalk) {
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("PACKET WALK: %s -> %s\n", walk.SourceSubnet, walk.DestinationSubnet)
	fmt.Println(strings.Repeat("=", 80))

	for _, step := range walk.Steps {
		fmt.Printf("\nStep %d: %s\n", step.StepNumber, step.Location)
		fmt.Printf("  ID: %s\n", step.LocationID)
		fmt.Printf("  Action: %s\n", step.Action)
		if step.Destination != "" {
			fmt.Printf("  Destination: %s\n", step.Destination)
		}
		if step.NextHop != "" {
			fmt.Printf("  Next Hop: %s\n", step.NextHop)
		}
		fmt.Printf("  Description: %s\n", step.Description)
	}

	fmt.Println(strings.Repeat("-", 80))
	if walk.Success {
		fmt.Println("✓ PACKET DELIVERED SUCCESSFULLY")
	} else {
		fmt.Printf("✗ PACKET DELIVERY FAILED: %s\n", walk.FailureReason)
	}
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println()
}
