package contract

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var (
	templateStableKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9_.-]*$`)
	templateTokenPattern     = regexp.MustCompile(`\{\{\s*([a-z][a-z0-9_.-]*)\s*\}\}`)
)

var templateSupportedVariableTypes = map[string]bool{
	"boolean": true, "date": true, "datetime": true, "email": true,
	"number": true, "text": true, "url": true,
}

const templateFallbackLimit = 5

// TemplateValidationError is the stable validation failure exposed at both
// Module and Remote boundaries. Params contain only wire-safe scalar values.
type TemplateValidationError struct {
	Code   string            `json:"code"`
	Params map[string]string `json:"params,omitempty"`
}

func (e *TemplateValidationError) Error() string {
	if e == nil {
		return ""
	}
	return e.Code
}

func invalidTemplate(code string, params ...string) error {
	values := map[string]string{}
	for index := 0; index+1 < len(params); index += 2 {
		if key := strings.TrimSpace(params[index]); key != "" {
			values[key] = params[index+1]
		}
	}
	if len(values) == 0 {
		values = nil
	}
	return &TemplateValidationError{Code: code, Params: values}
}

type ProviderTemplateValidation func(*NotificationProviderTemplate) error

type NotificationTemplateProvider struct {
	Capability               NotificationTemplateCapability
	ValidateProviderTemplate ProviderTemplateValidation
}

type notificationTemplateCapabilityEntry struct {
	capability NotificationTemplateCapability
	validate   ProviderTemplateValidation
}

// NotificationTemplateCapabilities is immutable after construction so a
// publication cannot observe provider rules changing during validation.
type NotificationTemplateCapabilities struct {
	entries map[string]notificationTemplateCapabilityEntry
}

func NewNotificationTemplateCapabilities(providers []NotificationTemplateProvider) (*NotificationTemplateCapabilities, error) {
	entries := make(map[string]notificationTemplateCapabilityEntry, len(providers))
	for _, provider := range providers {
		capability := provider.Capability
		capability.Channel = strings.TrimSpace(capability.Channel)
		capability.Provider = strings.TrimSpace(capability.Provider)
		if capability.Channel == "" || capability.MaxFacts < 0 || capability.MaxActions < 0 {
			return nil, fmt.Errorf("notification template capability is invalid")
		}
		if capability.SupportsProviderTemplate && provider.ValidateProviderTemplate == nil {
			return nil, fmt.Errorf("notification provider-template validator is required for %q", templateCapabilityKey(capability.Channel, capability.Provider))
		}
		key := templateCapabilityKey(capability.Channel, capability.Provider)
		if _, exists := entries[key]; exists {
			return nil, fmt.Errorf("notification template capability %q is duplicated", key)
		}
		entries[key] = notificationTemplateCapabilityEntry{capability: capability, validate: provider.ValidateProviderTemplate}
	}
	return &NotificationTemplateCapabilities{entries: entries}, nil
}

func (c *NotificationTemplateCapabilities) resolve(channel, provider string) (notificationTemplateCapabilityEntry, bool) {
	if c == nil {
		return notificationTemplateCapabilityEntry{}, false
	}
	entry, found := c.entries[templateCapabilityKey(channel, provider)]
	return entry, found
}

func templateCapabilityKey(channel, provider string) string {
	return strings.TrimSpace(channel) + "/" + strings.TrimSpace(provider)
}

type NotificationTemplateValidator struct {
	capabilities *NotificationTemplateCapabilities
}

func NewNotificationTemplateValidator(capabilities *NotificationTemplateCapabilities) (*NotificationTemplateValidator, error) {
	if capabilities == nil {
		return nil, fmt.Errorf("notification template capabilities are required")
	}
	return &NotificationTemplateValidator{capabilities: capabilities}, nil
}

func (v *NotificationTemplateValidator) ValidateAll(templates []NotificationTemplate) error {
	seen := map[string]bool{}
	for _, value := range templates {
		if seen[value.Key] {
			return invalidTemplate("backend.notification.template_key_duplicate", "template_key", value.Key)
		}
		seen[value.Key] = true
	}
	for index := range templates {
		if err := v.Validate(templates[index]); err != nil {
			return fmt.Errorf("notification_templates[%d]: %w", index, err)
		}
		for _, fallback := range templates[index].Fallbacks {
			if !seen[fallback.TemplateKey] {
				return invalidTemplate("backend.notification.fallback_template_not_found", "template_key", templates[index].Key, "fallback_template_key", fallback.TemplateKey)
			}
		}
	}
	return nil
}

func (v *NotificationTemplateValidator) Validate(value NotificationTemplate) error {
	if v == nil || v.capabilities == nil {
		return invalidTemplate("backend.notification.template_provider_unsupported")
	}
	if !templateStableKeyPattern.MatchString(strings.TrimSpace(value.Key)) {
		return invalidTemplate("backend.notification.template_key_invalid", "template_key", value.Key)
	}
	if strings.TrimSpace(value.Name) == "" {
		return invalidTemplate("backend.notification.template_name_required", "template_key", value.Key)
	}
	channel, provider := strings.TrimSpace(value.Channel), strings.TrimSpace(value.Provider)
	entry, known := v.capabilities.resolve(channel, provider)
	if !known {
		return invalidTemplate("backend.notification.template_provider_unsupported", "channel", channel, "provider", provider)
	}
	if strings.TrimSpace(value.Status) != "published" {
		return invalidTemplate("backend.notification.template_status_invalid", "status", value.Status)
	}
	if value.Version < 1 {
		return invalidTemplate("backend.notification.template_version_invalid", "template_key", value.Key)
	}
	defaultLocale := strings.TrimSpace(value.DefaultLocale)
	if defaultLocale == "" {
		return invalidTemplate("backend.notification.template_default_locale_required", "template_key", value.Key)
	}
	if len(value.Locales) == 0 {
		return invalidTemplate("backend.notification.template_locales_required", "template_key", value.Key)
	}
	if _, found := value.Locales[defaultLocale]; !found {
		return invalidTemplate("backend.notification.template_default_locale_missing", "template_key", value.Key, "locale", defaultLocale)
	}
	variables, err := validateTemplateVariables(value)
	if err != nil {
		return err
	}
	if err := validateTemplateFallbacks(value); err != nil {
		return err
	}
	locales := make([]string, 0, len(value.Locales))
	for locale := range value.Locales {
		locales = append(locales, locale)
	}
	sort.Strings(locales)
	for _, locale := range locales {
		if err := validateTemplateContent(value, locale, value.Locales[locale], variables, entry); err != nil {
			return err
		}
	}
	return nil
}

func (v *NotificationTemplateValidator) ValidateEditable(value NotificationTemplate) error {
	status := strings.TrimSpace(value.Status)
	if status != "draft" && status != "published" {
		return invalidTemplate("backend.notification.template_status_invalid", "status", value.Status)
	}
	value.Status, value.ContentHash = "published", ""
	return v.Validate(value)
}

func validateTemplateVariables(value NotificationTemplate) (map[string]NotificationTemplateVariable, error) {
	variables := make(map[string]NotificationTemplateVariable, len(value.Variables))
	for _, variable := range value.Variables {
		key := strings.TrimSpace(variable.Key)
		if !templateStableKeyPattern.MatchString(key) {
			return nil, invalidTemplate("backend.notification.template_variable_key_invalid", "template_key", value.Key, "variable", key)
		}
		if _, exists := variables[key]; exists {
			return nil, invalidTemplate("backend.notification.template_variable_duplicate", "template_key", value.Key, "variable", key)
		}
		variable.Key, variable.Type = key, strings.TrimSpace(variable.Type)
		if !templateSupportedVariableTypes[variable.Type] {
			return nil, invalidTemplate("backend.notification.template_variable_type_unsupported", "template_key", value.Key, "variable", key, "type", variable.Type)
		}
		variables[key] = variable
	}
	return variables, nil
}

func validateTemplateFallbacks(value NotificationTemplate) error {
	if len(value.Fallbacks) > templateFallbackLimit {
		return invalidTemplate("backend.notification.fallback_limit_exceeded", "template_key", value.Key)
	}
	seen := map[string]bool{}
	for _, fallback := range value.Fallbacks {
		target, connector := strings.TrimSpace(fallback.TemplateKey), strings.TrimSpace(fallback.ConnectorKey)
		connection, operation := strings.TrimSpace(fallback.ConnectionKey), strings.TrimSpace(fallback.Operation)
		if target == value.Key || !templateStableKeyPattern.MatchString(target) || !templateStableKeyPattern.MatchString(connector) || connection == "" || operation == "" {
			return invalidTemplate("backend.notification.fallback_invalid", "template_key", value.Key)
		}
		identity := connector + "\x00" + connection + "\x00" + operation + "\x00" + target
		if seen[identity] {
			return invalidTemplate("backend.notification.fallback_duplicate", "template_key", value.Key)
		}
		seen[identity] = true
	}
	return nil
}

func validateTemplateContent(value NotificationTemplate, locale string, content NotificationTemplateContent, variables map[string]NotificationTemplateVariable, entry notificationTemplateCapabilityEntry) error {
	capability := entry.capability
	if strings.TrimSpace(locale) == "" {
		return invalidTemplate("backend.notification.template_locale_invalid", "template_key", value.Key)
	}
	if capability.Channel == "email" && strings.TrimSpace(content.Subject) == "" {
		return invalidTemplate("backend.notification.template_subject_required", "template_key", value.Key, "locale", locale)
	}
	if strings.ContainsAny(content.Subject+content.Title, "\r\n") {
		return invalidTemplate("backend.notification.template_subject_invalid", "template_key", value.Key, "locale", locale)
	}
	if capability.Channel == "email" && strings.TrimSpace(content.Text) == "" && strings.TrimSpace(content.HTML) == "" || capability.Channel != "email" && strings.TrimSpace(content.Text) == "" && strings.TrimSpace(content.Markdown) == "" {
		return invalidTemplate("backend.notification.template_body_required", "template_key", value.Key, "locale", locale)
	}
	if strings.TrimSpace(content.HTML) != "" && !capability.SupportsHTML || strings.TrimSpace(content.Markdown) != "" && !capability.SupportsMarkdown {
		return invalidTemplate("backend.notification.template_channel_mismatch", "template_key", value.Key, "locale", locale)
	}
	if len(content.Facts) > capability.MaxFacts || len(content.Actions) > capability.MaxActions {
		return invalidTemplate("backend.notification.template_components_limit", "template_key", value.Key, "locale", locale)
	}
	sources := []string{content.Subject, content.Title, content.Text, content.HTML, content.Markdown}
	for _, fact := range content.Facts {
		if strings.TrimSpace(fact.Key) == "" || strings.TrimSpace(fact.Value) == "" || !capability.SupportsFacts {
			return invalidTemplate("backend.notification.template_fact_invalid", "template_key", value.Key, "locale", locale)
		}
		sources = append(sources, fact.Key, fact.Value)
	}
	for _, action := range content.Actions {
		style := strings.TrimSpace(action.Style)
		if strings.TrimSpace(action.Label) == "" || strings.TrimSpace(action.URL) == "" || !capability.SupportsURLActions || style != "" && style != "primary" && style != "secondary" && style != "danger" {
			return invalidTemplate("backend.notification.template_action_invalid", "template_key", value.Key, "locale", locale)
		}
		sources = append(sources, action.Label, action.URL)
	}
	if content.ProviderTemplate != nil {
		if !capability.SupportsProviderTemplate {
			return invalidTemplate("backend.notification.provider_template_unsupported", "template_key", value.Key, "locale", locale, "provider", value.Provider)
		}
		if len(content.Facts) > 0 || len(content.Actions) > 0 {
			return invalidTemplate("backend.notification.provider_template_components_conflict", "template_key", value.Key, "locale", locale)
		}
		if entry.validate != nil {
			if err := entry.validate(content.ProviderTemplate); err != nil {
				return invalidTemplate("backend.notification.provider_template_invalid", "template_key", value.Key, "locale", locale, "provider", value.Provider)
			}
		}
		for _, component := range content.ProviderTemplate.Components {
			sources = append(sources, component.Parameters...)
		}
	}
	for _, source := range sources {
		if err := validateTemplateTokens(value.Key, locale, source, variables); err != nil {
			return err
		}
	}
	return nil
}

func validateTemplateTokens(templateKey, locale, source string, variables map[string]NotificationTemplateVariable) error {
	matches := templateTokenPattern.FindAllStringSubmatch(source, -1)
	consumed := templateTokenPattern.ReplaceAllString(source, "")
	if strings.Contains(consumed, "{{") || strings.Contains(consumed, "}}") {
		return invalidTemplate("backend.notification.template_syntax_invalid", "template_key", templateKey, "locale", locale)
	}
	for _, match := range matches {
		root := strings.Split(match[1], ".")[0]
		if _, found := variables[root]; !found {
			return invalidTemplate("backend.notification.template_variable_unknown", "template_key", templateKey, "locale", locale, "variable", root)
		}
	}
	return nil
}

func NotificationTemplateContentHash(value NotificationTemplate) string {
	value.ContentHash = ""
	raw, _ := json.Marshal(value)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func NotificationValueHash(value any) string {
	raw, _ := json.Marshal(value)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
