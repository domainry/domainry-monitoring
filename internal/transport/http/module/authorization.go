package module

import (
	"strings"
	"time"

	identitysdk "github.com/domainry/domainry-identity-sdk"
)

// hasUnrestrictedPermission requires both halves of one exact Permission:
// its function grant and the data policy for the same resource/action. A
// Monitoring metrics snapshot aggregates the whole Runtime and has no row,
// owner, or organization facts that could safely implement a narrower scope,
// so only canonical `all` is meaningful. Every narrower or unrecognized
// policy fails closed at the resource-owning handler boundary.
func hasUnrestrictedPermission(principal identitysdk.Principal, permission string, now time.Time) bool {
	permission = strings.TrimSpace(permission)
	separator := strings.LastIndexByte(permission, '.')
	if !principal.Known || separator <= 0 || separator == len(permission)-1 ||
		!principal.HasPermission(permission) || principal.AccessBundle == nil {
		return false
	}
	bundle := principal.AccessBundle
	if err := bundle.Validate(now.UTC()); err != nil ||
		strings.TrimSpace(string(bundle.Subject.WorkspaceID)) != strings.TrimSpace(principal.WorkspaceID) ||
		strings.TrimSpace(string(bundle.Subject.SubjectID)) != strings.TrimSpace(principal.UserID) {
		return false
	}
	resource, action := permission[:separator], permission[separator+1:]
	unrestricted := false
	for _, policy := range bundle.DataPolicies {
		if strings.TrimSpace(string(policy.Resource)) != resource || strings.TrimSpace(string(policy.Action)) != action {
			continue
		}
		if policy.Effect != identitysdk.EffectAllow {
			// A singleton aggregate has no resource facts with which to evaluate a
			// deny predicate without risking an authorization expansion.
			return false
		}
		for _, scope := range policy.DataScopes {
			switch scope {
			case identitysdk.DataScopeAll:
				unrestricted = true
			case identitysdk.DataScopeOwner, identitysdk.DataScopeOrg, identitysdk.DataScopeOrgChild, identitysdk.DataScopeTargetOrg:
				// These scopes need record ownership or organization facts. The
				// aggregated Runtime snapshot deliberately has neither.
			default:
				return false
			}
		}
	}
	for _, guardrail := range bundle.Guardrails {
		resourceMatches := guardrail.Resource == "" || guardrail.Resource == "*" || strings.TrimSpace(string(guardrail.Resource)) == resource
		actionMatches := guardrail.Action == "" || guardrail.Action == "*" || strings.TrimSpace(string(guardrail.Action)) == action
		if resourceMatches && actionMatches && guardrail.Effect == identitysdk.EffectDeny {
			// The untyped aggregate cannot safely evaluate record predicates or
			// field restrictions. Unconditional denies are also authoritative.
			return false
		}
	}
	return unrestricted
}
