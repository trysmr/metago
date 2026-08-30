package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/trysmr/metago/internal/metadata"
	"github.com/trysmr/metago/internal/render"
	"github.com/trysmr/metago/internal/salesforceurl"
)

const developmentVersion = "dev"

// リリースビルドは-ldflags -Xで差し替える。
var version = developmentVersion

// テストから差し替えるためパッケージ変数にしている。
var readBuildInfo = debug.ReadBuildInfo

func resolveVersion() string {
	info, ok := readBuildInfo()
	if !ok {
		return version
	}

	return selectVersion(version, info.Main.Version)
}

// go installは-ldflagsを適用しない代わりに、モジュールへタグを記録する。
// (devel)とv0.0.0-はタグのないビルドを指すため、リリースとしては扱わない。
func selectVersion(injected string, moduleVersion string) string {
	if injected != developmentVersion {
		return injected
	}

	if moduleVersion == "" ||
		moduleVersion == "(devel)" ||
		strings.HasPrefix(moduleVersion, "v0.0.0-") {
		return developmentVersion
	}

	return moduleVersion
}

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return
		}

		log.Fatal(err)
	}
}

func run(args []string, stdout io.Writer, stderr io.Writer) error {
	flags := flag.NewFlagSet("metago", flag.ContinueOnError)
	flags.SetOutput(stderr)

	showVersion := flags.Bool(
		"version",
		false,
		"print the version and exit",
	)
	input := flags.String(
		"input",
		"",
		"Salesforce objects directory",
	)
	output := flags.String(
		"output",
		"",
		"output directory (markdown and html are replaced on every run once the marker exists)",
	)
	orgURL := flags.String(
		"org-url",
		"",
		"base URL of the Salesforce org",
	)
	language := flags.String(
		"lang",
		string(render.LanguageEnglish),
		"language of the generated headings (en or ja)",
	)
	if err := flags.Parse(args); err != nil {
		return err
	}

	if *showVersion {
		fmt.Fprintf(stdout, "metago %s\n", resolveVersion())

		return nil
	}

	if *input == "" {
		return errors.New("--input is required")
	}
	if *output == "" {
		return errors.New("--output is required")
	}
	if err := validateSeparatePaths(*input, *output); err != nil {
		return err
	}

	documentLanguage, err := render.ParseLanguage(*language)
	if err != nil {
		return err
	}

	objects, err := metadata.LoadObjects(*input)
	if err != nil {
		return err
	}

	var links *salesforceurl.Builder
	if *orgURL != "" {
		builder, err := salesforceurl.NewBuilder(*orgURL)
		if err != nil {
			return err
		}

		links = &builder
	}

	if err := render.WriteDocumentation(*output, objects, links, documentLanguage); err != nil {
		return err
	}

	fmt.Fprintf(
		stdout,
		"Generated Markdown and HTML for %s: %s\n",
		objectCountLabel(len(objects)),
		*output,
	)

	return nil
}

func validateSeparatePaths(inputPath string, outputPath string) error {
	resolvedInput, err := resolvePath(inputPath)
	if err != nil {
		return fmt.Errorf("cannot resolve the input directory (%s): %w", inputPath, err)
	}

	resolvedOutput, err := resolvePath(outputPath)
	if err != nil {
		return fmt.Errorf("cannot resolve the output directory (%s): %w", outputPath, err)
	}

	inputContainsOutput, err := pathContains(resolvedInput, resolvedOutput)
	if err != nil {
		return err
	}
	outputContainsInput, err := pathContains(resolvedOutput, resolvedInput)
	if err != nil {
		return err
	}
	if inputContainsOutput || outputContainsInput {
		return fmt.Errorf(
			"the input and output directories must not overlap (%s, %s)",
			inputPath,
			outputPath,
		)
	}

	return nil
}

// 存在しない生成先も、存在する最寄りの親までシンボリックリンクを解決する。
// これにより、検証のために生成先を作成せずに入力との重なりを判定できる。
func resolvePath(path string) (string, error) {
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	currentPath := absolutePath
	missingNames := make([]string, 0)
	for {
		resolvedPath, err := filepath.EvalSymlinks(currentPath)
		if err == nil {
			for index := len(missingNames) - 1; index >= 0; index-- {
				resolvedPath = filepath.Join(resolvedPath, missingNames[index])
			}

			return filepath.Clean(resolvedPath), nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}

		parentPath := filepath.Dir(currentPath)
		if parentPath == currentPath {
			return "", err
		}

		missingNames = append(missingNames, filepath.Base(currentPath))
		currentPath = parentPath
	}
}

func pathContains(parentPath string, childPath string) (bool, error) {
	if !strings.EqualFold(filepath.VolumeName(parentPath), filepath.VolumeName(childPath)) {
		return false, nil
	}

	relativePath, err := filepath.Rel(parentPath, childPath)
	if err != nil {
		return false, fmt.Errorf(
			"cannot compare the input and output directories (%s, %s): %w",
			parentPath,
			childPath,
			err,
		)
	}

	return relativePath == "." ||
		(relativePath != ".." && !strings.HasPrefix(relativePath, ".."+string(filepath.Separator))), nil
}

func objectCountLabel(count int) string {
	if count == 1 {
		return "1 object"
	}

	return fmt.Sprintf("%d objects", count)
}
