package aws

import (
	"testing"
)

func TestGetMockAWSEnvironment(t *testing.T) {
	env := GetMockAWSEnvironment()

	if env == nil {
		t.Fatal("Expected non-nil environment")
	}

	// Test Transit Gateways
	if len(env.TransitGateways) != 1 {
		t.Errorf("Expected 1 TGW, got %d", len(env.TransitGateways))
	}

	tgw := env.TransitGateways[0]
	if tgw.ID != "tgw-0123456789abcdef0" {
		t.Errorf("Expected TGW ID tgw-0123456789abcdef0, got %s", tgw.ID)
	}

	if tgw.Name != "main-tgw" {
		t.Errorf("Expected TGW name main-tgw, got %s", tgw.Name)
	}

	// Test VPCs
	if len(env.VPCs) != 4 {
		t.Errorf("Expected 4 VPCs, got %d", len(env.VPCs))
	}

	vpcNames := make(map[string]bool)
	for _, vpc := range env.VPCs {
		vpcNames[vpc.Name] = true
	}

	expectedVPCs := []string{"production-vpc", "development-vpc", "shared-services-vpc", "firewall-vpc"}
	for _, name := range expectedVPCs {
		if !vpcNames[name] {
			t.Errorf("Expected VPC %s not found", name)
		}
	}

	// Test TGW Attachments
	if len(tgw.Attachments) != 4 {
		t.Errorf("Expected 4 TGW attachments, got %d", len(tgw.Attachments))
	}

	// Test TGW Route Tables
	if len(tgw.RouteTables) != 3 {
		t.Errorf("Expected 3 TGW route tables, got %d", len(tgw.RouteTables))
	}

	// Verify spoke route table has default route to firewall
	spokeRT := tgw.RouteTables[1] // spoke-route-table
	if spokeRT.Name != "spoke-route-table" {
		t.Errorf("Expected spoke-route-table, got %s", spokeRT.Name)
	}

	if len(spokeRT.Routes) != 1 {
		t.Errorf("Expected 1 route in spoke route table, got %d", len(spokeRT.Routes))
	}

	defaultRoute := spokeRT.Routes[0]
	if defaultRoute.DestinationCIDR != "0.0.0.0/0" {
		t.Errorf("Expected default route 0.0.0.0/0, got %s", defaultRoute.DestinationCIDR)
	}

	if defaultRoute.AttachmentID != "tgw-attach-firewall" {
		t.Errorf("Expected route to firewall attachment, got %s", defaultRoute.AttachmentID)
	}
}

func TestProductionVPC(t *testing.T) {
	vpc := createProductionVPC()

	if vpc.ID != "vpc-prod-001" {
		t.Errorf("Expected VPC ID vpc-prod-001, got %s", vpc.ID)
	}

	if vpc.CIDR != "10.1.0.0/16" {
		t.Errorf("Expected CIDR 10.1.0.0/16, got %s", vpc.CIDR)
	}

	// Should have 4 subnets (2 app + 2 tgw)
	if len(vpc.Subnets) != 4 {
		t.Errorf("Expected 4 subnets, got %d", len(vpc.Subnets))
	}

	// Should have 2 route tables
	if len(vpc.RouteTables) != 2 {
		t.Errorf("Expected 2 route tables, got %d", len(vpc.RouteTables))
	}

	// Check private route table has TGW route
	privateRT := vpc.RouteTables[0]
	hasTGWRoute := false
	for _, route := range privateRT.Routes {
		if route.TargetType == "transit-gateway" {
			hasTGWRoute = true
			break
		}
	}

	if !hasTGWRoute {
		t.Error("Expected private route table to have TGW route")
	}
}

func TestFirewallVPC(t *testing.T) {
	vpc := createFirewallVPC()

	if vpc.Name != "firewall-vpc" {
		t.Errorf("Expected name firewall-vpc, got %s", vpc.Name)
	}

	if vpc.CIDR != "10.100.0.0/16" {
		t.Errorf("Expected CIDR 10.100.0.0/16, got %s", vpc.CIDR)
	}

	// Should have 2 subnets (inspection + tgw)
	if len(vpc.Subnets) != 2 {
		t.Errorf("Expected 2 subnets, got %d", len(vpc.Subnets))
	}

	// Should have 2 route tables
	if len(vpc.RouteTables) != 2 {
		t.Errorf("Expected 2 route tables, got %d", len(vpc.RouteTables))
	}

	// Check inspection route table has routes to spoke VPCs
	inspectionRT := vpc.RouteTables[0]
	spokeRoutes := 0
	for _, route := range inspectionRT.Routes {
		if route.TargetType == "transit-gateway" && route.DestinationCIDR != "0.0.0.0/0" {
			spokeRoutes++
		}
	}

	if spokeRoutes != 3 {
		t.Errorf("Expected 3 routes to spoke VPCs, got %d", spokeRoutes)
	}
}
