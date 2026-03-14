package resources

var ECS = &ResourceDef{
	TFType:        "aws_ecs_service",
	DisplayName:   "ECS Service",
	Category:      "Compute",
	TFRefAttr:     "id",
	TFOutputAttr:  "",
	ParentRefAttr: "cluster",
	DefaultProps: map[string]interface{}{
		"desired_count":   1,
		"launch_type":     "FARGATE",
		"task_definition": "",
	},
	PropSchema: []PropDef{
		{Key: "desired_count", Label: "Desired Count", Type: PropTypeInt},
		{Key: "launch_type", Label: "Launch Type", Type: PropTypeString},
		{Key: "task_definition", Label: "Task Definition", Type: PropTypeString, Required: true},
	},
}
