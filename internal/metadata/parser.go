package metadata

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/trysmr/metago/internal/model"
)

func LoadObjects(path string) ([]model.ObjectDefinition, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("オブジェクトディレクトリを読み込めません（%s）: %w", path, err)
	}

	definitions := make([]model.ObjectDefinition, 0, len(entries))

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		objectPath := filepath.Join(path, entry.Name())
		definition, err := loadObjectDirectory(objectPath)
		if err != nil {
			return nil, err
		}

		definitions = append(definitions, definition)
	}

	return definitions, nil
}

func loadObjectDirectory(path string) (model.ObjectDefinition, error) {
	apiName := filepath.Base(path)
	definition := model.ObjectDefinition{
		APIName: apiName,
	}

	objectPath := filepath.Join(path, apiName+".object-meta.xml")
	source, err := decodeXMLFile[objectXML](objectPath, "オブジェクト")
	if err != nil {
		// 標準オブジェクトはobject-meta.xmlを持たないことがあるため、欠けていても続行する。
		if !errors.Is(err, os.ErrNotExist) {
			return model.ObjectDefinition{}, err
		}
	} else {
		definition = source.toDefinition(apiName)
	}

	fieldsPath := filepath.Join(path, "fields")
	entries, err := os.ReadDir(fieldsPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return definition, nil
		}

		return model.ObjectDefinition{}, fmt.Errorf("項目ディレクトリが読み込めません（%s）: %w", fieldsPath, err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".field-meta.xml") {
			continue
		}

		fieldPath := filepath.Join(fieldsPath, entry.Name())
		field, err := decodeXMLFile[fieldXML](fieldPath, "項目")
		if err != nil {
			return model.ObjectDefinition{}, err
		}
		if err := field.validate(); err != nil {
			return model.ObjectDefinition{}, fmt.Errorf("%w（%s）", err, fieldPath)
		}

		definition.Fields = append(definition.Fields, field.toDefinition())
	}

	return definition, nil
}

type objectXML struct {
	XMLName          xml.Name `xml:"http://soap.sforce.com/2006/04/metadata CustomObject"`
	FullName         string   `xml:"fullName"`
	Label            string   `xml:"label"`
	PluralLabel      string   `xml:"pluralLabel"`
	DeploymentStatus string   `xml:"deploymentStatus"`
	SharingModel     string   `xml:"sharingModel"`
	EnableActivities bool     `xml:"enableActivities"`
	EnableReports    bool     `xml:"enableReports"`
	EnableHistory    bool     `xml:"enableHistory"`
}

func (source objectXML) toDefinition(fallbackAPIName string) model.ObjectDefinition {
	apiName := source.FullName
	if apiName == "" {
		apiName = fallbackAPIName
	}

	return model.ObjectDefinition{
		APIName:          apiName,
		Label:            source.Label,
		PluralLabel:      source.PluralLabel,
		DeploymentStatus: source.DeploymentStatus,
		SharingModel:     source.SharingModel,
		EnableActivities: source.EnableActivities,
		EnableReports:    source.EnableReports,
		EnableHistory:    source.EnableHistory,
	}
}

type fieldXML struct {
	XMLName      xml.Name `xml:"http://soap.sforce.com/2006/04/metadata CustomField"`
	FullName     string   `xml:"fullName"`
	Label        string   `xml:"label"`
	Type         string   `xml:"type"`
	Required     bool     `xml:"required"`
	Unique       bool     `xml:"unique"`
	ExternalID   bool     `xml:"externalId"`
	Encrypted    bool     `xml:"encrypted"`
	TrackHistory bool     `xml:"trackHistory"`
	Length       *int     `xml:"length"`
	Precision    *int     `xml:"precision"`
	Scale        *int     `xml:"scale"`
	ReferenceTo  []string `xml:"referenceTo"`
	Relationship string   `xml:"relationshipName"`
	Formula      string   `xml:"formula"`
}

// オブジェクトはディレクトリ名をAPI名の代わりにできるが、項目のAPI名は
// 一覧表の主要な列であり、欠けたまま生成すると空欄の資料ができてしまう。
// メタデータ側の不備に気付けるよう、ここで生成を止める。
func (source fieldXML) validate() error {
	if strings.TrimSpace(source.FullName) == "" {
		return errors.New("項目API名がありません")
	}

	return nil
}

func (source fieldXML) toDefinition() model.FieldDefinition {
	return model.FieldDefinition{
		APIName:      source.FullName,
		Label:        source.Label,
		Type:         source.Type,
		Required:     source.Required,
		Unique:       source.Unique,
		ExternalID:   source.ExternalID,
		Encrypted:    source.Encrypted,
		TrackHistory: source.TrackHistory,
		Length:       source.Length,
		Precision:    source.Precision,
		Scale:        source.Scale,
		ReferenceTo:  source.ReferenceTo,
		Relationship: source.Relationship,
		Formula:      source.Formula,
	}
}

// 読み込みとデコードを分けているのは、ファイルが無い場合の判定（errors.Is）を
// 呼び出し側に残しつつ、デコード失敗を別の原因として報告するため。
func decodeXMLFile[T any](path string, label string) (T, error) {
	var decoded T

	data, err := os.ReadFile(path)
	if err != nil {
		return decoded, fmt.Errorf("%sXMLを読み込めません（%s）: %w", label, path, err)
	}

	if err := xml.NewDecoder(bytes.NewReader(data)).Decode(&decoded); err != nil {
		return decoded, fmt.Errorf("%sXMLが不正です（%s）: %w", label, path, err)
	}

	return decoded, nil
}
