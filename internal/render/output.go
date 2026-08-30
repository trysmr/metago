package render

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/trysmr/metago/internal/model"
	"github.com/trysmr/metago/internal/salesforceurl"
)

const (
	outputMarkerFilename = ".metago-generated"
	outputMarkerContent  = "metago\n"

	markdownDirectoryName = "markdown"
	htmlDirectoryName     = "html"
)

// マーカーがない場合に既存の内容を保護する必要があるディレクトリ。
// マーカー自体は所有を示す側なので、この一覧には含めない。
var generatedDirectoryNames = []string{markdownDirectoryName, htmlDirectoryName}

// 入れ替えと復元を同じ単位で行う出力ルート直下の項目。
// マーカーの設置に失敗した場合も以前の生成結果を復元するため、マーカーを含める。
var generatedOutputEntryNames = []string{
	markdownDirectoryName,
	htmlDirectoryName,
	outputMarkerFilename,
}

// WriteDocumentationがマーカーを入れ替え前のステージングへ書くことを
// テストで確認できるよう、書き込み処理を差し替え可能にする。
var writeOutputMarker = markOutputDirectory

func WriteDocumentation(
	outputDirectory string,
	objects []model.ObjectDefinition,
	links *salesforceurl.Builder,
	language Language,
) (resultErr error) {
	if err := os.MkdirAll(outputDirectory, 0o755); err != nil {
		return fmt.Errorf(
			"cannot create the output directory (%s): %w",
			outputDirectory,
			err,
		)
	}
	if err := verifyOutputDirectoryOwnership(outputDirectory); err != nil {
		return err
	}

	stagingDirectory, err := os.MkdirTemp(
		outputDirectory,
		".metago-",
	)
	if err != nil {
		return fmt.Errorf("cannot create the staging directory: %w", err)
	}
	preserveStagingDirectory := false
	defer func() {
		if preserveStagingDirectory {
			return
		}

		if err := os.RemoveAll(stagingDirectory); err != nil {
			resultErr = errors.Join(
				resultErr,
				fmt.Errorf("cannot remove the staging directory: %w", err),
			)
		}
	}()

	availableObjects := availableObjectNames(objects)
	l := labelsFor(language)

	markdownDirectory := filepath.Join(stagingDirectory, markdownDirectoryName)
	if err := writeMarkdown(markdownDirectory, objects, links, availableObjects, l); err != nil {
		return fmt.Errorf("cannot generate Markdown: %w", err)
	}

	htmlDirectory := filepath.Join(stagingDirectory, htmlDirectoryName)
	if err := writeHTML(htmlDirectory, objects, links, availableObjects, l); err != nil {
		return fmt.Errorf("cannot generate HTML: %w", err)
	}
	if err := writeOutputMarker(stagingDirectory); err != nil {
		return fmt.Errorf("cannot prepare the output marker: %w", err)
	}

	if err := replaceGeneratedOutputEntries(
		outputDirectory,
		stagingDirectory,
		generatedOutputEntryNames,
	); err != nil {
		preserveStagingDirectory = true
		return fmt.Errorf(
			"cannot swap the generated output; kept the staging directory (%s): %w",
			stagingDirectory,
			err,
		)
	}

	return nil
}

// マーカーがないのに生成先ディレクトリが既にある場合は、利用者が用意したものと
// 見なして出力を拒む。
func verifyOutputDirectoryOwnership(outputDirectory string) error {
	markerPath := filepath.Join(outputDirectory, outputMarkerFilename)
	content, err := os.ReadFile(markerPath)
	if err == nil {
		if string(content) != outputMarkerContent {
			return fmt.Errorf("the marker in the output directory is not recognized (%s)", markerPath)
		}

		return nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("cannot check the marker (%s): %w", markerPath, err)
	}

	for _, name := range generatedDirectoryNames {
		path := filepath.Join(outputDirectory, name)
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf(
				"the output directory has no marker but %s already exists (%s)",
				name,
				path,
			)
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("cannot check the existing generated directory (%s): %w", path, err)
		}
	}

	return nil
}

func markOutputDirectory(outputDirectory string) error {
	markerPath := filepath.Join(outputDirectory, outputMarkerFilename)
	if err := os.WriteFile(markerPath, []byte(outputMarkerContent), 0o644); err != nil {
		return fmt.Errorf("cannot write the marker (%s): %w", markerPath, err)
	}

	return nil
}

type generatedOutputEntryBackup struct {
	name string
	path string
}

func replaceGeneratedOutputEntries(
	outputDirectory string,
	stagingDirectory string,
	entryNames []string,
) error {
	backups := make([]generatedOutputEntryBackup, 0, len(entryNames))

	for _, name := range entryNames {
		target := filepath.Join(outputDirectory, name)
		if _, err := os.Stat(target); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}

			return fmt.Errorf("既存の生成先を確認できません（%s）: %w", target, err)
		}

		backup := filepath.Join(stagingDirectory, "previous-"+name)
		if err := os.Rename(target, backup); err != nil {
			rollbackErr := restoreGeneratedOutputEntries(outputDirectory, backups)
			return errors.Join(
				fmt.Errorf("cannot move the existing generated output entry aside (%s): %w", target, err),
				rollbackErr,
			)
		}

		backups = append(backups, generatedOutputEntryBackup{
			name: name,
			path: backup,
		})
	}

	installed := make([]string, 0, len(entryNames))
	for _, name := range entryNames {
		source := filepath.Join(stagingDirectory, name)
		target := filepath.Join(outputDirectory, name)
		if err := os.Rename(source, target); err != nil {
			rollbackErr := rollbackGeneratedOutputEntries(
				outputDirectory,
				installed,
				backups,
			)
			return errors.Join(
				fmt.Errorf("cannot swap the generated output entry (%s): %w", target, err),
				rollbackErr,
			)
		}

		installed = append(installed, name)
	}

	return nil
}

func rollbackGeneratedOutputEntries(
	outputDirectory string,
	installed []string,
	backups []generatedOutputEntryBackup,
) error {
	var rollbackErr error
	for _, name := range installed {
		path := filepath.Join(outputDirectory, name)
		if err := os.RemoveAll(path); err != nil {
			rollbackErr = errors.Join(
				rollbackErr,
				fmt.Errorf("cannot remove the new generated output entry (%s): %w", path, err),
			)
		}
	}

	return errors.Join(
		rollbackErr,
		restoreGeneratedOutputEntries(outputDirectory, backups),
	)
}

func restoreGeneratedOutputEntries(
	outputDirectory string,
	backups []generatedOutputEntryBackup,
) error {
	var restoreErr error
	for _, backup := range backups {
		target := filepath.Join(outputDirectory, backup.name)
		if err := os.Rename(backup.path, target); err != nil {
			restoreErr = errors.Join(
				restoreErr,
				fmt.Errorf("cannot restore the previous generated output entry (%s): %w", target, err),
			)
		}
	}

	return restoreErr
}
