package resources

var RDS = &ResourceDef{
	TFType:        "aws_db_instance",
	DisplayName:   "RDS",
	Category:      "Databases",
	TFRefAttr:     "id",
	TFOutputAttr:  "id",
	ParentRefAttr: "db_subnet_group_name",
	DefaultProps: map[string]interface{}{
		"engine":            "mysql",
		"instance_class":    "db.t3.micro",
		"allocated_storage": 20,
		"username":          "admin",
		"password":          "changeme",
	},
	PropSchema: []PropDef{
		{Key: "engine", Label: "Engine", Type: PropTypeString, Required: true},
		{Key: "instance_class", Label: "Instance Class", Type: PropTypeString, Required: true},
		{Key: "allocated_storage", Label: "Storage (GB)", Type: PropTypeInt},
		{Key: "username", Label: "Username", Type: PropTypeString},
		{Key: "password", Label: "Password", Type: PropTypeString},
	},
}
