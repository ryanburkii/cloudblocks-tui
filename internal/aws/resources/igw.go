package resources

var IGW = &ResourceDef{
	TFType:        "aws_internet_gateway",
	DisplayName:   "Internet Gateway",
	Category:      "Networking",
	TFRefAttr:     "id",
	TFOutputAttr:  "id",
	ParentRefAttr: "vpc_id",
	DefaultProps:  map[string]interface{}{},
	PropSchema:    []PropDef{},
}
