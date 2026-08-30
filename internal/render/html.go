package render

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"os"
	"path/filepath"

	"github.com/trysmr/metago/internal/model"
	"github.com/trysmr/metago/internal/salesforceurl"
)

//go:embed templates/*.html.tmpl
var templateFiles embed.FS

const htmlExtension = ".html"

func writeHTML(
	outputDirectory string,
	objects []model.ObjectDefinition,
	links *salesforceurl.Builder,
	availableObjects map[string]struct{},
) error {
	if err := os.MkdirAll(outputDirectory, 0o755); err != nil {
		return fmt.Errorf(
			"cannot create the HTML output directory (%s): %w",
			outputDirectory,
			err,
		)
	}

	index, err := renderHTML(indexTemplate, objects)
	if err != nil {
		return err
	}

	indexPath := filepath.Join(outputDirectory, "index"+htmlExtension)
	if err := writeGeneratedFile(indexPath, index); err != nil {
		return err
	}

	for _, object := range objects {
		content, err := renderHTML(
			objectTemplate,
			newObjectHTMLData(object, links, availableObjects),
		)
		if err != nil {
			return fmt.Errorf(
				"cannot render HTML for the object (%s): %w",
				object.APIName,
				err,
			)
		}

		path := filepath.Join(
			outputDirectory,
			objectFilename(object.APIName, htmlExtension),
		)

		if err := writeGeneratedFile(path, content); err != nil {
			return err
		}
	}

	return nil
}

func renderHTML(page *template.Template, data any) (string, error) {
	var output bytes.Buffer
	if err := page.Execute(&output, data); err != nil {
		return "", fmt.Errorf("cannot render HTML: %w", err)
	}

	return output.String(), nil
}

// テンプレートは埋め込み済みで実行中に変化しないため、起動時に一度だけ解析する。
// 解析に失敗するのはテンプレートを壊したときだけなので、その場で停止させる。
var (
	indexTemplate = mustParseTemplate("index.html.tmpl", template.FuncMap{
		"displayLabel": displayLabel,
		"objectFilename": func(apiName string) string {
			return objectFilename(apiName, htmlExtension)
		},
	})

	objectTemplate = mustParseTemplate("object.html.tmpl", template.FuncMap{
		"enabledText": enabledText,
	})
)

func mustParseTemplate(name string, functions template.FuncMap) *template.Template {
	return template.Must(
		template.New(name).
			Funcs(functions).
			ParseFS(
				templateFiles,
				"templates/"+name,
				"templates/styles.html.tmpl",
			),
	)
}

type objectHTMLData struct {
	Object     model.ObjectDefinition
	Label      string
	DetailURL  string
	FieldsURL  string
	FlagLabels []string
	Fields     []fieldHTMLData
}

type fieldHTMLData struct {
	Definition model.FieldDefinition
	Details    []detailHTMLData
	Flags      []flagHTMLData
}

type detailHTMLData struct {
	Label      string
	Value      string
	References []referenceHTMLData
}

type flagHTMLData struct {
	Label   string
	Enabled bool
}

type referenceHTMLData struct {
	APIName string
	URL     string
}

func newObjectHTMLData(
	object model.ObjectDefinition,
	links *salesforceurl.Builder,
	availableObjects map[string]struct{},
) objectHTMLData {
	data := objectHTMLData{
		Object:     object,
		Label:      displayLabel(object),
		FlagLabels: fieldFlagLabels(),
		Fields:     make([]fieldHTMLData, 0, len(object.Fields)),
	}

	if links != nil {
		data.DetailURL = links.ObjectDetails(object.APIName)
		data.FieldsURL = links.ObjectFields(object.APIName)
	}

	for _, field := range object.Fields {
		fieldData := fieldHTMLData{
			Definition: field,
			Details:    newDetailHTMLData(field, availableObjects),
			Flags:      make([]flagHTMLData, 0, len(fieldFlags)),
		}

		for _, flag := range fieldFlags {
			fieldData.Flags = append(fieldData.Flags, flagHTMLData{
				Label:   flag.Label,
				Enabled: flag.Enabled(field),
			})
		}

		data.Fields = append(data.Fields, fieldData)
	}

	return data
}

func newDetailHTMLData(
	field model.FieldDefinition,
	availableObjects map[string]struct{},
) []detailHTMLData {
	details := fieldDetails(field)
	converted := make([]detailHTMLData, 0, len(details))

	for _, detail := range details {
		converted = append(converted, detailHTMLData{
			Label:      detail.Label,
			Value:      detail.Value,
			References: newReferenceHTMLData(detail.References, availableObjects),
		})
	}

	return converted
}

func newReferenceHTMLData(
	references []string,
	availableObjects map[string]struct{},
) []referenceHTMLData {
	converted := make([]referenceHTMLData, 0, len(references))

	for _, reference := range references {
		data := referenceHTMLData{APIName: reference}
		if _, exists := availableObjects[reference]; exists {
			data.URL = objectFilename(reference, htmlExtension)
		}

		converted = append(converted, data)
	}

	return converted
}
