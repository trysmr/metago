package salesforceurl

import (
	"fmt"
	"net/url"
	"strings"
)

type Builder struct {
	baseURL string
}

func NewBuilder(baseURL string) (Builder, error) {
	normalized := strings.TrimSpace(baseURL)
	parsed, err := url.Parse(normalized)
	if err != nil {
		return Builder{}, fmt.Errorf("Salesforce組織のURLを解析できません: %w", err)
	}

	if parsed.Scheme != "https" || parsed.Host == "" {
		return Builder{}, fmt.Errorf("Salesforce組織のURLにはhttpsから始まるURLを指定してください")
	}
	if parsed.User != nil {
		return Builder{}, fmt.Errorf("Salesforce組織のURLに認証情報は指定できません")
	}

	if parsed.Path != "" && parsed.Path != "/" {
		return Builder{}, fmt.Errorf("Salesforce組織のURLにパスは指定できません")
	}

	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return Builder{}, fmt.Errorf("Salesforce組織のURLにクエリやフラグメントは指定できません")
	}

	return Builder{
		baseURL: strings.TrimRight(normalized, "/"),
	}, nil
}

func (builder Builder) ObjectDetails(objectAPIName string) string {
	return builder.objectManagerURL(objectAPIName, "Details")
}

func (builder Builder) ObjectFields(objectAPIName string) string {
	return builder.objectManagerURL(objectAPIName, "FieldsAndRelationships")
}

func (builder Builder) objectManagerURL(objectAPIName string, section string) string {
	return fmt.Sprintf(
		"%s/lightning/setup/ObjectManager/%s/%s/view",
		builder.baseURL,
		url.PathEscape(objectAPIName),
		section,
	)
}
