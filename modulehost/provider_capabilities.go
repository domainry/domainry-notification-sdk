package modulehost

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/domainry/domainry-notification-sdk/contract"
)

// DefaultProviderCapabilityFor resolves one entry from the Notification-owned
// default module-host binding.
func DefaultProviderCapabilityFor(channel, provider string) (contract.NotificationTemplateCapability, bool) {
	value, ok := defaultProviderCapabilities[strings.TrimSpace(channel)+"/"+strings.TrimSpace(provider)]
	return value, ok
}

// DefaultProviderCapabilities is the Notification-owned provider catalog used
// by an in-process module host when the project does not supply another
// deployment binding.
func DefaultProviderCapabilities() []contract.NotificationTemplateCapability {
	result := make([]contract.NotificationTemplateCapability, 0, len(defaultProviderCapabilities))
	for _, value := range defaultProviderCapabilities {
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Channel == result[j].Channel {
			return result[i].Provider < result[j].Provider
		}
		return result[i].Channel < result[j].Channel
	})
	return result
}

// DefaultTemplateCapabilityCatalog builds the immutable template validator
// catalog for the default Notification module-host binding.
func DefaultTemplateCapabilityCatalog() (*contract.NotificationTemplateCapabilities, error) {
	providers := make([]contract.NotificationTemplateProvider, 0, len(defaultProviderCapabilities))
	for _, value := range DefaultProviderCapabilities() {
		provider := contract.NotificationTemplateProvider{Capability: value}
		if value.SupportsProviderTemplate {
			provider.ValidateProviderTemplate = validateWhatsAppProviderTemplate
		}
		providers = append(providers, provider)
	}
	return contract.NewNotificationTemplateCapabilities(providers)
}

// DefaultProviderTemplateValidator exposes provider-template validation to the
// Notification module through the SDK host boundary.
type DefaultProviderTemplateValidator struct{}

func (DefaultProviderTemplateValidator) ValidateProviderTemplate(channel, provider string, value contract.NotificationProviderTemplate) error {
	if strings.TrimSpace(channel) != "whatsapp" || strings.TrimSpace(provider) != "meta_cloud_api" {
		return fmt.Errorf("unsupported Notification provider template")
	}
	return validateWhatsAppProviderTemplate(&value)
}

var defaultProviderCapabilities = map[string]contract.NotificationTemplateCapability{
	"email/":                          {Channel: "email", SupportsHTML: true, SupportsFacts: true, SupportsURLActions: true, MaxFacts: 10, MaxActions: 5},
	"whatsapp/meta_cloud_api":         {Channel: "whatsapp", Provider: "meta_cloud_api", SupportsFacts: true, SupportsURLActions: true, SupportsProviderTemplate: true, MaxFacts: 10, MaxActions: 5},
	"collaboration/feishu":            collaborationCapability("feishu", 5),
	"collaboration/dingtalk":          collaborationCapability("dingtalk", 5),
	"collaboration/enterprise_wechat": collaborationCapability("enterprise_wechat", 3),
	"collaboration/slack":             collaborationCapability("slack", 5),
	"collaboration/teams":             collaborationCapability("teams", 5),
	"collaboration/microsoft_365":     collaborationCapability("microsoft_365", 5),
	"collaboration/discord":           collaborationCapability("discord", 5),
	"collaboration/google_workspace":  collaborationCapability("google_workspace", 5),
	"collaboration/line":              collaborationCapability("line", 5),
	"collaboration/line_works":        collaborationCapability("line_works", 5),
}

func collaborationCapability(provider string, maxActions int) contract.NotificationTemplateCapability {
	return contract.NotificationTemplateCapability{Channel: "collaboration", Provider: provider, SupportsMarkdown: true, SupportsFacts: true, SupportsURLActions: true, MaxFacts: 10, MaxActions: maxActions}
}

var providerTemplateName = regexp.MustCompile(`^[a-z0-9_]+$`)
var providerTemplateLanguage = regexp.MustCompile(`^[a-z]{2,3}([_-][A-Z]{2})?$`)
var providerButtonIndex = regexp.MustCompile(`^[0-9]$`)

func validateWhatsAppProviderTemplate(value *contract.NotificationProviderTemplate) error {
	if value == nil || !providerTemplateName.MatchString(strings.TrimSpace(value.Name)) || !providerTemplateLanguage.MatchString(strings.TrimSpace(value.Language)) || len(value.Components) > 12 {
		return fmt.Errorf("invalid WhatsApp provider template")
	}
	for _, component := range value.Components {
		typeName, subtype, index := strings.TrimSpace(component.Type), strings.TrimSpace(component.SubType), strings.TrimSpace(component.Index)
		if (typeName == "header" || typeName == "body") && (subtype != "" || index != "") {
			return fmt.Errorf("invalid WhatsApp provider template component")
		}
		if typeName == "button" && ((subtype != "url" && subtype != "quick_reply") || !providerButtonIndex.MatchString(index)) {
			return fmt.Errorf("invalid WhatsApp provider template button")
		}
		if typeName != "header" && typeName != "body" && typeName != "button" {
			return fmt.Errorf("invalid WhatsApp provider template component")
		}
		if len(component.Parameters) == 0 || len(component.Parameters) > 10 {
			return fmt.Errorf("invalid WhatsApp provider template parameters")
		}
		for _, parameter := range component.Parameters {
			if strings.TrimSpace(parameter) == "" {
				return fmt.Errorf("invalid WhatsApp provider template parameter")
			}
		}
	}
	return nil
}

var _ ProviderTemplateValidator = DefaultProviderTemplateValidator{}
