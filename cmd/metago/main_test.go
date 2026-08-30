package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"testing"
)

func TestRunPrintsVersion(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := run([]string{"--version"}, &stdout, &stderr)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := stdout.String(), "metago dev\n"; got != want {
		t.Errorf("バージョン出力 = %q, want %q", got, want)
	}
}

func TestSelectVersionPrefersInjectedValueThenModuleVersion(t *testing.T) {
	if got, want := selectVersion("v0.1.0", "v0.0.0-20260829093243-c2a0cf6ea38b"), "v0.1.0"; got != want {
		t.Errorf("GitHub Releasesの実行ファイル = %q, want %q", got, want)
	}
	if got, want := selectVersion("dev", "v0.1.0"), "v0.1.0"; got != want {
		t.Errorf("go installで入れた場合 = %q, want %q", got, want)
	}
	if got, want := selectVersion("dev", "v1.2.3-rc.1"), "v1.2.3-rc.1"; got != want {
		t.Errorf("go installでプレリリースを指定した場合 = %q, want %q", got, want)
	}
	if got, want := selectVersion("dev", "v0.0.0-20260829093243-c2a0cf6ea38b"), "dev"; got != want {
		t.Errorf("リポジトリ内でgo buildした場合 = %q, want %q", got, want)
	}
	if got, want := selectVersion("dev", "(devel)"), "dev"; got != want {
		t.Errorf("go testやbuildvcsなしのビルド = %q, want %q", got, want)
	}
	if got, want := selectVersion("dev", ""), "dev"; got != want {
		t.Errorf("モジュール情報がない場合 = %q, want %q", got, want)
	}
}

func TestResolveVersionUsesModuleVersionWhenNotInjected(t *testing.T) {
	original := readBuildInfo
	t.Cleanup(func() { readBuildInfo = original })

	readBuildInfo = func() (*debug.BuildInfo, bool) {
		info := &debug.BuildInfo{}
		info.Main.Version = "v0.1.0"

		return info, true
	}

	if got, want := resolveVersion(), "v0.1.0"; got != want {
		t.Errorf("go installで入れた場合のバージョン = %q, want %q", got, want)
	}
}

func TestResolveVersionFallsBackWhenBuildInfoIsUnavailable(t *testing.T) {
	original := readBuildInfo
	t.Cleanup(func() { readBuildInfo = original })

	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return nil, false
	}

	if got, want := resolveVersion(), "dev"; got != want {
		t.Errorf("ビルド情報を読めない場合のバージョン = %q, want %q", got, want)
	}
}

func TestObjectCountLabelUsesSingularForOne(t *testing.T) {
	if got, want := objectCountLabel(0), "0 objects"; got != want {
		t.Errorf("0件の表記 = %q, want %q", got, want)
	}
	if got, want := objectCountLabel(1), "1 object"; got != want {
		t.Errorf("1件の表記 = %q, want %q", got, want)
	}
	if got, want := objectCountLabel(2), "2 objects"; got != want {
		t.Errorf("2件の表記 = %q, want %q", got, want)
	}
}

func TestRunPrintsHelp(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := run([]string{"--help"}, &stdout, &stderr)
	if !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("--helpのエラー = %v, want %v", err, flag.ErrHelp)
	}
	if !strings.Contains(stderr.String(), "Usage of metago:") {
		t.Errorf("ヘルプ出力にUsageがありません\n%s", stderr.String())
	}
}

func TestRunRejectsMissingInput(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := run([]string{"--output", t.TempDir()}, &stdout, &stderr)
	if err == nil {
		t.Fatal("--inputがないのにコマンドが成功しました")
	}
	if !strings.Contains(err.Error(), "--input is required") {
		t.Errorf("エラーに不足したオプションがありません: %v", err)
	}
}

func TestRunRejectsOutputInsideInput(t *testing.T) {
	objectsDirectory := filepath.Join(t.TempDir(), "objects")
	if err := os.MkdirAll(filepath.Join(objectsDirectory, "Account"), 0o755); err != nil {
		t.Fatal(err)
	}

	outputDirectory := filepath.Join(objectsDirectory, "generated")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := run(
		[]string{
			"--input",
			objectsDirectory,
			"--output",
			outputDirectory,
		},
		&stdout,
		&stderr,
	)
	if err == nil {
		t.Fatal("入力ディレクトリ内の生成先を受け付けました")
	}
	if !strings.Contains(err.Error(), "must not overlap") {
		t.Errorf("エラーに入出力の重なりが示されていません: %v", err)
	}
	if _, statErr := os.Stat(outputDirectory); !errors.Is(statErr, os.ErrNotExist) {
		t.Errorf("拒否した生成先が作成されました: %v", statErr)
	}
}

func TestValidateSeparatePathsRejectsEveryOverlapDirection(t *testing.T) {
	rootDirectory := t.TempDir()
	inputDirectory := filepath.Join(rootDirectory, "objects")
	if err := os.MkdirAll(inputDirectory, 0o755); err != nil {
		t.Fatal(err)
	}

	for _, paths := range []struct {
		name   string
		input  string
		output string
	}{
		{name: "same", input: inputDirectory, output: inputDirectory},
		{name: "output inside input", input: inputDirectory, output: filepath.Join(inputDirectory, "generated")},
		{name: "input inside output", input: inputDirectory, output: rootDirectory},
	} {
		t.Run(paths.name, func(t *testing.T) {
			if err := validateSeparatePaths(paths.input, paths.output); err == nil {
				t.Fatal("重なる入出力パスを受け付けました")
			}
		})
	}
}

func TestValidateSeparatePathsResolvesSymbolicLinks(t *testing.T) {
	rootDirectory := t.TempDir()
	inputDirectory := filepath.Join(rootDirectory, "objects")
	if err := os.MkdirAll(inputDirectory, 0o755); err != nil {
		t.Fatal(err)
	}

	inputLink := filepath.Join(rootDirectory, "objects-link")
	if err := os.Symlink(inputDirectory, inputLink); err != nil {
		t.Skipf("シンボリックリンクを作成できません: %v", err)
	}

	if err := validateSeparatePaths(
		inputDirectory,
		filepath.Join(inputLink, "generated"),
	); err == nil {
		t.Fatal("シンボリックリンクを経由して重なる入出力パスを受け付けました")
	}
}

func TestRunGeneratesMarkdownAndHTMLFromLocalMetadata(t *testing.T) {
	objectsDirectory := filepath.Join(t.TempDir(), "objects")
	fieldsDirectory := filepath.Join(objectsDirectory, "Account", "fields")
	if err := os.MkdirAll(fieldsDirectory, 0o755); err != nil {
		t.Fatal(err)
	}

	fieldXML := `<?xml version="1.0" encoding="UTF-8"?>
<CustomField xmlns="http://soap.sforce.com/2006/04/metadata">
  <fullName>Score__c</fullName>
  <label>スコア</label>
  <type>Number</type>
  <precision>10</precision>
  <scale>2</scale>
</CustomField>`
	if err := os.WriteFile(
		filepath.Join(fieldsDirectory, "Score__c.field-meta.xml"),
		[]byte(fieldXML),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	outputDirectory := filepath.Join(t.TempDir(), "generated")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := run(
		[]string{
			"--input",
			objectsDirectory,
			"--output",
			outputDirectory,
		},
		&stdout,
		&stderr,
	)
	if err != nil {
		t.Fatalf("metagoの生成が失敗しました: %v\n%s", err, stderr.String())
	}

	wantOutput := fmt.Sprintf(
		"Generated Markdown and HTML for 1 object: %s\n",
		outputDirectory,
	)
	if got := stdout.String(); got != wantOutput {
		t.Errorf("コマンド出力 = %q, want %q", got, wantOutput)
	}

	for _, path := range []string{
		filepath.Join(outputDirectory, ".metago-generated"),
		filepath.Join(outputDirectory, "markdown", "index.md"),
		filepath.Join(outputDirectory, "markdown", "Account.md"),
		filepath.Join(outputDirectory, "html", "index.html"),
		filepath.Join(outputDirectory, "html", "Account.html"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("生成ファイルを確認できません（%s）: %v", path, err)
		}
	}
}
