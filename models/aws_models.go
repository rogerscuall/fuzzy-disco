package models

// TransitGateway represents an AWS Transit Gateway
type TransitGateway struct {
	ID                    string
	Name                  string
	AmazonSideASN         int64
	DefaultRouteTableID   string
	Attachments           []TGWAttachment
	RouteTables           []TGWRouteTable
	State                 string
}

// TGWAttachment represents a Transit Gateway attachment
type TGWAttachment struct {
	ID                  string
	TransitGatewayID    string
	ResourceType        string // vpc, vpn, direct-connect-gateway
	ResourceID          string // VPC ID, VPN ID, etc.
	State               string
	RouteTableID        string // Associated route table
}

// TGWRouteTable represents a Transit Gateway route table
type TGWRouteTable struct {
	ID               string
	TransitGatewayID string
	Name             string
	DefaultTable     bool
	Routes           []TGWRoute
	Associations     []string // Attachment IDs
	Propagations     []string // Attachment IDs
}

// TGWRoute represents a route in a TGW route table
type TGWRoute struct {
	DestinationCIDR string
	AttachmentID    string
	RouteType       string // static, propagated
	State           string
}

// VPC represents an AWS VPC
type VPC struct {
	ID              string
	Name            string
	CIDR            string
	Subnets         []Subnet
	RouteTables     []RouteTable
	TGWAttachmentID string
}

// Subnet represents a VPC subnet
type Subnet struct {
	ID               string
	VPCID            string
	CIDR             string
	AvailabilityZone string
	Name             string
	RouteTableID     string
}

// RouteTable represents a VPC route table
type RouteTable struct {
	ID      string
	VPCID   string
	Name    string
	Routes  []Route
	Subnets []string // Subnet IDs associated with this route table
	Main    bool
}

// Route represents a route in a VPC route table
type Route struct {
	DestinationCIDR string
	Target          string // local, tgw-id, igw-id, nat-id, etc.
	TargetType      string // local, transit-gateway, internet-gateway, nat-gateway
}

// AWSEnvironment represents the complete AWS environment
type AWSEnvironment struct {
	TransitGateways []TransitGateway
	VPCs            []VPC
}
