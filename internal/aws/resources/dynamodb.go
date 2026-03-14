package resources

var DynamoDB = &ResourceDef{
	TFType:       "aws_dynamodb_table",
	DisplayName:  "DynamoDB",
	Category:     "Databases",
	TFRefAttr:    "id",
	TFOutputAttr: "id",
	DefaultProps: map[string]interface{}{
		"billing_mode": "PAY_PER_REQUEST",
		"hash_key":     "id",
	},
	PropSchema: []PropDef{
		{Key: "billing_mode", Label: "Billing Mode", Type: PropTypeString},
		{Key: "hash_key", Label: "Hash Key", Type: PropTypeString, Required: true},
	},
}
