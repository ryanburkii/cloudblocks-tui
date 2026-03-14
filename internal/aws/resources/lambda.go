package resources

var Lambda = &ResourceDef{
	TFType:       "aws_lambda_function",
	DisplayName:  "Lambda Function",
	Category:     "Compute",
	TFRefAttr:    "arn",
	TFOutputAttr: "arn",
	DefaultProps: map[string]interface{}{
		"runtime":     "python3.11",
		"handler":     "index.handler",
		"memory_size": 128,
	},
	PropSchema: []PropDef{
		{Key: "runtime", Label: "Runtime", Type: PropTypeString, Required: true},
		{Key: "handler", Label: "Handler", Type: PropTypeString, Required: true},
		{Key: "memory_size", Label: "Memory (MB)", Type: PropTypeInt},
	},
}
