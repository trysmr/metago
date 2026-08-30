package render

import (
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"

	"github.com/trysmr/metago/internal/model"
	"github.com/trysmr/metago/internal/salesforceurl"
)

const markdownExtension = ".md"

func writeMarkdown(
	outputDirectory string,
	objects []model.ObjectDefinition,
	links *salesforceurl.Builder,
	availableObjects map[string]struct{},
) error {
	if err := os.MkdirAll(outputDirectory, 0o755); err != nil {
		return fmt.Errorf(
			"cannot create the Markdown output directory (%s): %w",
			outputDirectory,
			err,
		)
	}

	indexPath := filepath.Join(outputDirectory, "index"+markdownExtension)
	if err := writeGeneratedFile(indexPath, markdownIndex(objects)); err != nil {
		return err
	}

	for _, object := range objects {
		path := filepath.Join(
			outputDirectory,
			objectFilename(object.APIName, markdownExtension),
		)

		if err := writeGeneratedFile(
			path,
			markdownObject(object, links, availableObjects),
		); err != nil {
			return err
		}
	}

	return nil
}

func markdownIndex(objects []model.ObjectDefinition) string {
	var output strings.Builder

	output.WriteString("# オブジェクト一覧\n\n")
	writeMarkdownRow(&output, []string{"表示ラベル", "API名", "項目数"})
	writeMarkdownRow(&output, markdownSeparators(3))

	for _, object := range objects {
		writeMarkdownRow(&output, []string{
			fmt.Sprintf(
				"[%s](%s)",
				markdownText(displayLabel(object)),
				objectFilename(object.APIName, markdownExtension),
			),
			markdownInlineCode(object.APIName),
			fmt.Sprint(len(object.Fields)),
		})
	}

	return output.String()
}

func markdownObject(
	object model.ObjectDefinition,
	links *salesforceurl.Builder,
	availableObjects map[string]struct{},
) string {
	var output strings.Builder

	fmt.Fprintf(
		&output,
		"# %s（%s）\n\n",
		markdownText(displayLabel(object)),
		markdownInlineCode(object.APIName),
	)

	if links != nil {
		fmt.Fprintf(
			&output,
			"Salesforce設定: [詳細](%s) | [項目とリレーション](%s)\n\n",
			links.ObjectDetails(object.APIName),
			links.ObjectFields(object.APIName),
		)
	}

	output.WriteString("## オブジェクト情報\n\n")
	writeMarkdownRow(&output, []string{"属性", "値"})
	writeMarkdownRow(&output, markdownSeparators(2))

	writeMarkdownProperty(&output, "API名", object.APIName)
	writeMarkdownProperty(&output, "表示ラベル", object.Label)
	writeMarkdownProperty(&output, "複数形の表示ラベル", object.PluralLabel)
	writeMarkdownProperty(&output, "リリース情報", object.DeploymentStatus)
	writeMarkdownProperty(&output, "共有モデル", object.SharingModel)
	writeMarkdownProperty(&output, "活動", enabledText(object.EnableActivities))
	writeMarkdownProperty(&output, "レポート", enabledText(object.EnableReports))
	writeMarkdownProperty(&output, "項目履歴管理", enabledText(object.EnableHistory))

	output.WriteString("\n## 項目一覧\n\n")

	if len(object.Fields) == 0 {
		output.WriteString("項目定義はありません。\n")

		return output.String()
	}

	headers := append([]string{"表示ラベル", "API名", "型・詳細"}, fieldFlagLabels()...)
	headers = append(headers, "数式")

	writeMarkdownRow(&output, headers)
	writeMarkdownRow(&output, markdownSeparators(len(headers)))

	for _, field := range object.Fields {
		cells := []string{
			markdownText(field.Label),
			markdownInlineCode(field.APIName),
			markdownTypeCell(field, availableObjects),
		}
		for _, flag := range fieldFlags {
			cells = append(cells, markdownBadge(flag.Enabled(field), flag.Label))
		}
		cells = append(cells, markdownFormulaCell(field.Formula))

		writeMarkdownRow(&output, cells)
	}

	return output.String()
}

func writeMarkdownRow(output *strings.Builder, cells []string) {
	fmt.Fprintf(output, "| %s |\n", strings.Join(cells, " | "))
}

func markdownSeparators(count int) []string {
	separators := make([]string, count)
	for index := range separators {
		separators[index] = "---"
	}

	return separators
}

func writeMarkdownProperty(
	output *strings.Builder,
	name string,
	value string,
) {
	writeMarkdownRow(output, []string{markdownText(name), markdownText(value)})
}

func markdownTypeCell(
	field model.FieldDefinition,
	availableObjects map[string]struct{},
) string {
	value := markdownInlineCode(field.Type)

	if details := markdownDetails(field, availableObjects); details != "" {
		value += " / " + details
	}

	return value
}

func markdownDetails(
	field model.FieldDefinition,
	availableObjects map[string]struct{},
) string {
	details := fieldDetails(field)
	values := make([]string, 0, len(details))

	for _, detail := range details {
		if detail.References != nil {
			values = append(
				values,
				detail.Label+": "+markdownReferences(detail.References, availableObjects),
			)

			continue
		}

		values = append(values, detail.Label+": "+markdownText(detail.Value))
	}

	return strings.Join(values, " / ")
}

func markdownReferences(
	references []string,
	availableObjects map[string]struct{},
) string {
	values := make([]string, 0, len(references))

	for _, reference := range references {
		if _, exists := availableObjects[reference]; exists {
			values = append(values, fmt.Sprintf(
				"[%s](%s)",
				markdownText(reference),
				objectFilename(reference, markdownExtension),
			))

			continue
		}

		values = append(values, markdownInlineCode(reference))
	}

	return strings.Join(values, ", ")
}

func markdownBadge(enabled bool, label string) string {
	if !enabled {
		return "-"
	}

	return markdownInlineCode(label)
}

// 数式の改行をHTMLへ変換すると、Markdown表の1行を保ったまま
// XMLに記述された改行位置を表示できる。
var markdownFormulaNewlineReplacer = strings.NewReplacer(
	"\r\n", "<br>",
	"\n", "<br>",
	"\r", "<br>",
)

func markdownFormulaCell(formula string) string {
	value := strings.TrimSpace(formula)
	if value == "" {
		return "—"
	}

	escaped := html.EscapeString(value)
	escaped = strings.ReplaceAll(escaped, "|", "&#124;")
	escaped = markdownFormulaNewlineReplacer.Replace(escaped)

	return "<code>" + escaped + "</code>"
}

var markdownEscapeReplacer = strings.NewReplacer(
	`\`, `\\`,
	"|", `\|`,
	"[", `\[`,
	"]", `\]`,
)

func markdownText(value string) string {
	return collapseNewlines(markdownEscapeReplacer.Replace(value))
}

func markdownInlineCode(value string) string {
	value = strings.ReplaceAll(value, "|", `\|`)

	fence := "`"
	for strings.Contains(value, fence) {
		fence += "`"
	}

	// 値の端がバッククォートや空白だと囲みと混ざるため、そのときだけ空白を挟む。
	if strings.HasPrefix(value, "`") || strings.HasSuffix(value, "`") ||
		strings.HasPrefix(value, " ") || strings.HasSuffix(value, " ") {
		return fence + " " + value + " " + fence
	}

	return fence + value + fence
}
