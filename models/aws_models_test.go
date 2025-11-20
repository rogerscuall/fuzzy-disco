package models

import (
	"testing"
)

func TestTransitGatewayStructure(t *testing.T) {
	tgw := TransitGateway{
		ID:                  "tgw-test123",
		Name:                "test-tgw",
		AmazonSideASN:       64512,
		DefaultRouteTableID: "rtb-default",
		State:               "available",
	}

	if tgw.ID != "tgw-test123" {
		t.Errorf("Expected ID tgw-test123, got %s", tgw.ID)
	}

	if tgw.AmazonSideASN != 64512 {
		t.Errorf("Expected ASN 64512, got %d", tgw.AmazonSideASN)
	}
}

func TestVPCStructure(t *testing.T) {
	vpc := VPC{
		ID:   "vpc-test123",
		Name: "test-vpc",
		CIDR: "10.0.0.0/16",
	}

	if vpc.CIDR != "10.0.0.0/16" {
		t.Errorf("Expected CIDR 10.0.0.0/16, got %s", vpc.CIDR)
	}
}

func TestSubnetStructure(t *testing.T) {
	subnet := Subnet{
		ID:               "subnet-test123",
		VPCID:            "vpc-test123",
		CIDR:             "10.0.1.0/24",
		AvailabilityZone: "us-east-1a",
		Name:             "test-subnet",
	}

	if subnet.CIDR != "10.0.1.0/24" {
		t.Errorf("Expected subnet CIDR 10.0.1.0/24, got %s", subnet.CIDR)
	}

	if subnet.AvailabilityZone != "us-east-1a" {
		t.Errorf("Expected AZ us-east-1a, got %s", subnet.AvailabilityZone)
	}
}

func TestRouteTableStructure(t *testing.T) {
	rt := RouteTable{
		ID:    "rtb-test123",
		VPCID: "vpc-test123",
		Name:  "test-rt",
		Main:  true,
		Routes: []Route{
			{
				DestinationCIDR: "10.0.0.0/16",
				Target:          "local",
				TargetType:      "local",
			},
		},
	}

	if !rt.Main {
		t.Error("Expected route table to be main")
	}

	if len(rt.Routes) != 1 {
		t.Errorf("Expected 1 route, got %d", len(rt.Routes))
	}

	if rt.Routes[0].TargetType != "local" {
		t.Errorf("Expected target type local, got %s", rt.Routes[0].TargetType)
	}
}

func TestTGWRouteTableStructure(t *testing.T) {
	tgwRT := TGWRouteTable{
		ID:               "tgw-rtb-test123",
		TransitGatewayID: "tgw-test123",
		Name:             "test-tgw-rt",
		DefaultTable:     true,
		Routes: []TGWRoute{
			{
				DestinationCIDR: "10.0.0.0/16",
				AttachmentID:    "tgw-attach-test123",
				RouteType:       "static",
				State:           "active",
			},
		},
	}

	if !tgwRT.DefaultTable {
		t.Error("Expected TGW route table to be default")
	}

	if len(tgwRT.Routes) != 1 {
		t.Errorf("Expected 1 route, got %d", len(tgwRT.Routes))
	}

	if tgwRT.Routes[0].State != "active" {
		t.Errorf("Expected route state active, got %s", tgwRT.Routes[0].State)
	}
}
