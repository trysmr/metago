package render

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/trysmr/metago/internal/model"
	"github.com/trysmr/metago/internal/salesforceurl"
)

func TestWriteDocumentationSeparatesFormatsAndOpensSalesforceLinksInNewTab(t *testing.T) {
	links, err := salesforceurl.NewBuilder("https://example.my.salesforce.com")
	if err != nil {
		t.Fatal(err)
	}

	objects := []model.ObjectDefinition{
		{
			APIName: "Account",
			Label:   "取引先",
		},
		{
			APIName: "Invoice__c",
			Label:   "請求書",
			Fields: []model.FieldDefinition{
				{
					APIName:     "Account__c",
					Label:       "取引先",
					Type:        "Lookup",
					ReferenceTo: []string{"Account"},
				},
			},
		},
	}

	outputDirectory := t.TempDir()
	if err := WriteDocumentation(outputDirectory, objects, &links); err != nil {
		t.Fatal(err)
	}

	for _, path := range []string{
		filepath.Join(outputDirectory, "markdown", "index.md"),
		filepath.Join(outputDirectory, "markdown", "Invoice__c.md"),
		filepath.Join(outputDirectory, "html", "index.html"),
		filepath.Join(outputDirectory, "html", "Invoice__c.html"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("生成ファイルを確認できません（%s）: %v", path, err)
		}
	}

	detailPath := filepath.Join(outputDirectory, "html", "Invoice__c.html")
	detail, err := os.ReadFile(detailPath)
	if err != nil {
		t.Fatal(err)
	}

	html := string(detail)
	if !strings.Contains(
		html,
		`href="https://example.my.salesforce.com/lightning/setup/ObjectManager/Invoice__c/Details/view" target="_blank" rel="noopener noreferrer"`,
	) {
		t.Errorf("Salesforce詳細リンクに別タブ用の属性がありません\n%s", html)
	}
	if !strings.Contains(html, `href="Account.html"`) {
		t.Errorf("参照先のローカルHTMLリンクがありません\n%s", html)
	}

	marker, err := os.ReadFile(filepath.Join(outputDirectory, outputMarkerFilename))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(marker), outputMarkerContent; got != want {
		t.Errorf("マーカーの内容 = %q, want %q", got, want)
	}
}

func TestWriteDocumentationPreservesFormulaLineBreaksInEachFormat(t *testing.T) {
	objects := []model.ObjectDefinition{{
		APIName: "Invoice__c",
		Fields: []model.FieldDefinition{{
			APIName: "Status__c",
			Type:    "Text",
			Formula: "IF(\r\nAmount__c > 0 |\r\n\"有効\",\n\"無効\"\r)",
		}},
	}}

	outputDirectory := t.TempDir()
	if err := WriteDocumentation(outputDirectory, objects, nil); err != nil {
		t.Fatal(err)
	}

	markdown, err := os.ReadFile(filepath.Join(outputDirectory, "markdown", "Invoice__c.md"))
	if err != nil {
		t.Fatal(err)
	}
	markdownFormula := `<code>IF(<br>Amount__c &gt; 0 &#124;<br>&#34;有効&#34;,<br>&#34;無効&#34;<br>)</code>`
	if !strings.Contains(string(markdown), markdownFormula) {
		t.Errorf("Markdownの数式がXMLの改行を維持していません\n%s", markdown)
	}

	html, err := os.ReadFile(filepath.Join(outputDirectory, "html", "Invoice__c.html"))
	if err != nil {
		t.Fatal(err)
	}
	htmlFormula := "<code class=\"formula\">IF(\r\nAmount__c &gt; 0 |\r\n&#34;有効&#34;,\n&#34;無効&#34;\r)</code>"
	if !strings.Contains(string(html), htmlFormula) {
		t.Errorf("HTMLの数式がXMLの改行を維持していません\n%s", html)
	}
}

func TestWriteDocumentationRendersFieldAttributesAndDetailsInEachFormat(t *testing.T) {
	precision := 18
	scale := 2
	objects := []model.ObjectDefinition{
		{APIName: "Account", Label: "取引先"},
		{
			APIName: "Invoice__c",
			Fields: []model.FieldDefinition{{
				APIName:      "Amount__c",
				Label:        "金額",
				Type:         "Currency",
				Required:     true,
				Unique:       true,
				ExternalID:   true,
				Encrypted:    true,
				TrackHistory: true,
				Precision:    &precision,
				Scale:        &scale,
				ReferenceTo:  []string{"Account"},
				Relationship: "Invoices",
			}},
		},
	}

	outputDirectory := t.TempDir()
	if err := WriteDocumentation(outputDirectory, objects, nil); err != nil {
		t.Fatal(err)
	}

	markdown, err := os.ReadFile(filepath.Join(outputDirectory, "markdown", "Invoice__c.md"))
	if err != nil {
		t.Fatal(err)
	}
	wantRow := "| 金額 | `Amount__c` | `Currency` / 精度: 18 / 小数点以下: 2 / 参照先: [Account](Account.md) / リレーション名: Invoices | `必須` | `一意` | `外部ID` | `暗号化` | `履歴管理` | — |"
	if !strings.Contains(string(markdown), wantRow) {
		t.Errorf("Markdownに項目属性と詳細がありません\n%s", markdown)
	}

	html, err := os.ReadFile(filepath.Join(outputDirectory, "html", "Invoice__c.html"))
	if err != nil {
		t.Fatal(err)
	}
	htmlText := string(html)
	for _, want := range []string{
		`<a href="Account.html">Account</a>`,
		`<span class="badge">必須</span>`,
		`<span class="badge">一意</span>`,
		`<span class="badge">外部ID</span>`,
		`<span class="badge">暗号化</span>`,
		`<span class="badge">履歴管理</span>`,
	} {
		if !strings.Contains(htmlText, want) {
			t.Errorf("HTMLに%qがありません\n%s", want, htmlText)
		}
	}
}

func TestWriteDocumentationReplacesPreviousGeneratedDirectories(t *testing.T) {
	outputDirectory := t.TempDir()
	initialObjects := []model.ObjectDefinition{{
		APIName: "Stale",
		Label:   "古いオブジェクト",
	}}
	if err := WriteDocumentation(outputDirectory, initialObjects, nil); err != nil {
		t.Fatal(err)
	}

	for _, path := range []string{
		filepath.Join(outputDirectory, "markdown", "Stale.md"),
		filepath.Join(outputDirectory, "html", "Stale.html"),
	} {
		if err := os.WriteFile(path, []byte("古い生成物"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	userFile := filepath.Join(outputDirectory, "keep.txt")
	if err := os.WriteFile(userFile, []byte("残す"), 0o644); err != nil {
		t.Fatal(err)
	}

	objects := []model.ObjectDefinition{{
		APIName: "Account",
		Label:   "取引先",
	}}
	if err := WriteDocumentation(outputDirectory, objects, nil); err != nil {
		t.Fatal(err)
	}

	for _, path := range []string{
		filepath.Join(outputDirectory, "markdown", "Stale.md"),
		filepath.Join(outputDirectory, "html", "Stale.html"),
	} {
		if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("古い生成物が残っています（%s）", path)
		}
	}

	if _, err := os.Stat(userFile); err != nil {
		t.Errorf("出力ルートの利用者ファイルを確認できません: %v", err)
	}
}

func TestWriteDocumentationRefusesUnmarkedExistingGeneratedDirectories(t *testing.T) {
	outputDirectory := t.TempDir()
	userFile := filepath.Join(outputDirectory, "markdown", "notes.md")
	if err := os.MkdirAll(filepath.Dir(userFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(userFile, []byte("利用者の文書"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := WriteDocumentation(outputDirectory, nil, nil)
	if err == nil {
		t.Fatal("マーカーがない既存ディレクトリへの出力が成功しました")
	}

	content, err := os.ReadFile(userFile)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(content), "利用者の文書"; got != want {
		t.Errorf("既存ファイルの内容 = %q, want %q", got, want)
	}
}

func TestWriteDocumentationKeepsOutputUnmarkedWhenGenerationFails(t *testing.T) {
	outputDirectory := t.TempDir()

	// API名がファイル名の上限を超えるため、生成の途中で書き込みが失敗する。
	objects := []model.ObjectDefinition{{
		APIName: strings.Repeat("A", 300),
		Label:   "名前が長すぎるオブジェクト",
	}}
	if err := WriteDocumentation(outputDirectory, objects, nil); err == nil {
		t.Fatal("ファイル名が長すぎるのに生成が成功しました")
	}

	markerPath := filepath.Join(outputDirectory, outputMarkerFilename)
	if _, err := os.Stat(markerPath); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("生成に失敗したのにマーカーが残っています（%s）", markerPath)
	}

	// マーカーが残っていなければ、後から置かれた利用者のディレクトリは次回も守られる。
	userFile := filepath.Join(outputDirectory, "markdown", "notes.md")
	if err := os.MkdirAll(filepath.Dir(userFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(userFile, []byte("利用者の文書"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := WriteDocumentation(outputDirectory, nil, nil); err == nil {
		t.Fatal("マーカーがない出力先への生成が成功しました")
	}

	content, err := os.ReadFile(userFile)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(content), "利用者の文書"; got != want {
		t.Errorf("既存ファイルの内容 = %q, want %q", got, want)
	}
}

func TestWriteDocumentationRejectsForeignMarker(t *testing.T) {
	outputDirectory := t.TempDir()
	markerPath := filepath.Join(outputDirectory, outputMarkerFilename)
	if err := os.WriteFile(markerPath, []byte("別のツール\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := WriteDocumentation(outputDirectory, nil, nil); err == nil {
		t.Fatal("他ツールのマーカーがある出力先への生成が成功しました")
	}
}

func TestReplaceGeneratedDirectoriesRestoresPreviousOutputOnFailure(t *testing.T) {
	outputDirectory := t.TempDir()
	stagingDirectory := filepath.Join(outputDirectory, ".staging")

	files := map[string]string{
		filepath.Join(outputDirectory, "markdown", "index.md"):  "以前のMarkdown",
		filepath.Join(outputDirectory, "html", "index.html"):    "以前のHTML",
		filepath.Join(stagingDirectory, "markdown", "index.md"): "新しいMarkdown",
	}
	for path, content := range files {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	err := replaceGeneratedDirectories(
		outputDirectory,
		stagingDirectory,
		[]string{"markdown", "html"},
	)
	if err == nil {
		t.Fatal("一時HTMLがないのに入れ替えが成功しました")
	}

	for path, want := range map[string]string{
		filepath.Join(outputDirectory, "markdown", "index.md"): "以前のMarkdown",
		filepath.Join(outputDirectory, "html", "index.html"):   "以前のHTML",
	} {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if got := string(content); got != want {
			t.Errorf("%sの内容 = %q, want %q", path, got, want)
		}
	}
}
