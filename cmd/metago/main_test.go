package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
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
	if !strings.Contains(err.Error(), "--inputを指定してください") {
		t.Errorf("エラーに不足したオプションがありません: %v", err)
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
		"1件のオブジェクトからMarkdownとHTMLを生成しました: %s\n",
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
