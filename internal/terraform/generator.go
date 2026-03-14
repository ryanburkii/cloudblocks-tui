// internal/terraform/generator.go
package terraform

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"cloudblocks-tui/internal/aws/resources"
	"cloudblocks-tui/internal/graph"
)

// Generate writes main.tf, variables.tf, and outputs.tf to outDir.
// lookup resolves a Terraform resource type string to its ResourceDef.
func Generate(arch *graph.Architecture, lookup func(string) *resources.ResourceDef, outDir string) error {
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}
	main, vars, outputs, err := buildHCL(arch, lookup)
	if err != nil {
		return err
	}
	files := map[string]string{
		"main.tf":      main,
		"variables.tf": vars,
		"outputs.tf":   outputs,
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(outDir, name), []byte(content), 0644); err != nil {
			return fmt.Errorf("write %s: %w", name, err)
		}
	}
	return nil
}

func buildHCL(arch *graph.Architecture, lookup func(string) *resources.ResourceDef) (mainOut, varsOut, outputsOut string, err error) {
	var main, outputs strings.Builder

	main.WriteString(`terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region  = var.aws_region
  profile = "default"
}

`)

	// Build parent index: for each "to" node, list its parent IDs.
	parents := make(map[string][]string)
	for _, e := range arch.Edges {
		parents[e.To] = append(parents[e.To], e.From)
	}

	for _, id := range arch.NodeOrder {
		n, ok := arch.Nodes[id]
		if !ok {
			continue
		}
		def := lookup(n.Type)

		main.WriteString(fmt.Sprintf("resource %q %q {\n", n.Type, n.ID))

		// Build prop-type index for this resource.
		propTypes := make(map[string]resources.PropType)
		if def != nil {
			for _, pd := range def.PropSchema {
				propTypes[pd.Key] = pd.Type
			}
		}

		// Write properties in sorted order for determinism.
		keys := make([]string, 0, len(n.Properties))
		for k := range n.Properties {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			main.WriteString(fmt.Sprintf("  %s = %s\n", k, formatHCL(n.Properties[k], propTypes[k])))
		}

		// Write cross-references from parent edges.
		if def != nil && def.ParentRefAttr != "" {
			for _, parentID := range parents[id] {
				parent, ok := arch.Nodes[parentID]
				if !ok {
					continue
				}
				parentDef := lookup(parent.Type)
				refAttr := "id"
				if parentDef != nil && parentDef.TFRefAttr != "" {
					refAttr = parentDef.TFRefAttr
				}
				main.WriteString(fmt.Sprintf("  %s = %s.%s.%s\n",
					def.ParentRefAttr, parent.Type, parent.ID, refAttr))
			}
		}

		main.WriteString("}\n\n")

		// Emit output block if this resource type declares a TFOutputAttr.
		if def != nil && def.TFOutputAttr != "" {
			outputs.WriteString(fmt.Sprintf("output %q {\n  value = %s.%s.%s\n}\n\n",
				n.ID, n.Type, n.ID, def.TFOutputAttr))
		}
	}

	varsOut = `variable "aws_region" {
  description = "AWS region"
  type        = string
  default     = "us-east-1"
}`
	return main.String(), varsOut, outputs.String(), nil
}

// formatHCL formats v as an HCL literal.
// PropTypeString → quoted; PropTypeInt / PropTypeBool → unquoted.
func formatHCL(v interface{}, pt resources.PropType) string {
	switch pt {
	case resources.PropTypeInt, resources.PropTypeBool:
		return fmt.Sprintf("%v", v)
	default:
		return fmt.Sprintf("%q", fmt.Sprintf("%v", v))
	}
}
