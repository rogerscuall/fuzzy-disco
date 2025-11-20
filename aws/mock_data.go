package aws

import (
	"github.com/rogerscuall/fuzzy-disco/models"
)

// GetMockAWSEnvironment creates a mock AWS environment with TGW, VPCs, and routing
// Scenarios included:
// 1. Direct VPC-to-VPC communication through TGW
// 2. Central Firewall VPC (inspection VPC) scenario
// 3. Spoke VPCs communicating through the firewall
func GetMockAWSEnvironment() *models.AWSEnvironment {
	tgwID := "tgw-0123456789abcdef0"
	tgwDefaultRT := "tgw-rtb-default"
	tgwFirewallRT := "tgw-rtb-firewall"
	tgwSpokeRT := "tgw-rtb-spoke"

	// Create VPCs
	vpcProd := createProductionVPC()
	vpcDev := createDevelopmentVPC()
	vpcShared := createSharedServicesVPC()
	vpcFirewall := createFirewallVPC()

	// Create TGW attachments
	attachments := []models.TGWAttachment{
		{
			ID:               "tgw-attach-prod",
			TransitGatewayID: tgwID,
			ResourceType:     "vpc",
			ResourceID:       vpcProd.ID,
			State:            "available",
			RouteTableID:     tgwSpokeRT,
		},
		{
			ID:               "tgw-attach-dev",
			TransitGatewayID: tgwID,
			ResourceType:     "vpc",
			ResourceID:       vpcDev.ID,
			State:            "available",
			RouteTableID:     tgwSpokeRT,
		},
		{
			ID:               "tgw-attach-shared",
			TransitGatewayID: tgwID,
			ResourceType:     "vpc",
			ResourceID:       vpcShared.ID,
			State:            "available",
			RouteTableID:     tgwSpokeRT,
		},
		{
			ID:               "tgw-attach-firewall",
			TransitGatewayID: tgwID,
			ResourceType:     "vpc",
			ResourceID:       vpcFirewall.ID,
			State:            "available",
			RouteTableID:     tgwFirewallRT,
		},
	}

	// Link attachments to VPCs
	vpcProd.TGWAttachmentID = "tgw-attach-prod"
	vpcDev.TGWAttachmentID = "tgw-attach-dev"
	vpcShared.TGWAttachmentID = "tgw-attach-shared"
	vpcFirewall.TGWAttachmentID = "tgw-attach-firewall"

	// Create TGW route tables
	routeTables := []models.TGWRouteTable{
		{
			ID:               tgwDefaultRT,
			TransitGatewayID: tgwID,
			Name:             "default-route-table",
			DefaultTable:     true,
			Routes:           []models.TGWRoute{},
			Associations:     []string{},
			Propagations:     []string{},
		},
		{
			ID:               tgwSpokeRT,
			TransitGatewayID: tgwID,
			Name:             "spoke-route-table",
			DefaultTable:     false,
			Routes: []models.TGWRoute{
				// All spoke traffic goes to firewall VPC
				{
					DestinationCIDR: "0.0.0.0/0",
					AttachmentID:    "tgw-attach-firewall",
					RouteType:       "static",
					State:           "active",
				},
			},
			Associations: []string{"tgw-attach-prod", "tgw-attach-dev", "tgw-attach-shared"},
			Propagations: []string{},
		},
		{
			ID:               tgwFirewallRT,
			TransitGatewayID: tgwID,
			Name:             "firewall-route-table",
			DefaultTable:     false,
			Routes: []models.TGWRoute{
				// Specific routes back to each spoke VPC
				{
					DestinationCIDR: vpcProd.CIDR,
					AttachmentID:    "tgw-attach-prod",
					RouteType:       "static",
					State:           "active",
				},
				{
					DestinationCIDR: vpcDev.CIDR,
					AttachmentID:    "tgw-attach-dev",
					RouteType:       "static",
					State:           "active",
				},
				{
					DestinationCIDR: vpcShared.CIDR,
					AttachmentID:    "tgw-attach-shared",
					RouteType:       "static",
					State:           "active",
				},
			},
			Associations: []string{"tgw-attach-firewall"},
			Propagations: []string{},
		},
	}

	// Create Transit Gateway
	tgw := models.TransitGateway{
		ID:                  tgwID,
		Name:                "main-tgw",
		AmazonSideASN:       64512,
		DefaultRouteTableID: tgwDefaultRT,
		Attachments:         attachments,
		RouteTables:         routeTables,
		State:               "available",
	}

	return &models.AWSEnvironment{
		TransitGateways: []models.TransitGateway{tgw},
		VPCs:            []models.VPC{vpcProd, vpcDev, vpcShared, vpcFirewall},
	}
}

func createProductionVPC() models.VPC {
	vpcID := "vpc-prod-001"
	return models.VPC{
		ID:   vpcID,
		Name: "production-vpc",
		CIDR: "10.1.0.0/16",
		Subnets: []models.Subnet{
			{
				ID:               "subnet-prod-app-1a",
				VPCID:            vpcID,
				CIDR:             "10.1.1.0/24",
				AvailabilityZone: "us-east-1a",
				Name:             "prod-app-subnet-1a",
				RouteTableID:     "rtb-prod-private",
			},
			{
				ID:               "subnet-prod-app-1b",
				VPCID:            vpcID,
				CIDR:             "10.1.2.0/24",
				AvailabilityZone: "us-east-1b",
				Name:             "prod-app-subnet-1b",
				RouteTableID:     "rtb-prod-private",
			},
			{
				ID:               "subnet-prod-tgw-1a",
				VPCID:            vpcID,
				CIDR:             "10.1.255.0/28",
				AvailabilityZone: "us-east-1a",
				Name:             "prod-tgw-subnet-1a",
				RouteTableID:     "rtb-prod-tgw",
			},
			{
				ID:               "subnet-prod-tgw-1b",
				VPCID:            vpcID,
				CIDR:             "10.1.255.16/28",
				AvailabilityZone: "us-east-1b",
				Name:             "prod-tgw-subnet-1b",
				RouteTableID:     "rtb-prod-tgw",
			},
		},
		RouteTables: []models.RouteTable{
			{
				ID:      "rtb-prod-private",
				VPCID:   vpcID,
				Name:    "prod-private-rt",
				Subnets: []string{"subnet-prod-app-1a", "subnet-prod-app-1b"},
				Main:    false,
				Routes: []models.Route{
					{
						DestinationCIDR: "10.1.0.0/16",
						Target:          "local",
						TargetType:      "local",
					},
					{
						DestinationCIDR: "0.0.0.0/0",
						Target:          "tgw-0123456789abcdef0",
						TargetType:      "transit-gateway",
					},
				},
			},
			{
				ID:      "rtb-prod-tgw",
				VPCID:   vpcID,
				Name:    "prod-tgw-rt",
				Subnets: []string{"subnet-prod-tgw-1a", "subnet-prod-tgw-1b"},
				Main:    false,
				Routes: []models.Route{
					{
						DestinationCIDR: "10.1.0.0/16",
						Target:          "local",
						TargetType:      "local",
					},
				},
			},
		},
	}
}

func createDevelopmentVPC() models.VPC {
	vpcID := "vpc-dev-001"
	return models.VPC{
		ID:   vpcID,
		Name: "development-vpc",
		CIDR: "10.2.0.0/16",
		Subnets: []models.Subnet{
			{
				ID:               "subnet-dev-app-1a",
				VPCID:            vpcID,
				CIDR:             "10.2.1.0/24",
				AvailabilityZone: "us-east-1a",
				Name:             "dev-app-subnet-1a",
				RouteTableID:     "rtb-dev-private",
			},
			{
				ID:               "subnet-dev-tgw-1a",
				VPCID:            vpcID,
				CIDR:             "10.2.255.0/28",
				AvailabilityZone: "us-east-1a",
				Name:             "dev-tgw-subnet-1a",
				RouteTableID:     "rtb-dev-tgw",
			},
		},
		RouteTables: []models.RouteTable{
			{
				ID:      "rtb-dev-private",
				VPCID:   vpcID,
				Name:    "dev-private-rt",
				Subnets: []string{"subnet-dev-app-1a"},
				Main:    true,
				Routes: []models.Route{
					{
						DestinationCIDR: "10.2.0.0/16",
						Target:          "local",
						TargetType:      "local",
					},
					{
						DestinationCIDR: "0.0.0.0/0",
						Target:          "tgw-0123456789abcdef0",
						TargetType:      "transit-gateway",
					},
				},
			},
			{
				ID:      "rtb-dev-tgw",
				VPCID:   vpcID,
				Name:    "dev-tgw-rt",
				Subnets: []string{"subnet-dev-tgw-1a"},
				Main:    false,
				Routes: []models.Route{
					{
						DestinationCIDR: "10.2.0.0/16",
						Target:          "local",
						TargetType:      "local",
					},
				},
			},
		},
	}
}

func createSharedServicesVPC() models.VPC {
	vpcID := "vpc-shared-001"
	return models.VPC{
		ID:   vpcID,
		Name: "shared-services-vpc",
		CIDR: "10.3.0.0/16",
		Subnets: []models.Subnet{
			{
				ID:               "subnet-shared-services-1a",
				VPCID:            vpcID,
				CIDR:             "10.3.1.0/24",
				AvailabilityZone: "us-east-1a",
				Name:             "shared-services-subnet-1a",
				RouteTableID:     "rtb-shared-private",
			},
			{
				ID:               "subnet-shared-tgw-1a",
				VPCID:            vpcID,
				CIDR:             "10.3.255.0/28",
				AvailabilityZone: "us-east-1a",
				Name:             "shared-tgw-subnet-1a",
				RouteTableID:     "rtb-shared-tgw",
			},
		},
		RouteTables: []models.RouteTable{
			{
				ID:      "rtb-shared-private",
				VPCID:   vpcID,
				Name:    "shared-private-rt",
				Subnets: []string{"subnet-shared-services-1a"},
				Main:    true,
				Routes: []models.Route{
					{
						DestinationCIDR: "10.3.0.0/16",
						Target:          "local",
						TargetType:      "local",
					},
					{
						DestinationCIDR: "0.0.0.0/0",
						Target:          "tgw-0123456789abcdef0",
						TargetType:      "transit-gateway",
					},
				},
			},
			{
				ID:      "rtb-shared-tgw",
				VPCID:   vpcID,
				Name:    "shared-tgw-rt",
				Subnets: []string{"subnet-shared-tgw-1a"},
				Main:    false,
				Routes: []models.Route{
					{
						DestinationCIDR: "10.3.0.0/16",
						Target:          "local",
						TargetType:      "local",
					},
				},
			},
		},
	}
}

func createFirewallVPC() models.VPC {
	vpcID := "vpc-firewall-001"
	return models.VPC{
		ID:   vpcID,
		Name: "firewall-vpc",
		CIDR: "10.100.0.0/16",
		Subnets: []models.Subnet{
			{
				ID:               "subnet-firewall-1a",
				VPCID:            vpcID,
				CIDR:             "10.100.1.0/24",
				AvailabilityZone: "us-east-1a",
				Name:             "firewall-subnet-1a",
				RouteTableID:     "rtb-firewall-inspection",
			},
			{
				ID:               "subnet-firewall-tgw-1a",
				VPCID:            vpcID,
				CIDR:             "10.100.255.0/28",
				AvailabilityZone: "us-east-1a",
				Name:             "firewall-tgw-subnet-1a",
				RouteTableID:     "rtb-firewall-tgw",
			},
		},
		RouteTables: []models.RouteTable{
			{
				ID:      "rtb-firewall-inspection",
				VPCID:   vpcID,
				Name:    "firewall-inspection-rt",
				Subnets: []string{"subnet-firewall-1a"},
				Main:    false,
				Routes: []models.Route{
					{
						DestinationCIDR: "10.100.0.0/16",
						Target:          "local",
						TargetType:      "local",
					},
					{
						DestinationCIDR: "10.1.0.0/16",
						Target:          "tgw-0123456789abcdef0",
						TargetType:      "transit-gateway",
					},
					{
						DestinationCIDR: "10.2.0.0/16",
						Target:          "tgw-0123456789abcdef0",
						TargetType:      "transit-gateway",
					},
					{
						DestinationCIDR: "10.3.0.0/16",
						Target:          "tgw-0123456789abcdef0",
						TargetType:      "transit-gateway",
					},
				},
			},
			{
				ID:      "rtb-firewall-tgw",
				VPCID:   vpcID,
				Name:    "firewall-tgw-rt",
				Subnets: []string{"subnet-firewall-tgw-1a"},
				Main:    false,
				Routes: []models.Route{
					{
						DestinationCIDR: "10.100.0.0/16",
						Target:          "local",
						TargetType:      "local",
					},
					{
						DestinationCIDR: "0.0.0.0/0",
						Target:          "vpce-firewall-endpoint",
						TargetType:      "vpc-endpoint",
					},
				},
			},
		},
	}
}
