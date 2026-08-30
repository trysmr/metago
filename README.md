# metago

[![CI](https://github.com/trysmr/metago/actions/workflows/ci.yml/badge.svg)](https://github.com/trysmr/metago/actions/workflows/ci.yml)

Generate Markdown and HTML documentation from local Salesforce metadata XML.

metago reads the `objects` directory of a Salesforce DX project and produces a browsable reference of every object and field. It never connects to a Salesforce org — it only reads files you already have on disk.

日本語版は[README.ja.md](README.ja.md)にあります。

## Features

- Generates an object index in both Markdown and HTML
- Generates a detail page per object
- Shows field type, length, precision, scale, referenced objects, and relationship name
- Shows required, unique, external ID, encrypted, and history-tracking as separate columns
- Links to referenced objects when they are part of the same run
- Links to the Salesforce setup screens when `--org-url` is given
- Writes to a staging directory first, then swaps it with the previous output

## Supported metadata

metago parses these files:

- `<ObjectApiName>.object-meta.xml`
- `fields/*.field-meta.xml`

It expects the standard Salesforce DX source layout:

```text
force-app/main/default/objects/
├── Account/
│   └── fields/
│       └── CustomField__c.field-meta.xml
└── Invoice__c/
    ├── Invoice__c.object-meta.xml
    └── fields/
        ├── Amount__c.field-meta.xml
        └── Status__c.field-meta.xml
```

When an object XML is absent — as is common for standard objects — the directory name is used as the API name.

**From the object XML**: API name, label, plural label, deployment status, sharing model, activities, reports, history tracking.

**From the field XML**: API name, label, type, required, unique, external ID, encrypted, history tracking, length, precision, scale, referenced objects, relationship name, formula.

Formulas are wrapped in an HTML `<code>` element in Markdown as well. Line breaks become `<br>`, so the Markdown table stays on one row while the formula still breaks where it does in the XML. In HTML the line breaks are kept as they are.

## Installation

Requires Go 1.26 or later.

```sh
go install github.com/trysmr/metago/cmd/metago@latest
```

The binary is placed in `$GOBIN`, or `$HOME/go/bin` when that is not set. Make sure the directory is on your `PATH`.

If you cannot install Go, prebuilt archives for Linux, macOS, and Windows on amd64 and arm64 are published on [GitHub Releases](https://github.com/trysmr/metago/releases). On macOS, Gatekeeper blocks binaries downloaded through a browser: fetch the archive with `curl`, or clear the attribute with `xattr -d com.apple.quarantine metago`.

## Usage

Point metago at an `objects` directory and an output directory.

```sh
metago \
  --input /path/to/force-app/main/default/objects \
  --output /path/to/output
```

Pass your org's base URL to also link to the Salesforce setup screens.

```sh
metago \
  --input /path/to/force-app/main/default/objects \
  --output /path/to/output \
  --org-url https://example.my.salesforce.com
```

### Options

| Option      | Required | Description                                    |
| ----------- | -------- | ---------------------------------------------- |
| `--input`   | Yes      | Salesforce `objects` directory                 |
| `--output`  | Yes      | Output directory                               |
| `--org-url` | No       | Base URL of the Salesforce org                 |
| `--lang`    | No       | Language of the generated headings: `en` (default) or `ja` |
| `--version` | No       | Print the version and exit                     |
| `--help`    | No       | Print the option list and exit                 |

`--org-url` accepts an HTTPS scheme and host only. URLs carrying a path, query, fragment, or credentials are rejected.

`--lang` picks the wording of the generated headings. Both sets follow the Salesforce setup screens in that language — `Fields & Relationships` in English, `項目とリレーション` in Japanese — so the generated reference lines up with the org you are looking at.

### Running on Windows

Messages are written as UTF-8. The classic Command Prompt does not use UTF-8 as its default code page, so Japanese text may be garbled. Switch the code page before running:

```bat
chcp 65001
```

Windows Terminal, PowerShell, and the Visual Studio Code terminal need no such step. The generated Markdown and HTML are always UTF-8, so only the on-screen output is affected.

## Output

metago creates one directory per format, plus a marker file:

```text
output/
├── .metago-generated
├── markdown/
│   ├── index.md
│   ├── Account.md
│   └── Invoice__c.md
└── html/
    ├── index.html
    ├── Account.html
    └── Invoice__c.html
```

`index.md` and `index.html` hold the object index. Each per-object file holds the object attributes and the field table.

With `--org-url`, links to the object detail and "Fields & Relationships" setup screens are included. In HTML those links open in a new tab.

## How the output directory is replaced

On a successful run metago writes `.metago-generated` into the output directory. From then on it replaces `markdown/` and `html/` wholesale, but only while that marker is present. Other files sitting directly in the output directory are left alone.

If `markdown/` or `html/` already exists without the marker, metago refuses to run so that it cannot delete files you put there. Move the existing directory aside after reviewing it, or point `--output` somewhere empty.

The input and output directories must be separate. metago rejects the same directory and any arrangement where one directory is inside the other, including paths that overlap through symbolic links.

If the swap itself fails, metago restores the previous output and keeps the staging directory so you can inspect it.

## Development

You can run the tool straight from a checkout:

```sh
go run ./cmd/metago \
  --input /path/to/force-app/main/default/objects \
  --output /path/to/output
```

After making a change:

```sh
gofmt -l .
go vet ./...
go test -race ./...
go build ./cmd/metago
```

## License

[MIT License](LICENSE)
