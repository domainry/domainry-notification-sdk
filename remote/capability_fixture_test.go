package remote_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	actioncontract "github.com/domainry/domainry-foundation/action"
	"github.com/domainry/domainry-foundation/modulecapability"
)

func remoteTestHandler(t testing.TB, next http.Handler) http.Handler {
	t.Helper()
	binding := remoteTestCapabilityBinding(t)
	capability, err := modulecapability.NewHTTPHandler(binding, func(*http.Request) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if strings.HasPrefix(request.URL.Path, modulecapability.HTTPPrefix+"/") {
			capability.ServeHTTP(response, request)
			return
		}
		next.ServeHTTP(response, request)
	})
}

func remoteTestCapabilitySHA256(t testing.TB) string {
	t.Helper()
	summary, err := remoteTestCapabilityBinding(t).CapabilitySummary(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return summary.Identity.ContractSHA256
}

func remoteTestCapabilityBinding(t testing.TB) *modulecapability.StaticBinding {
	t.Helper()
	operation, err := json.Marshal(map[string]any{
		"operationId": "getNotificationTestCapability",
		"responses":   map[string]any{"200": map[string]any{"description": "test"}},
		modulecapability.OperationExtensionKey: modulecapability.OperationExtension{
			Owner: "notification", Authorization: modulecapability.Authorization{Strategy: actioncontract.AuthorizationAuthenticatedPrincipal, WorkspaceScope: "authenticated_workspace_principal"},
			Effect: modulecapability.EffectRead, Idempotency: modulecapability.Idempotency{Mode: "not_applicable"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	binding, err := modulecapability.NewStaticBinding(modulecapability.ModuleSummary{
		Identity: modulecapability.ModuleIdentity{Key: "notification", SourceOwner: "notification", ModuleVersion: "test", ValidationRevision: "test-v1", SupportedDeploymentModes: []modulecapability.DeploymentMode{modulecapability.DeploymentModeModule, modulecapability.DeploymentModeSaaS}},
		Name:     "Notification", Description: "Remote test Notification capability contract.",
		Scenarios: modulecapability.AdaptationScenarios{
			UseWhen: []string{"test notification"}, DoNotUseWhen: []string{"not a notification"}, RequirementSignals: []string{"notify"}, ProvidedCapabilities: []string{"notification.test"},
			RequiredModules: []string{}, OptionalModules: []string{}, ConflictingModules: []string{}, AssemblyChains: []string{"test_chain"}, ValidationScopes: []string{"notification.test"},
			SelectionExamples: []modulecapability.ScenarioExample{{Requirement: "notify", Reason: "test"}}, RejectionExamples: []modulecapability.ScenarioExample{{Requirement: "store", Reason: "test"}},
		},
	}, []modulecapability.CategoryDocument{{
		Category: modulecapability.CategorySummary{Key: "notification.test", Name: "Test", Description: "Test category.", OperationCount: 1, AssemblyChains: []string{"test_chain"}, ValidationScopes: []string{"notification.test"}},
		OpenAPI:  modulecapability.OpenAPIFragment{OpenAPI: "3.1.0", Paths: map[string]map[string]json.RawMessage{"/notification-test": {"get": operation}}},
		ValidationContracts: []modulecapability.ValidationScopeContract{{
			Kind: "notification.test", Description: "Validate one test Notification authoring fragment.", Coverage: modulecapability.ValidationCoverageExplicit, CandidateCollections: []string{"notifications"},
		}},
	}}, func(context.Context, modulecapability.ValidationRequest) (modulecapability.ValidationResult, error) {
		return modulecapability.ValidationResult{Diagnostics: []modulecapability.Diagnostic{}}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return binding
}
