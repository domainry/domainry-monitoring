package module

import (
	"strings"
	"testing"
	"time"

	identitysdk "github.com/domainry/domainry-identity-sdk"
	monitoringsdk "github.com/domainry/domainry-monitoring-sdk"
)

func TestMetricsSnapshotAcceptsOnlyExactPermissionWithAllDataScope(t *testing.T) {
	now := time.Date(2026, time.September, 3, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name               string
		functionPermission string
		dataPermission     string
		scope              identitysdk.DataScope
		want               bool
	}{
		{name: "exact all", functionPermission: monitoringsdk.ActionMonitoringMetricsRead, dataPermission: monitoringsdk.ActionMonitoringMetricsRead, scope: identitysdk.DataScopeAll, want: true},
		{name: "function only", functionPermission: monitoringsdk.ActionMonitoringMetricsRead},
		{name: "data only", dataPermission: monitoringsdk.ActionMonitoringMetricsRead, scope: identitysdk.DataScopeAll},
		{name: "different function key", functionPermission: "monitoring.metrics.list", dataPermission: monitoringsdk.ActionMonitoringMetricsRead, scope: identitysdk.DataScopeAll},
		{name: "different data key", functionPermission: monitoringsdk.ActionMonitoringMetricsRead, dataPermission: "monitoring.metrics.list", scope: identitysdk.DataScopeAll},
		{name: "owner cannot filter aggregate", functionPermission: monitoringsdk.ActionMonitoringMetricsRead, dataPermission: monitoringsdk.ActionMonitoringMetricsRead, scope: identitysdk.DataScopeOwner},
		{name: "org cannot filter aggregate", functionPermission: monitoringsdk.ActionMonitoringMetricsRead, dataPermission: monitoringsdk.ActionMonitoringMetricsRead, scope: identitysdk.DataScopeOrg},
		{name: "org child cannot filter aggregate", functionPermission: monitoringsdk.ActionMonitoringMetricsRead, dataPermission: monitoringsdk.ActionMonitoringMetricsRead, scope: identitysdk.DataScopeOrgChild},
		{name: "target org cannot filter aggregate", functionPermission: monitoringsdk.ActionMonitoringMetricsRead, dataPermission: monitoringsdk.ActionMonitoringMetricsRead, scope: identitysdk.DataScopeTargetOrg},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			principal := metricsPrincipal(now, test.functionPermission, test.dataPermission, test.scope)
			if got := hasUnrestrictedPermission(principal, monitoringsdk.ActionMonitoringMetricsRead, now); got != test.want {
				t.Fatalf("hasUnrestrictedPermission()=%t want=%t", got, test.want)
			}
		})
	}
}

func TestMetricsSnapshotFailsClosedForUnusablePolicyState(t *testing.T) {
	now := time.Date(2026, time.September, 3, 0, 0, 0, 0, time.UTC)
	valid := metricsPrincipal(now, monitoringsdk.ActionMonitoringMetricsRead, monitoringsdk.ActionMonitoringMetricsRead, identitysdk.DataScopeAll)
	tests := []struct {
		name   string
		mutate func(*identitysdk.Principal)
	}{
		{name: "expired bundle", mutate: func(value *identitysdk.Principal) { value.AccessBundle.ExpiresAt = now }},
		{name: "workspace mismatch", mutate: func(value *identitysdk.Principal) { value.AccessBundle.Subject.WorkspaceID = "other-workspace" }},
		{name: "subject mismatch", mutate: func(value *identitysdk.Principal) { value.AccessBundle.Subject.SubjectID = "other-user" }},
		{name: "matching deny cannot be evaluated", mutate: func(value *identitysdk.Principal) {
			value.AccessBundle.DataPolicies = append(value.AccessBundle.DataPolicies, identitysdk.DataPolicy{
				Key: "deny-monitoring-owner", Resource: "monitoring.metrics", Action: "read", Effect: identitysdk.EffectDeny,
				DataScopes: []identitysdk.DataScope{identitysdk.DataScopeOwner},
				Predicate:  identitysdk.Predicate{Fact: "owner_id", Operator: identitysdk.OperatorEqual, Value: "$subject.id"},
			})
		}},
		{name: "matching guardrail cannot be evaluated", mutate: func(value *identitysdk.Principal) {
			predicate := identitysdk.Predicate{Fact: "owner_id", Operator: identitysdk.OperatorEqual, Value: "$subject.id"}
			value.AccessBundle.Guardrails = append(value.AccessBundle.Guardrails, identitysdk.Guardrail{
				Key: "monitoring-owner-deny", Resource: "monitoring.metrics", Action: "read", Effect: identitysdk.EffectDeny, Predicate: &predicate,
			})
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			principal := cloneMetricsPrincipal(valid)
			test.mutate(&principal)
			if hasUnrestrictedPermission(principal, monitoringsdk.ActionMonitoringMetricsRead, now) {
				t.Fatal("unusable policy state authorized")
			}
		})
	}
}

func metricsPrincipal(now time.Time, functionPermission, dataPermission string, scope identitysdk.DataScope) identitysdk.Principal {
	bundle := identitysdk.AccessBundle{
		ContractVersion: identitysdk.CurrentPolicyBundleVersion, AuthorizationRevision: "monitoring-test-authorization",
		ExpiresAt: now.Add(time.Hour), Subject: identitysdk.Subject{WorkspaceID: "workspace", SubjectID: "user", OrgID: "org", OrgScopeIDs: []string{"org", "child"}, SupportOrgScopeIDs: []string{"support"}},
	}
	if resource, action, ok := permissionParts(functionPermission); ok {
		bundle.FunctionGrants = append(bundle.FunctionGrants, identitysdk.FunctionGrant{Resource: identitysdk.ResourceType(resource), Action: identitysdk.Action(action), Effect: identitysdk.EffectAllow})
	}
	if resource, action, ok := permissionParts(dataPermission); ok {
		policy := identitysdk.DataPolicy{Key: "data-" + strings.ReplaceAll(dataPermission, ".", "-"), Resource: identitysdk.ResourceType(resource), Action: identitysdk.Action(action), Effect: identitysdk.EffectAllow}
		if scope != "" {
			policy.DataScopes = []identitysdk.DataScope{scope}
			if scope != identitysdk.DataScopeAll {
				policy.Predicate = identitysdk.Predicate{Fact: "owner_id", Operator: identitysdk.OperatorEqual, Value: "$subject.id"}
			}
		}
		bundle.DataPolicies = append(bundle.DataPolicies, policy)
	}
	return identitysdk.Principal{ContractVersion: identitysdk.PrincipalContextContractVersion, Known: true, WorkspaceID: "workspace", UserID: "user", AccessBundle: &bundle}
}

func permissionParts(permission string) (string, string, bool) {
	permission = strings.TrimSpace(permission)
	separator := strings.LastIndexByte(permission, '.')
	if separator <= 0 || separator == len(permission)-1 {
		return "", "", false
	}
	return permission[:separator], permission[separator+1:], true
}

func cloneMetricsPrincipal(source identitysdk.Principal) identitysdk.Principal {
	clone := source
	bundle := *source.AccessBundle
	bundle.FunctionGrants = append([]identitysdk.FunctionGrant(nil), source.AccessBundle.FunctionGrants...)
	bundle.DataPolicies = append([]identitysdk.DataPolicy(nil), source.AccessBundle.DataPolicies...)
	bundle.Guardrails = append([]identitysdk.Guardrail(nil), source.AccessBundle.Guardrails...)
	clone.AccessBundle = &bundle
	return clone
}
