package render

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/trysmr/metago/internal/model"
)

// MarkdownとHTMLの見出し・セルをどちらもこの定義から組み立てるため、
// 属性を増減するときの修正はこのスライス1箇所で済む。
type fieldFlag struct {
	Label   string
	Enabled func(model.FieldDefinition) bool
}

var fieldFlags = []fieldFlag{
	{Label: "必須", Enabled: func(field model.FieldDefinition) bool { return field.Required }},
	{Label: "一意", Enabled: func(field model.FieldDefinition) bool { return field.Unique }},
	{Label: "外部ID", Enabled: func(field model.FieldDefinition) bool { return field.ExternalID }},
	{Label: "暗号化", Enabled: func(field model.FieldDefinition) bool { return field.Encrypted }},
	{Label: "履歴管理", Enabled: func(field model.FieldDefinition) bool { return field.TrackHistory }},
}

func fieldFlagLabels() []string {
	labels := make([]string, 0, len(fieldFlags))
	for _, flag := range fieldFlags {
		labels = append(labels, flag.Label)
	}

	return labels
}

// 参照先はオブジェクトへのリンクになるため、値ではなくAPI名の一覧で保持し、
// リンクの組み立ては出力形式ごとのレンダラーに任せる。
type fieldDetail struct {
	Label      string
	Value      string
	References []string
}

func fieldDetails(field model.FieldDefinition) []fieldDetail {
	details := make([]fieldDetail, 0, 5)

	if field.Length != nil {
		details = append(details, fieldDetail{Label: "長さ", Value: strconv.Itoa(*field.Length)})
	}
	if field.Precision != nil {
		details = append(details, fieldDetail{Label: "精度", Value: strconv.Itoa(*field.Precision)})
	}
	if field.Scale != nil {
		details = append(details, fieldDetail{Label: "小数点以下", Value: strconv.Itoa(*field.Scale)})
	}
	if len(field.ReferenceTo) > 0 {
		details = append(details, fieldDetail{Label: "参照先", References: field.ReferenceTo})
	}
	if field.Relationship != "" {
		details = append(details, fieldDetail{Label: "リレーション名", Value: field.Relationship})
	}

	return details
}

func displayLabel(object model.ObjectDefinition) string {
	if object.Label != "" {
		return object.Label
	}

	return object.APIName
}

func enabledText(enabled bool) string {
	if enabled {
		return "有効"
	}

	return "無効"
}

var newlineReplacer = strings.NewReplacer(
	"\r\n", " ",
	"\n", " ",
	"\r", " ",
)

// 改行を含む値でも表のセルに収まるようにする。
func collapseNewlines(value string) string {
	return newlineReplacer.Replace(value)
}

// 参照先が生成対象に含まれるときだけ相互リンクを張るために使う。
func availableObjectNames(objects []model.ObjectDefinition) map[string]struct{} {
	names := make(map[string]struct{}, len(objects))
	for _, object := range objects {
		names[object.APIName] = struct{}{}
	}

	return names
}

// 想定外の文字がAPI名に含まれてもリンクが壊れないようエスケープする。
func objectFilename(apiName string, extension string) string {
	return url.PathEscape(apiName) + extension
}

func writeGeneratedFile(path string, content string) error {
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("cannot write the generated file (%s): %w", path, err)
	}

	return nil
}
