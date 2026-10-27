// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package security_group

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestAlignPlanRuleIDsByFingerprint_newRuleGetsUnknownID(t *testing.T) {
	ctx := context.Background()
	prefix, _ := types.ListValueFrom(ctx, types.StringType, []string{"0.0.0.0/0"})

	plan := SecurityGroupResourceModel{
		Rules: []SecurityGroupRuleModel{
			{
				Protocol:       types.StringValue("tcp"),
				EtherType:      types.StringValue("ipv4"),
				Direction:      types.StringValue("ingress"),
				PortRange:      types.Int64Value(80),
				RemoteIPPrefix: prefix,
			},
			{
				Protocol:       types.StringValue("udp"),
				EtherType:      types.StringValue("ipv4"),
				Direction:      types.StringValue("ingress"),
				PortRange:      types.Int64Value(80),
				RemoteIPPrefix: prefix,
			},
		},
	}
	stateRules := []SecurityGroupRuleModel{
		{
			ID:             types.StringValue("existing-tcp-rule"),
			Protocol:       types.StringValue("tcp"),
			EtherType:      types.StringValue("ipv4"),
			Direction:      types.StringValue("ingress"),
			PortRange:      types.Int64Value(80),
			RemoteIPPrefix: prefix,
		},
	}

	if !alignPlanRuleIDsByFingerprint(&plan, stateRules) {
		t.Fatal("expected plan to change")
	}
	if plan.Rules[0].ID.ValueString() != "existing-tcp-rule" {
		t.Fatalf("tcp rule id: got %q", plan.Rules[0].ID.ValueString())
	}
	if !plan.Rules[1].ID.IsUnknown() {
		t.Fatalf("new udp rule id should be unknown, got %#v", plan.Rules[1].ID)
	}
}
