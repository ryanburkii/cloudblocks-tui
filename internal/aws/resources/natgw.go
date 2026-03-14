package resources

var NatGW = &ResourceDef{
	TFType:        "aws_nat_gateway",
	DisplayName:   "NAT Gateway",
	Category:      "Networking",
	TFRefAttr:     "id",
	TFOutputAttr:  "id",
	ParentRefAttr: "subnet_id",
	DefaultProps: map[string]interface{}{
		"connectivity_type": "public",
	},
	PropSchema: []PropDef{
		{Key: "connectivity_type", Label: "Connectivity", Type: PropTypeString},
	},
}
