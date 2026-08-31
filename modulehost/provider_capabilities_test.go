package modulehost

import (
	"testing"

	"github.com/domainry/domainry-notification-sdk/contract"
)

func TestDefaultProviderCapabilitiesAreSortedAndComplete(t *testing.T) {
	if value, found := DefaultProviderCapabilityFor(" whatsapp ", " meta_cloud_api "); !found || !value.SupportsProviderTemplate {
		t.Fatalf("WhatsApp capability=%+v found=%v", value, found)
	}
	values := DefaultProviderCapabilities()
	if len(values) != 12 {
		t.Fatalf("capability count=%d", len(values))
	}
	for index := 1; index < len(values); index++ {
		previous, current := values[index-1], values[index]
		if previous.Channel > current.Channel || previous.Channel == current.Channel && previous.Provider > current.Provider {
			t.Fatalf("capabilities are not sorted at %d: %#v then %#v", index, previous, current)
		}
	}
	if values[0].Channel != "collaboration" || values[len(values)-1].Provider != "meta_cloud_api" {
		t.Fatalf("unexpected capability boundaries: %#v ... %#v", values[0], values[len(values)-1])
	}
}

func TestDefaultProviderTemplateValidatorOwnsWhatsAppRules(t *testing.T) {
	validator := DefaultProviderTemplateValidator{}
	valid := contract.NotificationProviderTemplate{
		Name:       "order_update",
		Language:   "zh_CN",
		Components: []contract.NotificationProviderTemplateComponent{{Type: "body", Parameters: []string{"order_id"}}},
	}
	if err := validator.ValidateProviderTemplate("whatsapp", "meta_cloud_api", valid); err != nil {
		t.Fatalf("valid template rejected: %v", err)
	}
	if err := validator.ValidateProviderTemplate("whatsapp", "another", valid); err == nil {
		t.Fatal("unsupported provider accepted")
	}
	valid.Components[0].Parameters = []string{""}
	if err := validator.ValidateProviderTemplate("whatsapp", "meta_cloud_api", valid); err == nil {
		t.Fatal("empty provider parameter accepted")
	}
}

func TestDefaultTemplateCapabilityCatalogBuilds(t *testing.T) {
	if _, err := DefaultTemplateCapabilityCatalog(); err != nil {
		t.Fatal(err)
	}
}
