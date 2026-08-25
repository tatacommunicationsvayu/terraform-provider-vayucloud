// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package security_group

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
)

var _ datasource.DataSource = &VirtualMachineListSecurityGroupRulesDataSource{}

// NewVirtualMachineListSecurityGroupRulesDataSource lists SGs and rules on a VM.
func NewVirtualMachineListSecurityGroupRulesDataSource() datasource.DataSource {
	return &VirtualMachineListSecurityGroupRulesDataSource{}
}

// VirtualMachineListSecurityGroupRulesDataSource implements
// vayucloud_virtualmachine_list_security_group_rules.
type VirtualMachineListSecurityGroupRulesDataSource struct {
	client *client.Client
}

// VirtualMachineListSecurityGroupRulesDataSourceModel is the Terraform model.
//
// The API returns one row per (port, security_group) with repeated port fields.
// This schema groups by port so Terraform consumers see:
//
//	ports[].security_groups[].rules[]
type VirtualMachineListSecurityGroupRulesDataSourceModel struct {
	ID         types.String               `tfsdk:"id"`
	InstanceID types.Int64                `tfsdk:"instance_id"`
	Filters    []client.FilterModel       `tfsdk:"filter"`
	Ports      []VMSecurityGroupPortModel `tfsdk:"ports"`
}

// VMSecurityGroupPortModel is one VM NIC/port with its attached security groups.
type VMSecurityGroupPortModel struct {
	PortID         types.String                      `tfsdk:"port_id"`
	PortName       types.String                      `tfsdk:"port_name"`
	PortIP         types.String                      `tfsdk:"port_ip"`
	SecurityGroups []VMPortSecurityGroupModel        `tfsdk:"security_groups"`
}

// VMPortSecurityGroupModel is one security group attached to a port.
type VMPortSecurityGroupModel struct {
	ID    types.String               `tfsdk:"id"`
	Name  types.String               `tfsdk:"name"`
	Rules []VMSecurityGroupRuleModel `tfsdk:"rules"`
}

// VMSecurityGroupRuleModel is one rule under a VM-attached security group.
type VMSecurityGroupRuleModel struct {
	ID             types.String `tfsdk:"id"`
	Protocol       types.String `tfsdk:"protocol"`
	EtherType      types.String `tfsdk:"ether_type"`
	Direction      types.String `tfsdk:"direction"`
	PortRangeMin   types.Int64  `tfsdk:"port_range_min"`
	PortRangeMax   types.Int64  `tfsdk:"port_range_max"`
	RemoteIPPrefix types.String `tfsdk:"remote_ip_prefix"`
	RemoteGroupID  types.String `tfsdk:"remote_group_id"`
	NormalizedCIDR types.String `tfsdk:"normalized_cidr"`
	Description    types.String `tfsdk:"description"`
	CreatedAt      types.String `tfsdk:"created_at"`
	UpdatedAt      types.String `tfsdk:"updated_at"`
}

func (d *VirtualMachineListSecurityGroupRulesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_virtualmachine_list_security_group_rules"
}

func (d *VirtualMachineListSecurityGroupRulesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	ruleAttrs := map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Description: "Rule UUID (`rule_id` from the API).",
			Computed:    true,
		},
		"protocol": schema.StringAttribute{
			Description: "Protocol (null when unrestricted).",
			Computed:    true,
		},
		"ether_type": schema.StringAttribute{
			Description: "Ether type (IPv4 / IPv6).",
			Computed:    true,
		},
		"direction": schema.StringAttribute{
			Description: "Direction (ingress / egress).",
			Computed:    true,
		},
		"port_range_min": schema.Int64Attribute{
			Description: "Minimum port (null when not set).",
			Computed:    true,
		},
		"port_range_max": schema.Int64Attribute{
			Description: "Maximum port (null when not set).",
			Computed:    true,
		},
		"remote_ip_prefix": schema.StringAttribute{
			Description: "Remote CIDR (null when using a remote group).",
			Computed:    true,
		},
		"remote_group_id": schema.StringAttribute{
			Description: "Remote security group UUID (null when using a CIDR).",
			Computed:    true,
		},
		"normalized_cidr": schema.StringAttribute{
			Description: "Normalized remote CIDR from the API.",
			Computed:    true,
		},
		"description": schema.StringAttribute{
			Description: "Rule description.",
			Computed:    true,
		},
		"created_at": schema.StringAttribute{
			Description: "Creation timestamp (RFC3339).",
			Computed:    true,
		},
		"updated_at": schema.StringAttribute{
			Description: "Last update timestamp (RFC3339).",
			Computed:    true,
		},
	}

	resp.Schema = schema.Schema{
		Description: "Lists security groups and rules associated with a virtual machine, grouped by port.",
		MarkdownDescription: "Reads `GET .../security-group/instance/{instance_id}` and reshapes the " +
			"flat API rows into a clearer hierarchy:\n\n" +
			"```\n" +
			"ports[]\n" +
			"  ├─ port_id / port_name / port_ip\n" +
			"  └─ security_groups[]\n" +
			"       ├─ id / name\n" +
			"       └─ rules[]\n" +
			"```\n\n" +
			"Optionally, use `filter` blocks to narrow results client-side. " +
			"Filters match against each port+security-group attachment using: " +
			"`port_id`, `port_name`, `port_ip`, `security_group_id`, `security_group_name`.",
		Blocks: map[string]schema.Block{
			"filter": client.FilterBlockSchema(),
		},
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Placeholder identifier (`instance_id`).",
				Computed:    true,
			},
			"instance_id": schema.Int64Attribute{
				Description: "Virtual machine instance ID.",
				Required:    true,
			},
			"ports": schema.ListNestedAttribute{
				Description: "VM ports with their attached security groups and rules.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"port_id": schema.StringAttribute{
							Description: "Neutron/port UUID.",
							Computed:    true,
						},
						"port_name": schema.StringAttribute{
							Description: "Port display name.",
							Computed:    true,
						},
						"port_ip": schema.StringAttribute{
							Description: "Primary IP on the port.",
							Computed:    true,
						},
						"security_groups": schema.ListNestedAttribute{
							Description: "Security groups attached to this port.",
							Computed:    true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"id": schema.StringAttribute{
										Description: "Security group UUID.",
										Computed:    true,
									},
									"name": schema.StringAttribute{
										Description: "Security group name.",
										Computed:    true,
									},
									"rules": schema.ListNestedAttribute{
										Description: "Rules in this security group (may be empty).",
										Computed:    true,
										NestedObject: schema.NestedAttributeObject{
											Attributes: ruleAttrs,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (d *VirtualMachineListSecurityGroupRulesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T.", req.ProviderData),
		)
		return
	}
	d.client = c
}

func (d *VirtualMachineListSecurityGroupRulesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data VirtualMachineListSecurityGroupRulesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	instanceID := data.InstanceID.ValueInt64()
	tflog.Debug(ctx, "Reading virtualmachine_list_security_group_rules data source", map[string]any{
		"instance_id": instanceID,
	})

	entries, err := ListInstanceSecurityGroups(d.client, ctx, instanceID)
	if err != nil {
		resp.Diagnostics.AddError("Error Listing Instance Security Groups", err.Error())
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%d", instanceID))

	filteredEntries := entries
	if len(data.Filters) > 0 {
		attachments := make([]vmSecurityGroupAttachmentFilterModel, 0, len(entries))
		for _, entry := range entries {
			attachments = append(attachments, vmSecurityGroupAttachmentFilterModel{
				PortID:            types.StringValue(entry.PortID),
				PortName:          types.StringValue(entry.PortName),
				PortIP:            types.StringValue(entry.PortIP),
				SecurityGroupID:   types.StringValue(entry.SecurityGroupID),
				SecurityGroupName: types.StringValue(entry.SecurityGroupName),
			})
		}

		tflog.Debug(ctx, "Applying filters to instance security group attachments", map[string]any{
			"filter_count":     len(data.Filters),
			"pre_filter_count": len(attachments),
		})

		filteredAttachments, filterErr := client.ApplyFilters(attachments, data.Filters)
		if filterErr != nil {
			resp.Diagnostics.AddError("Error Filtering Instance Security Groups", filterErr.Error())
			return
		}

		keep := make(map[string]struct{}, len(filteredAttachments))
		for _, a := range filteredAttachments {
			keep[attachmentKey(a.PortID.ValueString(), a.SecurityGroupID.ValueString())] = struct{}{}
		}
		filteredEntries = make([]InstanceSecurityGroupEntry, 0, len(filteredAttachments))
		for _, entry := range entries {
			if _, ok := keep[attachmentKey(entry.PortID, entry.SecurityGroupID)]; ok {
				filteredEntries = append(filteredEntries, entry)
			}
		}

		tflog.Debug(ctx, "Filters applied to instance security group attachments", map[string]any{
			"post_filter_count": len(filteredEntries),
		})
	}

	data.Ports = groupInstanceSecurityGroupsByPort(filteredEntries)

	tflog.Info(ctx, "Read virtualmachine_list_security_group_rules data source", map[string]any{
		"instance_id": instanceID,
		"api_rows":    len(entries),
		"ports":       len(data.Ports),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// vmSecurityGroupAttachmentFilterModel is the flat filter surface for each
// API (port, security_group) row before grouping into ports[].
type vmSecurityGroupAttachmentFilterModel struct {
	PortID            types.String `tfsdk:"port_id"`
	PortName          types.String `tfsdk:"port_name"`
	PortIP            types.String `tfsdk:"port_ip"`
	SecurityGroupID   types.String `tfsdk:"security_group_id"`
	SecurityGroupName types.String `tfsdk:"security_group_name"`
}

func attachmentKey(portID, sgID string) string {
	return portID + "|" + sgID
}

// groupInstanceSecurityGroupsByPort collapses repeated API rows into ports → SGs → rules.
func groupInstanceSecurityGroupsByPort(entries []InstanceSecurityGroupEntry) []VMSecurityGroupPortModel {
	order := make([]string, 0)
	ports := map[string]*VMSecurityGroupPortModel{}

	for _, entry := range entries {
		key := entry.PortID
		if key == "" {
			key = strings.Join([]string{entry.PortName, entry.PortIP}, "|")
		}
		port, ok := ports[key]
		if !ok {
			order = append(order, key)
			m := VMSecurityGroupPortModel{
				PortID:         types.StringValue(entry.PortID),
				PortName:       types.StringValue(entry.PortName),
				PortIP:         types.StringValue(entry.PortIP),
				SecurityGroups: []VMPortSecurityGroupModel{},
			}
			ports[key] = &m
			port = ports[key]
		}

		rules := make([]VMSecurityGroupRuleModel, 0, len(entry.Rules))
		for _, rule := range entry.Rules {
			rules = append(rules, VMSecurityGroupRuleModel{
				ID:             types.StringValue(rule.RuleID),
				Protocol:       optionalString(rule.Protocol),
				EtherType:      types.StringValue(rule.EtherType),
				Direction:      types.StringValue(rule.Direction),
				PortRangeMin:   optionalInt64(rule.PortRangeMin),
				PortRangeMax:   optionalInt64(rule.PortRangeMax),
				RemoteIPPrefix: optionalString(rule.RemoteIPPrefix),
				RemoteGroupID:  optionalString(rule.RemoteGroupID),
				NormalizedCIDR: optionalString(rule.NormalizedCIDR),
				Description:    types.StringValue(rule.Description),
				CreatedAt:      types.StringValue(rule.CreatedAt),
				UpdatedAt:      types.StringValue(rule.UpdatedAt),
			})
		}

		port.SecurityGroups = append(port.SecurityGroups, VMPortSecurityGroupModel{
			ID:    types.StringValue(entry.SecurityGroupID),
			Name:  types.StringValue(entry.SecurityGroupName),
			Rules: rules,
		})
	}

	out := make([]VMSecurityGroupPortModel, 0, len(order))
	for _, key := range order {
		out = append(out, *ports[key])
	}
	return out
}

func optionalString(v *string) types.String {
	if v == nil || *v == "" {
		return types.StringNull()
	}
	return types.StringValue(*v)
}

func optionalInt64(v *int64) types.Int64 {
	if v == nil {
		return types.Int64Null()
	}
	return types.Int64Value(*v)
}
