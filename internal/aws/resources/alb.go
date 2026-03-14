package resources

// ALB has no ParentRefAttr: subnet association uses a list (not supported
// by the single-value cross-reference generator in V1). Connect ALB to
// a subnet visually but the HCL cross-reference is not auto-generated.
var ALB = &ResourceDef{
	TFType:       "aws_lb",
	DisplayName:  "Application Load Balancer",
	Category:     "Load Balancing",
	TFRefAttr:    "id",
	TFOutputAttr: "id",
	DefaultProps: map[string]interface{}{
		"internal":           false,
		"load_balancer_type": "application",
	},
	PropSchema: []PropDef{
		{Key: "internal", Label: "Internal", Type: PropTypeBool},
		{Key: "load_balancer_type", Label: "Type", Type: PropTypeString},
	},
}
