// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// FilterModel represents a single filter block in a data source configuration.
//
// Usage in Terraform:
//
//	filter {
//	  name   = "field_name"
//	  values = ["value1", "value2"]
//	}
//
// Filter logic:
//   - Multiple filter blocks: AND (all filters must match)
//   - Multiple values within a filter: OR (any value can match)
//   - Matching: case-insensitive exact match
type FilterModel struct {
	// Name is the field name to filter on (must match a tfsdk tag in the model)
	Name types.String `tfsdk:"name"`

	// Values is the list of values to match against (OR logic, substring match)
	Values []types.String `tfsdk:"values"`
}

// FilterBlockSchema returns the reusable schema block definition for filters.
// Add this to any data source schema's Blocks map under the key "filter".
//
// Example:
//
//	resp.Schema = schema.Schema{
//	    Attributes: map[string]schema.Attribute{...},
//	    Blocks: map[string]schema.Block{
//	        "filter": FilterBlockSchema(),
//	    },
//	}
func FilterBlockSchema() schema.ListNestedBlock {
	return schema.ListNestedBlock{
		Description:         "One or more filter blocks to narrow down results.",
		MarkdownDescription: "One or more `filter` blocks to narrow down results. Multiple filters are combined with **AND** logic. Multiple values within a filter use **OR** logic with case-insensitive exact matching.",
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"name": schema.StringAttribute{
					Description:         "The name of the field to filter by. Must match a field name in the data source (e.g., 'engagement_name', 'id', 'service_name').",
					MarkdownDescription: "The name of the field to filter by. Must match a field name in the data source (e.g., `engagement_name`, `id`, `service_name`).",
					Required:            true,
				},
				"values": schema.ListAttribute{
					Description:         "The values to match against. An item matches if the field value equals any of these values (case-insensitive).",
					MarkdownDescription: "The values to match against. An item matches if the field value equals any of these values (case-insensitive).",
					Required:            true,
					ElementType:         types.StringType,
				},
			},
		},
	}
}

// ApplyFilters filters a slice of items based on the provided filter criteria.
// T must be a struct type with tfsdk tags on its fields.
//
// Filter logic:
//   - Multiple filters: AND (all filters must match for an item to be included)
//   - Multiple values in a filter: OR (any value can match)
//   - Matching: case-insensitive exact match
//
// Returns an error if a filter references a field name that doesn't exist in the struct.
func ApplyFilters[T any](items []T, filters []FilterModel) ([]T, error) {
	if len(filters) == 0 {
		return items, nil
	}

	// Validate filter names against the struct's tfsdk tags upfront
	if len(items) > 0 {
		validFields := getValidFieldNames[T]()
		for _, filter := range filters {
			filterName := filter.Name.ValueString()
			if _, ok := validFields[filterName]; !ok {
				return nil, fmt.Errorf("invalid filter name %q: valid filter names are %v", filterName, validFieldList(validFields))
			}
		}
	}

	var result []T
	for _, item := range items {
		match, err := matchesAllFilters(item, filters)
		if err != nil {
			return nil, err
		}
		if match {
			result = append(result, item)
		}
	}

	if result == nil {
		result = make([]T, 0)
	}

	return result, nil
}

// matchesAllFilters checks if a single item matches all filter criteria (AND logic).
func matchesAllFilters[T any](item T, filters []FilterModel) (bool, error) {
	for _, filter := range filters {
		fieldValue, err := getFieldValueByTag(item, filter.Name.ValueString())
		if err != nil {
			return false, err
		}
		if !matchesAnyValue(fieldValue, filter.Values) {
			return false, nil
		}
	}
	return true, nil
}

// matchesAnyValue checks if a field value matches any of the filter values (OR logic).
// Uses case-insensitive exact matching.
func matchesAnyValue(fieldValue string, values []types.String) bool {
	for _, v := range values {
		if strings.EqualFold(fieldValue, v.ValueString()) {
			return true
		}
	}
	return false
}

// getFieldValueByTag retrieves a struct field's string value by its tfsdk tag.
// Supports types.String, types.Int64, types.Float64, types.Bool, and types.Number.
func getFieldValueByTag[T any](item T, tagName string) (string, error) {
	v := reflect.ValueOf(item)
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("tfsdk")
		if tag == tagName {
			return fieldValueToString(v.Field(i))
		}
	}
	return "", fmt.Errorf("filter field %q not found in data source schema", tagName)
}

// fieldValueToString converts a Terraform framework type value to its string representation.
func fieldValueToString(v reflect.Value) (string, error) {
	iface := v.Interface()
	switch val := iface.(type) {
	case types.String:
		if val.IsNull() || val.IsUnknown() {
			return "", nil
		}
		return val.ValueString(), nil
	case types.Int64:
		if val.IsNull() || val.IsUnknown() {
			return "", nil
		}
		return fmt.Sprintf("%d", val.ValueInt64()), nil
	case types.Float64:
		if val.IsNull() || val.IsUnknown() {
			return "", nil
		}
		return fmt.Sprintf("%g", val.ValueFloat64()), nil
	case types.Bool:
		if val.IsNull() || val.IsUnknown() {
			return "", nil
		}
		return fmt.Sprintf("%t", val.ValueBool()), nil
	default:
		return fmt.Sprintf("%v", iface), nil
	}
}

// getValidFieldNames returns a map of valid tfsdk tag names for a struct type.
func getValidFieldNames[T any]() map[string]struct{} {
	var zero T
	t := reflect.TypeOf(zero)
	fields := make(map[string]struct{}, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		tag := t.Field(i).Tag.Get("tfsdk")
		if tag != "" && tag != "-" {
			fields[tag] = struct{}{}
		}
	}
	return fields
}

// validFieldList returns a sorted comma-separated list of valid field names for error messages.
func validFieldList(fields map[string]struct{}) []string {
	list := make([]string, 0, len(fields))
	for k := range fields {
		list = append(list, k)
	}
	return list
}
