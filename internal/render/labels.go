package render

import "fmt"

type Language string

const (
	LanguageEnglish  Language = "en"
	LanguageJapanese Language = "ja"
)

func ParseLanguage(value string) (Language, error) {
	switch language := Language(value); language {
	case LanguageEnglish, LanguageJapanese:
		return language, nil
	default:
		return "", fmt.Errorf("unsupported language %q (use en or ja)", value)
	}
}

// labelsは生成物の見出しをまとめる。値はSalesforceの設定画面で使われている
// 表記に合わせてあり、生成した資料と設定画面を突き合わせられるようにしている。
type labels struct {
	HTMLLang          string
	ObjectIndexTitle  string
	ObjectDetails     string
	Fields            string
	NoFields          string
	BackToIndex       string
	SetupPrefix       string
	SetupDetails      string
	SetupFields       string
	Attribute         string
	Value             string
	Label             string
	APIName           string
	FieldCount        string
	PluralLabel       string
	DeploymentStatus  string
	SharingModel      string
	Activities        string
	Reports           string
	TrackFieldHistory string
	TypeAndDetails    string
	Formula           string
	Enabled           string
	Disabled          string
	Required          string
	Unique            string
	ExternalID        string
	Encrypted         string
	TrackHistory      string
	Length            string
	Precision         string
	DecimalPlaces     string
	RelatedTo         string
	RelationshipName  string
}

var englishLabels = labels{
	HTMLLang:          "en",
	ObjectIndexTitle:  "Objects",
	ObjectDetails:     "Object Details",
	Fields:            "Fields",
	NoFields:          "No fields are defined.",
	BackToIndex:       "Back to the object list",
	SetupPrefix:       "Salesforce Setup",
	SetupDetails:      "Details",
	SetupFields:       "Fields & Relationships",
	Attribute:         "Attribute",
	Value:             "Value",
	Label:             "Label",
	APIName:           "API Name",
	FieldCount:        "Fields",
	PluralLabel:       "Plural Label",
	DeploymentStatus:  "Deployment Status",
	SharingModel:      "Sharing Model",
	Activities:        "Allow Activities",
	Reports:           "Allow Reports",
	TrackFieldHistory: "Track Field History",
	TypeAndDetails:    "Type & Details",
	Formula:           "Formula",
	Enabled:           "Enabled",
	Disabled:          "Disabled",
	Required:          "Required",
	Unique:            "Unique",
	ExternalID:        "External ID",
	Encrypted:         "Encrypted",
	TrackHistory:      "Track History",
	Length:            "Length",
	Precision:         "Precision",
	DecimalPlaces:     "Decimal Places",
	RelatedTo:         "Related To",
	RelationshipName:  "Relationship Name",
}

var japaneseLabels = labels{
	HTMLLang:          "ja",
	ObjectIndexTitle:  "オブジェクト一覧",
	ObjectDetails:     "オブジェクト情報",
	Fields:            "項目一覧",
	NoFields:          "項目定義はありません。",
	BackToIndex:       "オブジェクト一覧へ戻る",
	SetupPrefix:       "Salesforce設定",
	SetupDetails:      "詳細",
	SetupFields:       "項目とリレーション",
	Attribute:         "属性",
	Value:             "値",
	Label:             "表示ラベル",
	APIName:           "API名",
	FieldCount:        "項目数",
	PluralLabel:       "複数形の表示ラベル",
	DeploymentStatus:  "リリース情報",
	SharingModel:      "共有モデル",
	Activities:        "活動",
	Reports:           "レポート",
	TrackFieldHistory: "項目履歴管理",
	TypeAndDetails:    "型・詳細",
	Formula:           "数式",
	Enabled:           "有効",
	Disabled:          "無効",
	Required:          "必須",
	Unique:            "一意",
	ExternalID:        "外部ID",
	Encrypted:         "暗号化",
	TrackHistory:      "履歴管理",
	Length:            "長さ",
	Precision:         "精度",
	DecimalPlaces:     "小数点以下",
	RelatedTo:         "参照先",
	RelationshipName:  "リレーション名",
}

func labelsFor(language Language) labels {
	if language == LanguageJapanese {
		return japaneseLabels
	}

	return englishLabels
}
