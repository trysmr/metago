package salesforceurl

import "testing"

func TestBuilderBuildsObjectManagerURLs(t *testing.T) {
	builder, err := NewBuilder(" https://example.my.salesforce.com/ ")
	if err != nil {
		t.Fatal(err)
	}

	if got, want := builder.ObjectDetails("Invoice__c"), "https://example.my.salesforce.com/lightning/setup/ObjectManager/Invoice__c/Details/view"; got != want {
		t.Errorf("詳細URL = %q, want %q", got, want)
	}
	if got, want := builder.ObjectFields("Invoice__c"), "https://example.my.salesforce.com/lightning/setup/ObjectManager/Invoice__c/FieldsAndRelationships/view"; got != want {
		t.Errorf("項目URL = %q, want %q", got, want)
	}
}

func TestNewBuilderRejectsUnsupportedBaseURL(t *testing.T) {
	for _, baseURL := range []string{
		"http://example.my.salesforce.com",
		"https:///missing-host",
		"https://example.my.salesforce.com/setup",
	} {
		if _, err := NewBuilder(baseURL); err == nil {
			t.Errorf("NewBuilder(%q)はエラーを返しませんでした", baseURL)
		}
	}
}

func TestNewBuilderRejectsURLWithQueryOrFragment(t *testing.T) {
	for _, baseURL := range []string{
		"https://example.my.salesforce.com?source=docs",
		"https://example.my.salesforce.com#objects",
	} {
		if _, err := NewBuilder(baseURL); err == nil {
			t.Errorf("NewBuilder(%q)はエラーを返しませんでした", baseURL)
		}
	}
}

func TestNewBuilderRejectsURLWithUserInformation(t *testing.T) {
	_, err := NewBuilder("https://user:password@example.my.salesforce.com")
	if err == nil {
		t.Fatal("認証情報を含むURLを受け付けました")
	}
}
