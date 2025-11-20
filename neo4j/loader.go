package neo4j

import (
	"fmt"

	"github.com/rogerscuall/fuzzy-disco/models"
)

// LoadAWSEnvironment loads the AWS environment into Neo4j
func (c *Neo4jClient) LoadAWSEnvironment(env *models.AWSEnvironment) error {
	// Clear existing data
	if err := c.ClearDatabase(); err != nil {
		return fmt.Errorf("failed to clear database: %w", err)
	}

	// Load each Transit Gateway
	for _, tgw := range env.TransitGateways {
		if err := c.loadTransitGateway(&tgw); err != nil {
			return fmt.Errorf("failed to load TGW %s: %w", tgw.ID, err)
		}
	}

	// Load each VPC
	for _, vpc := range env.VPCs {
		if err := c.loadVPC(&vpc); err != nil {
			return fmt.Errorf("failed to load VPC %s: %w", vpc.ID, err)
		}
	}

	// Create relationships between TGW and VPCs through attachments
	for _, tgw := range env.TransitGateways {
		for _, attachment := range tgw.Attachments {
			if err := c.createAttachmentRelationship(&attachment); err != nil {
				return fmt.Errorf("failed to create attachment relationship: %w", err)
			}
		}
	}

	return nil
}

func (c *Neo4jClient) loadTransitGateway(tgw *models.TransitGateway) error {
	// Create TGW node
	query := `
		CREATE (tgw:TransitGateway {
			id: $id,
			name: $name,
			asn: $asn,
			state: $state
		})
	`
	params := map[string]any{
		"id":    tgw.ID,
		"name":  tgw.Name,
		"asn":   tgw.AmazonSideASN,
		"state": tgw.State,
	}

	if err := c.ExecuteWriteQuery(query, params); err != nil {
		return err
	}

	// Create TGW route tables
	for _, rt := range tgw.RouteTables {
		if err := c.loadTGWRouteTable(tgw.ID, &rt); err != nil {
			return err
		}
	}

	// Create TGW attachments
	for _, attachment := range tgw.Attachments {
		if err := c.loadTGWAttachment(tgw.ID, &attachment); err != nil {
			return err
		}
	}

	return nil
}

func (c *Neo4jClient) loadTGWRouteTable(tgwID string, rt *models.TGWRouteTable) error {
	// Create route table node
	query := `
		MATCH (tgw:TransitGateway {id: $tgwId})
		CREATE (rt:TGWRouteTable {
			id: $id,
			name: $name,
			isDefault: $isDefault
		})
		CREATE (tgw)-[:HAS_ROUTE_TABLE]->(rt)
	`
	params := map[string]any{
		"tgwId":     tgwID,
		"id":        rt.ID,
		"name":      rt.Name,
		"isDefault": rt.DefaultTable,
	}

	if err := c.ExecuteWriteQuery(query, params); err != nil {
		return err
	}

	// Create routes
	for _, route := range rt.Routes {
		if err := c.loadTGWRoute(rt.ID, &route); err != nil {
			return err
		}
	}

	return nil
}

func (c *Neo4jClient) loadTGWRoute(rtID string, route *models.TGWRoute) error {
	query := `
		MATCH (rt:TGWRouteTable {id: $rtId})
		CREATE (r:TGWRoute {
			destination: $destination,
			attachmentId: $attachmentId,
			routeType: $routeType,
			state: $state
		})
		CREATE (rt)-[:HAS_ROUTE]->(r)
	`
	params := map[string]any{
		"rtId":         rtID,
		"destination":  route.DestinationCIDR,
		"attachmentId": route.AttachmentID,
		"routeType":    route.RouteType,
		"state":        route.State,
	}

	return c.ExecuteWriteQuery(query, params)
}

func (c *Neo4jClient) loadTGWAttachment(tgwID string, attachment *models.TGWAttachment) error {
	query := `
		MATCH (tgw:TransitGateway {id: $tgwId})
		CREATE (att:TGWAttachment {
			id: $id,
			resourceType: $resourceType,
			resourceId: $resourceId,
			state: $state,
			routeTableId: $routeTableId
		})
		CREATE (tgw)-[:HAS_ATTACHMENT]->(att)
	`
	params := map[string]any{
		"tgwId":        tgwID,
		"id":           attachment.ID,
		"resourceType": attachment.ResourceType,
		"resourceId":   attachment.ResourceID,
		"state":        attachment.State,
		"routeTableId": attachment.RouteTableID,
	}

	if err := c.ExecuteWriteQuery(query, params); err != nil {
		return err
	}

	// Link attachment to its route table
	linkQuery := `
		MATCH (att:TGWAttachment {id: $attId})
		MATCH (rt:TGWRouteTable {id: $rtId})
		CREATE (att)-[:USES_ROUTE_TABLE]->(rt)
	`
	linkParams := map[string]any{
		"attId": attachment.ID,
		"rtId":  attachment.RouteTableID,
	}

	return c.ExecuteWriteQuery(linkQuery, linkParams)
}

func (c *Neo4jClient) loadVPC(vpc *models.VPC) error {
	// Create VPC node
	query := `
		CREATE (vpc:VPC {
			id: $id,
			name: $name,
			cidr: $cidr,
			tgwAttachmentId: $tgwAttachmentId
		})
	`
	params := map[string]any{
		"id":              vpc.ID,
		"name":            vpc.Name,
		"cidr":            vpc.CIDR,
		"tgwAttachmentId": vpc.TGWAttachmentID,
	}

	if err := c.ExecuteWriteQuery(query, params); err != nil {
		return err
	}

	// Create subnets
	for _, subnet := range vpc.Subnets {
		if err := c.loadSubnet(vpc.ID, &subnet); err != nil {
			return err
		}
	}

	// Create route tables
	for _, rt := range vpc.RouteTables {
		if err := c.loadVPCRouteTable(vpc.ID, &rt); err != nil {
			return err
		}
	}

	return nil
}

func (c *Neo4jClient) loadSubnet(vpcID string, subnet *models.Subnet) error {
	query := `
		MATCH (vpc:VPC {id: $vpcId})
		CREATE (subnet:Subnet {
			id: $id,
			cidr: $cidr,
			az: $az,
			name: $name,
			routeTableId: $routeTableId
		})
		CREATE (vpc)-[:HAS_SUBNET]->(subnet)
	`
	params := map[string]any{
		"vpcId":        vpcID,
		"id":           subnet.ID,
		"cidr":         subnet.CIDR,
		"az":           subnet.AvailabilityZone,
		"name":         subnet.Name,
		"routeTableId": subnet.RouteTableID,
	}

	return c.ExecuteWriteQuery(query, params)
}

func (c *Neo4jClient) loadVPCRouteTable(vpcID string, rt *models.RouteTable) error {
	// Create route table node
	query := `
		MATCH (vpc:VPC {id: $vpcId})
		CREATE (rt:VPCRouteTable {
			id: $id,
			name: $name,
			isMain: $isMain
		})
		CREATE (vpc)-[:HAS_ROUTE_TABLE]->(rt)
	`
	params := map[string]any{
		"vpcId":  vpcID,
		"id":     rt.ID,
		"name":   rt.Name,
		"isMain": rt.Main,
	}

	if err := c.ExecuteWriteQuery(query, params); err != nil {
		return err
	}

	// Link subnets to route table
	for _, subnetID := range rt.Subnets {
		linkQuery := `
			MATCH (rt:VPCRouteTable {id: $rtId})
			MATCH (subnet:Subnet {id: $subnetId})
			CREATE (subnet)-[:USES_ROUTE_TABLE]->(rt)
		`
		linkParams := map[string]any{
			"rtId":     rt.ID,
			"subnetId": subnetID,
		}
		if err := c.ExecuteWriteQuery(linkQuery, linkParams); err != nil {
			return err
		}
	}

	// Create routes
	for _, route := range rt.Routes {
		if err := c.loadVPCRoute(rt.ID, &route); err != nil {
			return err
		}
	}

	return nil
}

func (c *Neo4jClient) loadVPCRoute(rtID string, route *models.Route) error {
	query := `
		MATCH (rt:VPCRouteTable {id: $rtId})
		CREATE (r:VPCRoute {
			destination: $destination,
			target: $target,
			targetType: $targetType
		})
		CREATE (rt)-[:HAS_ROUTE]->(r)
	`
	params := map[string]any{
		"rtId":       rtID,
		"destination": route.DestinationCIDR,
		"target":     route.Target,
		"targetType": route.TargetType,
	}

	return c.ExecuteWriteQuery(query, params)
}

func (c *Neo4jClient) createAttachmentRelationship(attachment *models.TGWAttachment) error {
	// Link VPC to TGW attachment
	query := `
		MATCH (vpc:VPC {id: $vpcId})
		MATCH (att:TGWAttachment {id: $attId})
		CREATE (vpc)-[:ATTACHED_VIA]->(att)
	`
	params := map[string]any{
		"vpcId": attachment.ResourceID,
		"attId": attachment.ID,
	}

	return c.ExecuteWriteQuery(query, params)
}
