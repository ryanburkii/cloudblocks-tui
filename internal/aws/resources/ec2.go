package resources

var EC2 = &ResourceDef{
	TFType:        "aws_instance",
	DisplayName:   "EC2 Instance",
	Category:      "Compute",
	TFRefAttr:     "id",
	TFOutputAttr:  "id",
	ParentRefAttr: "subnet_id",
	DefaultProps: map[string]interface{}{
		"ami":           "ami-0c55b159cbfafe1f0",
		"instance_type": "t3.micro",
	},
	PropSchema: []PropDef{
		{Key: "ami", Label: "AMI", Type: PropTypeString, Required: true},
		{Key: "instance_type", Label: "Instance Type", Type: PropTypeString, Required: true},
	},
}
