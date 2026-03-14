// internal/catalog/catalog.go
package catalog

import "cloudblocks-tui/internal/aws/resources"

// All returns all resource definitions in catalog display order.
func All() []*resources.ResourceDef {
	return all
}

// ByCategory returns a stable category order and a map of category → resources.
func ByCategory() ([]string, map[string][]*resources.ResourceDef) {
	order := []string{"Networking", "Compute", "Databases", "Storage", "Load Balancing"}
	m := make(map[string][]*resources.ResourceDef)
	for _, r := range all {
		m[r.Category] = append(m[r.Category], r)
	}
	return order, m
}

// ByTFType returns the ResourceDef for the given Terraform type, or nil.
func ByTFType(tfType string) *resources.ResourceDef {
	for _, r := range all {
		if r.TFType == tfType {
			return r
		}
	}
	return nil
}

var all = []*resources.ResourceDef{
	resources.VPC,
	resources.Subnet,
	resources.IGW,
	resources.NatGW,
	resources.SecurityGroup,
	resources.EC2,
	resources.ECS,
	resources.Lambda,
	resources.RDS,
	resources.DynamoDB,
	resources.S3,
	resources.ALB,
}
