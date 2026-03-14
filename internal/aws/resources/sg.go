package resources

var SecurityGroup = &ResourceDef{
	TFType:        "aws_security_group",
	DisplayName:   "Security Group",
	Category:      "Networking",
	TFRefAttr:     "id",
	TFOutputAttr:  "id",
	ParentRefAttr: "vpc_id",
	DefaultProps: map[string]interface{}{
		"name":        "my-sg",
		"description": "Managed by CloudBlocks",
	},
	PropSchema: []PropDef{
		{Key: "name", Label: "Name", Type: PropTypeString, Required: true},
		{Key: "description", Label: "Description", Type: PropTypeString},
	},
}
