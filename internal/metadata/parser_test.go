package metadata

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadObjectsParsesObjectFieldsFormulaAndReferences(t *testing.T) {
	objectsDirectory := t.TempDir()
	objectDirectory := filepath.Join(objectsDirectory, "Invoice__c")
	fieldsDirectory := filepath.Join(objectDirectory, "fields")
	if err := os.MkdirAll(fieldsDirectory, 0o755); err != nil {
		t.Fatal(err)
	}

	objectXML := `<?xml version="1.0" encoding="UTF-8"?>
<CustomObject xmlns="http://soap.sforce.com/2006/04/metadata">
  <fullName>Invoice__c</fullName>
  <label>請求書</label>
  <pluralLabel>請求書</pluralLabel>
  <deploymentStatus>Deployed</deploymentStatus>
  <sharingModel>ReadWrite</sharingModel>
  <enableActivities>true</enableActivities>
  <enableReports>true</enableReports>
  <enableHistory>false</enableHistory>
</CustomObject>`
	if err := os.WriteFile(
		filepath.Join(objectDirectory, "Invoice__c.object-meta.xml"),
		[]byte(objectXML),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	fieldXML := `<?xml version="1.0" encoding="UTF-8"?>
<CustomField xmlns="http://soap.sforce.com/2006/04/metadata">
  <fullName>Account__c</fullName>
  <label>取引先</label>
  <type>Lookup</type>
  <referenceTo>Account</referenceTo>
  <referenceTo>PersonAccount</referenceTo>
  <relationshipName>Invoices</relationshipName>
  <formula>TODAY() - DATEVALUE(CreatedDate)</formula>
</CustomField>`
	if err := os.WriteFile(
		filepath.Join(fieldsDirectory, "Account__c.field-meta.xml"),
		[]byte(fieldXML),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	objects, err := LoadObjects(objectsDirectory)
	if err != nil {
		t.Fatal(err)
	}
	if len(objects) != 1 {
		t.Fatalf("オブジェクト数 = %d, want 1", len(objects))
	}

	object := objects[0]
	if object.APIName != "Invoice__c" || object.Label != "請求書" ||
		!object.EnableActivities || !object.EnableReports || object.EnableHistory {
		t.Errorf("オブジェクト定義 = %#v", object)
	}
	if len(object.Fields) != 1 {
		t.Fatalf("項目数 = %d, want 1", len(object.Fields))
	}

	field := object.Fields[0]
	if field.APIName != "Account__c" || field.Relationship != "Invoices" ||
		field.Formula != "TODAY() - DATEVALUE(CreatedDate)" {
		t.Errorf("項目定義 = %#v", field)
	}
	if got := field.ReferenceTo; len(got) != 2 || got[0] != "Account" || got[1] != "PersonAccount" {
		t.Errorf("参照先 = %#v", got)
	}
}

func TestLoadObjectsUsesDirectoryNameWhenObjectFileIsMissing(t *testing.T) {
	objectsDirectory := t.TempDir()
	if err := os.MkdirAll(filepath.Join(objectsDirectory, "Account"), 0o755); err != nil {
		t.Fatal(err)
	}

	objects, err := LoadObjects(objectsDirectory)
	if err != nil {
		t.Fatal(err)
	}
	if len(objects) != 1 || objects[0].APIName != "Account" {
		t.Errorf("オブジェクト定義 = %#v", objects)
	}
}

func TestLoadObjectsParsesNumericAndBooleanFieldAttributes(t *testing.T) {
	objectsDirectory := t.TempDir()
	fieldsDirectory := filepath.Join(objectsDirectory, "Invoice__c", "fields")
	if err := os.MkdirAll(fieldsDirectory, 0o755); err != nil {
		t.Fatal(err)
	}

	amountXML := `<?xml version="1.0" encoding="UTF-8"?>
<CustomField xmlns="http://soap.sforce.com/2006/04/metadata">
  <fullName>Amount__c</fullName>
  <label>金額</label>
  <type>Number</type>
  <required>true</required>
  <unique>true</unique>
  <externalId>true</externalId>
  <trackHistory>true</trackHistory>
  <precision>18</precision>
  <scale>2</scale>
</CustomField>`
	if err := os.WriteFile(
		filepath.Join(fieldsDirectory, "Amount__c.field-meta.xml"),
		[]byte(amountXML),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	secretXML := `<?xml version="1.0" encoding="UTF-8"?>
<CustomField xmlns="http://soap.sforce.com/2006/04/metadata">
  <fullName>Secret__c</fullName>
  <label>秘密</label>
  <type>EncryptedText</type>
  <encrypted>true</encrypted>
  <length>100</length>
</CustomField>`
	if err := os.WriteFile(
		filepath.Join(fieldsDirectory, "Secret__c.field-meta.xml"),
		[]byte(secretXML),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	objects, err := LoadObjects(objectsDirectory)
	if err != nil {
		t.Fatal(err)
	}
	if len(objects) != 1 || len(objects[0].Fields) != 2 {
		t.Fatalf("解析結果 = %#v", objects)
	}

	amount := objects[0].Fields[0]
	if amount.APIName != "Amount__c" || amount.Label != "金額" || amount.Type != "Number" ||
		!amount.Required || !amount.Unique || !amount.ExternalID || !amount.TrackHistory {
		t.Errorf("数値項目 = %#v", amount)
	}
	if amount.Precision == nil || *amount.Precision != 18 || amount.Scale == nil || *amount.Scale != 2 {
		t.Errorf("数値項目の桁数 = %#v", amount)
	}

	secret := objects[0].Fields[1]
	if secret.APIName != "Secret__c" || secret.Type != "EncryptedText" || !secret.Encrypted {
		t.Errorf("暗号化項目 = %#v", secret)
	}
	if secret.Length == nil || *secret.Length != 100 {
		t.Errorf("暗号化項目の長さ = %#v", secret)
	}
}

func TestLoadObjectsRejectsWrongXMLRoot(t *testing.T) {
	objectsDirectory := t.TempDir()
	objectDirectory := filepath.Join(objectsDirectory, "Broken__c")
	if err := os.MkdirAll(objectDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(objectDirectory, "Broken__c.object-meta.xml"),
		[]byte(`<CustomField xmlns="http://soap.sforce.com/2006/04/metadata"/>`),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	if _, err := LoadObjects(objectsDirectory); err == nil {
		t.Fatal("CustomFieldをオブジェクト定義として受け付けました")
	}
}

func TestLoadObjectsRejectsDuplicateObjectAPINames(t *testing.T) {
	objectsDirectory := t.TempDir()
	for index, directoryName := range []string{"First__c", "Second__c"} {
		objectDirectory := filepath.Join(objectsDirectory, directoryName)
		if err := os.MkdirAll(objectDirectory, 0o755); err != nil {
			t.Fatal(err)
		}

		apiName := []string{"Duplicate__c", "duplicate__c"}[index]
		objectXML := fmt.Sprintf(
			`<CustomObject xmlns="http://soap.sforce.com/2006/04/metadata"><fullName>%s</fullName></CustomObject>`,
			apiName,
		)
		if err := os.WriteFile(
			filepath.Join(objectDirectory, directoryName+".object-meta.xml"),
			[]byte(objectXML),
			0o644,
		); err != nil {
			t.Fatal(err)
		}
	}

	_, err := LoadObjects(objectsDirectory)
	if err == nil {
		t.Fatal("重複したオブジェクトAPI名を受け付けました")
	}
	for _, want := range []string{"Duplicate__c", "duplicate__c", "First__c", "Second__c"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("エラーに%qがありません: %v", want, err)
		}
	}
}

func TestLoadObjectsRejectsFieldWithoutAPIName(t *testing.T) {
	objectsDirectory := t.TempDir()
	fieldsDirectory := filepath.Join(objectsDirectory, "Account", "fields")
	if err := os.MkdirAll(fieldsDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(fieldsDirectory, "Broken.field-meta.xml"),
		[]byte(`<CustomField xmlns="http://soap.sforce.com/2006/04/metadata"><label>不正項目</label></CustomField>`),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	if _, err := LoadObjects(objectsDirectory); err == nil {
		t.Fatal("API名がない項目を受け付けました")
	}
}
