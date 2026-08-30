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
	l labels,
) error {
	if err := os.MkdirAll(outputDirectory, 0o755); err != nil {
		return fmt.Errorf(
			"cannot create the Markdown output directory (%s): %w",
			outputDirectory,
			err,
		)
	}

	indexPath := filepath.Join(outputDirectory, "index"+markdownExtension)
	if err := writeGeneratedFile(indexPath, markdownIndex(objects, l)); err != nil {
		return err
	}

	for _, object := range objects {
		path := filepath.Join(
			outputDirectory,
			objectFilename(object.APIName, markdownExtension),
		)

		if err := writeGeneratedFile(
			path,
			markdownObject(object, links, availableObjects, l),
		); err != nil {
			return err
		}
	}

	return nil
}

func markdownIndex(objects []model.ObjectDefinition, l labels) string {
	var output strings.Builder

	fmt.Fprintf(&output, "# %s\n\n", l.ObjectIndexTitle)
	writeMarkdownRow(&output, []string{l.Label, l.APIName, l.FieldCount})
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
	l labels,
) string {
	var output strings.Builder

	fmt.Fprintf(
		&output,
		"# %s (%s)\n\n",
		markdownText(displayLabel(object)),
		markdownInlineCode(object.APIName),
	)

	if links != nil {
		fmt.Fprintf(
			&output,
			"%s: [%s](%s) | [%s](%s)\n\n",
			l.SetupPrefix,
			l.SetupDetails,
			links.ObjectDetails(object.APIName),
			l.SetupFields,
			links.ObjectFields(object.APIName),
		)
	}

	fmt.Fprintf(&output, "## %s\n\n", l.ObjectDetails)
	writeMarkdownRow(&output, []string{l.Attribute, l.Value})
	writeMarkdownRow(&output, markdownSeparators(2))

	writeMarkdownProperty(&output, l.APIName, object.APIName)
	writeMarkdownProperty(&output, l.Label, object.Label)
	writeMarkdownProperty(&output, l.PluralLabel, object.PluralLabel)
	writeMarkdownProperty(&output, l.DeploymentStatus, object.DeploymentStatus)
	writeMarkdownProperty(&output, l.SharingModel, object.SharingModel)
	writeMarkdownProperty(&output, l.Activities, enabledText(object.EnableActivities, l))
	writeMarkdownProperty(&output, l.Reports, enabledText(object.EnableReports, l))
	writeMarkdownProperty(&output, l.TrackFieldHistory, enabledText(object.EnableHistory, l))

	fmt.Fprintf(&output, "\n## %s\n\n", l.Fields)

	if len(object.Fields) == 0 {
		fmt.Fprintf(&output, "%s\n", l.NoFields)

		return output.String()
	}

	flags := fieldFlagsFor(l)
	headers := append([]string{l.Label, l.APIName, l.TypeAndDetails}, fieldFlagLabels(flags)...)
	headers = append(headers, l.Formula)

	writeMarkdownRow(&output, headers)
	writeMarkdownRow(&output, markdownSeparators(len(headers)))

	for _, field := range object.Fields {
		cells := []string{
			markdownText(field.Label),
			markdownInlineCode(field.APIName),
			markdownTypeCell(field, availableObjects, l),
		}
		for _, flag := range flags {
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
	l labels,
) string {
	value := markdownInlineCode(field.Type)

	if details := markdownDetails(field, availableObjects, l); details != "" {
		value += " / " + details
	}

	return value
}

func markdownDetails(
	field model.FieldDefinition,
	availableObjects map[string]struct{},
	l labels,
) string {
	details := fieldDetails(field, l)
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
