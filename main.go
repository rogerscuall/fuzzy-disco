package main

import (
	"fmt"
	"log"
	"os"

	"github.com/rogerscuall/fuzzy-disco/aws"
	"github.com/rogerscuall/fuzzy-disco/models"
	"github.com/rogerscuall/fuzzy-disco/neo4j"
	"github.com/rogerscuall/fuzzy-disco/packetwalk"
)

func main() {
	fmt.Println("AWS Transit Gateway to Neo4j Graph Database")
	fmt.Println("Packet Walk Demonstration")
	fmt.Println()

	// Get Neo4j connection details from environment variables or use defaults
	neo4jURI := getEnv("NEO4J_URI", "bolt://localhost:7687")
	neo4jUser := getEnv("NEO4J_USER", "neo4j")
	neo4jPassword := getEnv("NEO4J_PASSWORD", "ecology-cafe-laptop-galileo-mobile-9640")

	// Create Neo4j client
	fmt.Println("Connecting to Neo4j...")
	client, err := neo4j.NewNeo4jClient(neo4jURI, neo4jUser, neo4jPassword)
	if err != nil {
		log.Fatalf("Failed to connect to Neo4j: %v", err)
	}
	defer client.Close()
	fmt.Println("✓ Connected to Neo4j")
	fmt.Println()

	// Get mock AWS environment
	fmt.Println("Loading AWS environment data...")
	env := aws.GetMockAWSEnvironment()
	fmt.Printf("✓ Loaded %d Transit Gateway(s)\n", len(env.TransitGateways))
	fmt.Printf("✓ Loaded %d VPC(s)\n", len(env.VPCs))
	fmt.Println()

	// Display environment summary
	displayEnvironmentSummary(env)

	// Load data into Neo4j
	fmt.Println("Loading AWS environment into Neo4j graph database...")
	if err := client.LoadAWSEnvironment(env); err != nil {
		log.Fatalf("Failed to load environment: %v", err)
	}
	fmt.Println("✓ AWS environment loaded into Neo4j")
	fmt.Println()

	// Create packet walker
	walker := packetwalk.NewWalker(client)

	// Demonstrate different packet walk scenarios
	runPacketWalkScenarios(walker)
}

func displayEnvironmentSummary(env *models.AWSEnvironment) {
	fmt.Println("=== AWS Environment Summary ===")
	fmt.Println()

	for _, tgw := range env.TransitGateways {
		fmt.Printf("Transit Gateway: %s (%s)\n", tgw.Name, tgw.ID)
		fmt.Printf("  ASN: %d\n", tgw.AmazonSideASN)
		fmt.Printf("  State: %s\n", tgw.State)
		fmt.Printf("  Attachments: %d\n", len(tgw.Attachments))
		fmt.Printf("  Route Tables: %d\n", len(tgw.RouteTables))
		fmt.Println()

		fmt.Println("  Route Tables:")
		for _, rt := range tgw.RouteTables {
			fmt.Printf("    - %s (%s)\n", rt.Name, rt.ID)
			fmt.Printf("      Default: %v\n", rt.DefaultTable)
			fmt.Printf("      Routes: %d\n", len(rt.Routes))
			for _, route := range rt.Routes {
				fmt.Printf("        • %s -> %s (%s)\n",
					route.DestinationCIDR, route.AttachmentID, route.RouteType)
			}
		}
		fmt.Println()
	}

	fmt.Println("VPCs:")
	for _, vpc := range env.VPCs {
		fmt.Printf("  - %s (%s)\n", vpc.Name, vpc.ID)
		fmt.Printf("    CIDR: %s\n", vpc.CIDR)
		fmt.Printf("    Subnets: %d\n", len(vpc.Subnets))
		fmt.Printf("    Route Tables: %d\n", len(vpc.RouteTables))
		fmt.Printf("    TGW Attachment: %s\n", vpc.TGWAttachmentID)
		fmt.Println()
	}
	fmt.Println()
}

func runPacketWalkScenarios(walker *packetwalk.Walker) {
	fmt.Println("=== PACKET WALK SCENARIOS ===")
	fmt.Println()

	scenarios := []struct {
		name        string
		description string
		sourceCIDR  string
		destCIDR    string
	}{
		{
			name:        "Scenario 1: Production to Development (via Firewall)",
			description: "Packet from Production VPC to Development VPC through central Firewall VPC",
			sourceCIDR:  "10.1.1.0/24",
			destCIDR:    "10.2.1.0/24",
		},
		{
			name:        "Scenario 2: Development to Shared Services (via Firewall)",
			description: "Packet from Development VPC to Shared Services VPC through central Firewall VPC",
			sourceCIDR:  "10.2.1.0/24",
			destCIDR:    "10.3.1.0/24",
		},
		{
			name:        "Scenario 3: Production to Shared Services (via Firewall)",
			description: "Packet from Production VPC to Shared Services VPC through central Firewall VPC",
			sourceCIDR:  "10.1.1.0/24",
			destCIDR:    "10.3.1.0/24",
		},
		{
			name:        "Scenario 4: Shared Services to Production (via Firewall)",
			description: "Return traffic from Shared Services VPC to Production VPC",
			sourceCIDR:  "10.3.1.0/24",
			destCIDR:    "10.1.1.0/24",
		},
	}

	for i, scenario := range scenarios {
		fmt.Printf("\n%s\n", scenario.name)
		fmt.Printf("Description: %s\n", scenario.description)
		fmt.Printf("Source: %s -> Destination: %s\n\n", scenario.sourceCIDR, scenario.destCIDR)

		walk, err := walker.WalkFromSubnetToSubnet(scenario.sourceCIDR, scenario.destCIDR)
		if err != nil {
			log.Printf("Error performing packet walk: %v", err)
			continue
		}

		packetwalk.PrintWalk(walk)

		if i < len(scenarios)-1 {
			fmt.Println("\n" + "Press Enter to continue to next scenario...")
			fmt.Scanln()
		}
	}

	fmt.Println("\n=== ALL SCENARIOS COMPLETED ===")
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
