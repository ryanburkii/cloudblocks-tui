package resources

var Subnet = &ResourceDef{
	TFType:        "aws_subnet",
	DisplayName:   "Subnet",
	Category:      "Networking",
	TFRefAttr:     "id",
	TFOutputAttr:  "id",
	ParentRefAttr: "vpc_id",
	DefaultProps: map[string]interface{}{
		"cidr_block":        "10.0.1.0/24",
		"availability_zone": "us-east-1a",
	},
	PropSchema: []PropDef{
		{Key: "cidr_block", Label: "CIDR Block", Type: PropTypeString, Required: true},
		{Key: "availability_zone", Label: "AZ", Type: PropTypeString},
	},
}
