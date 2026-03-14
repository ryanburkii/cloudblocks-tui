package resources

var S3 = &ResourceDef{
	TFType:       "aws_s3_bucket",
	DisplayName:  "S3",
	Category:     "Storage",
	TFRefAttr:    "bucket",
	TFOutputAttr: "bucket",
	DefaultProps: map[string]interface{}{
		"bucket":        "",
		"force_destroy": false,
	},
	PropSchema: []PropDef{
		{Key: "bucket", Label: "Bucket Name", Type: PropTypeString, Required: true},
		{Key: "force_destroy", Label: "Force Destroy", Type: PropTypeBool},
	},
}
