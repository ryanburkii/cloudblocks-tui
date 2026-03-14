package resources

var VPC = &ResourceDef{
	TFType:       "aws_vpc",
	DisplayName:  "VPC",
	Category:     "Networking",
	TFRefAttr:    "id",
	TFOutputAttr: "id",
	DefaultProps: map[string]interface{}{
		"cidr_block":           "10.0.0.0/16",
		"enable_dns_hostnames": true,
	},
	PropSchema: []PropDef{
		{Key: "cidr_block", Label: "CIDR Block", Type: PropTypeString, Required: true},
		{Key: "enable_dns_hostnames", Label: "DNS Hostnames", Type: PropTypeBool},
	},
}
