// internal/aws/resources/types.go
package resources

// PropType determines how a property value is rendered in the TUI editor
// and serialized in Terraform HCL.
type PropType string

const (
	PropTypeString PropType = "string" // HCL value is quoted
	PropTypeInt    PropType = "int"    // HCL value is unquoted
	PropTypeBool   PropType = "bool"   // HCL value is unquoted
)

// PropDef describes one configurable property of an AWS resource.
type PropDef struct {
	Key      string
	Label    string
	Type     PropType
	Required bool
}

// ResourceDef describes an AWS resource type — its Terraform type, display
// name, default properties, and how it participates in HCL generation.
type ResourceDef struct {
	TFType        string                 // Terraform resource type, e.g. "aws_vpc"
	DisplayName   string                 // Human-readable name shown in the catalog
	Category      string                 // Catalog grouping, e.g. "Networking"
	DefaultProps  map[string]interface{} // Pre-filled property values for new nodes
	PropSchema    []PropDef              // Ordered list of editable properties

	// ParentRefAttr is the HCL attribute on THIS resource that references its
	// parent when an edge exists (e.g. "vpc_id" for Subnet). Empty if no
	// cross-reference should be emitted.
	ParentRefAttr string

	// TFRefAttr is the trailing HCL attribute used when ANOTHER resource
	// references this one (e.g. "id" for VPC/Subnet, "arn" for Lambda).
	// Defaults to "id" if empty.
	TFRefAttr string

	// TFOutputAttr is the attribute exposed in outputs.tf (e.g. "id", "arn",
	// "bucket"). Empty means no output block is emitted for this resource type.
	TFOutputAttr string
}
