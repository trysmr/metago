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

// 所有権の確認と入れ替えを同じ定義から駆動し、形式を増やしたときに
// 片方だけ追従し忘れて利用者のディレクトリを消す事故を防ぐ。
var generatedDirectoryNames = []string{markdownDirectoryName, htmlDirectoryName}

func WriteDocumentation(
	outputDirectory string,
	objects []model.ObjectDefinition,
	links *salesforceurl.Builder,
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

	markdownDirectory := filepath.Join(stagingDirectory, markdownDirectoryName)
	if err := writeMarkdown(markdownDirectory, objects, links, availableObjects); err != nil {
		return fmt.Errorf("cannot generate Markdown: %w", err)
	}

	htmlDirectory := filepath.Join(stagingDirectory, htmlDirectoryName)
	if err := writeHTML(htmlDirectory, objects, links, availableObjects); err != nil {
		return fmt.Errorf("cannot generate HTML: %w", err)
	}

	if err := replaceGeneratedDirectories(
		outputDirectory,
		stagingDirectory,
		generatedDirectoryNames,
	); err != nil {
		preserveStagingDirectory = true
		return fmt.Errorf(
			"cannot swap the generated directories; kept the staging directory (%s): %w",
			stagingDirectory,
			err,
		)
	}

	// 入れ替えが成功して初めて所有を記録する。失敗した回にマーカーだけが残ると、
	// 次回以降は所有済みと見なされ、利用者が置いたディレクトリを警告なく消してしまう。
	return markOutputDirectory(outputDirectory)
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

type generatedDirectoryBackup struct {
	name string
	path string
}

func replaceGeneratedDirectories(
	outputDirectory string,
	stagingDirectory string,
	directoryNames []string,
) error {
	backups := make([]generatedDirectoryBackup, 0, len(directoryNames))

	for _, name := range directoryNames {
		target := filepath.Join(outputDirectory, name)
		if _, err := os.Stat(target); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}

			return fmt.Errorf("既存の生成先を確認できません（%s）: %w", target, err)
		}

		backup := filepath.Join(stagingDirectory, "previous-"+name)
		if err := os.Rename(target, backup); err != nil {
			rollbackErr := restoreGeneratedDirectories(outputDirectory, backups)
			return errors.Join(
				fmt.Errorf("cannot move the existing generated directory aside (%s): %w", target, err),
				rollbackErr,
			)
		}

		backups = append(backups, generatedDirectoryBackup{
			name: name,
			path: backup,
		})
	}

	installed := make([]string, 0, len(directoryNames))
	for _, name := range directoryNames {
		source := filepath.Join(stagingDirectory, name)
		target := filepath.Join(outputDirectory, name)
		if err := os.Rename(source, target); err != nil {
			rollbackErr := rollbackGeneratedDirectories(
				outputDirectory,
				installed,
				backups,
			)
			return errors.Join(
				fmt.Errorf("cannot swap the generated directory (%s): %w", target, err),
				rollbackErr,
			)
		}

		installed = append(installed, name)
	}

	return nil
}

func rollbackGeneratedDirectories(
	outputDirectory string,
	installed []string,
	backups []generatedDirectoryBackup,
) error {
	var rollbackErr error
	for _, name := range installed {
		path := filepath.Join(outputDirectory, name)
		if err := os.RemoveAll(path); err != nil {
			rollbackErr = errors.Join(
				rollbackErr,
				fmt.Errorf("cannot remove the new generated directory (%s): %w", path, err),
			)
		}
	}

	return errors.Join(
		rollbackErr,
		restoreGeneratedDirectories(outputDirectory, backups),
	)
}

func restoreGeneratedDirectories(
	outputDirectory string,
	backups []generatedDirectoryBackup,
) error {
	var restoreErr error
	for _, backup := range backups {
		target := filepath.Join(outputDirectory, backup.name)
		if err := os.Rename(backup.path, target); err != nil {
			restoreErr = errors.Join(
				restoreErr,
				fmt.Errorf("cannot restore the previous generated directory (%s): %w", target, err),
			)
		}
	}

	return restoreErr
}
