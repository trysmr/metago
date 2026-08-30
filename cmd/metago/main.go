package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
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

	if err := render.WriteDocumentation(*output, objects, links); err != nil {
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

func objectCountLabel(count int) string {
	if count == 1 {
		return "1 object"
	}

	return fmt.Sprintf("%d objects", count)
}
