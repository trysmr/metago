# metago

[![CI](https://github.com/trysmr/metago/actions/workflows/ci.yml/badge.svg)](https://github.com/trysmr/metago/actions/workflows/ci.yml)

ローカルに取得済みのSalesforce Metadata XMLを解析し、オブジェクト定義と項目定義をMarkdownおよびHTMLへ変換するGo製CLIです。

Salesforce APIへの接続やMetadataの取得は行いません。処理対象はローカルファイルだけです。

English documentation is available in [README.md](README.md).

## 主な機能

- オブジェクト一覧のMarkdownとHTMLを生成
- オブジェクトごとの詳細ページを生成
- 項目の型、長さ、精度、小数点以下、参照先、リレーション名を表示
- 必須、一意、外部ID、暗号化、履歴管理を属性ごとの列に表示
- 生成対象に含まれる参照先オブジェクトへのリンクを生成
- `--org-url`指定時にSalesforce設定画面へのリンクを生成
- 一時ディレクトリで生成してから、以前の生成結果と入れ替え

## 対応するMetadata

現在は次のファイルを解析します。

- `<オブジェクトAPI名>.object-meta.xml`
- `fields/*.field-meta.xml`

想定するディレクトリ構成は次のとおりです。

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

標準オブジェクトなどでオブジェクトXMLが存在しない場合は、オブジェクトディレクトリ名をAPI名として扱います。

### オブジェクトXMLから読み取る情報

- API名
- 表示ラベル
- 複数形の表示ラベル
- リリース情報
- 共有モデル
- 活動
- レポート
- 項目履歴管理

### 項目XMLから読み取る情報

- API名
- 表示ラベル
- 型
- 必須
- 一意
- 外部ID
- 暗号化
- 履歴管理
- 長さ
- 精度
- 小数点以下
- 参照先
- リレーション名
- 数式

数式はMarkdownでもHTMLの`<code>`要素で囲みます。改行が含まれる場合は`<br>`へ変換するため、Markdownの表を1行に保ったまま、表示上はXMLと同じ位置で改行されます。HTMLでは`code`要素内にXMLの改行をそのまま維持します。

## インストール

Go 1.26以上が必要です。

```sh
go install github.com/trysmr/metago/cmd/metago@latest
```

`$GOBIN`、これを設定していない場合は`$HOME/go/bin`へ配置します。このディレクトリに`PATH`が通っていることを確認してください。

Goを用意できない場合は、[GitHub Releases](https://github.com/trysmr/metago/releases)でLinux、macOS、Windows向けにamd64とarm64のアーカイブを配布しています。macOSではブラウザ経由でダウンロードした実行ファイルをGatekeeperが止めるため、`curl`で取得するか、`xattr -d com.apple.quarantine metago`で隔離属性を外してください。

## 実行方法

解析対象の`objects`ディレクトリと生成先を指定します。

```sh
metago \
  --input /path/to/force-app/main/default/objects \
  --output /path/to/output
```

Salesforce設定画面へのリンクも生成する場合は、組織のベースURLを指定します。

```sh
metago \
  --input /path/to/force-app/main/default/objects \
  --output /path/to/output \
  --org-url https://example.my.salesforce.com
```

### オプション

| オプション  | 必須   | 説明                              |
| ----------- | ------ | --------------------------------- |
| `--input`   | はい   | Salesforceの`objects`ディレクトリ |
| `--output`  | はい   | 生成先ディレクトリ                |
| `--org-url` | いいえ | Salesforce組織のベースURL         |
| `--lang`    | いいえ | 生成する見出しの言語。`en`（既定）または`ja` |
| `--version` | いいえ | バージョンを表示して終了          |
| `--help`    | いいえ | オプション一覧を表示して終了      |

`--org-url`にはHTTPSのスキームとホストだけを指定してください。パス、クエリ、フラグメント、認証情報を含むURLは受け付けません。

`--lang`は生成する見出しの言語を選びます。どちらもSalesforceの設定画面の表記に合わせてあるため、生成した資料と設定画面をそのまま突き合わせられます。既定は英語なので、日本語の組織を扱う場合は`--lang ja`を指定してください。

### Windowsで実行する場合

進捗やエラーのメッセージはUTF-8で出力します。従来のコマンドプロンプトは既定のコードページがUTF-8ではないため、日本語が化けることがあります。その場合は実行前にコードページを切り替えてください。

```bat
chcp 65001
```

Windows TerminalやPowerShell、Visual Studio Codeのターミナルでは、この操作は不要です。生成するMarkdownとHTMLは実行環境にかかわらずUTF-8で書き出すため、影響を受けるのは画面表示だけです。

## 出力

指定した出力先に、形式別のディレクトリと管理用マーカーを作成します。

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

`index.md`と`index.html`にはオブジェクト一覧を出力します。オブジェクトごとのファイルには、オブジェクト情報と項目一覧を出力します。

`--org-url`を指定した場合は、オブジェクト詳細と「項目とリレーション」のSalesforce設定画面へのリンクも表示します。HTMLではSalesforceへのリンクを別タブで開きます。

## 出力先の安全な置き換え

生成に成功すると、出力先へ`.metago-generated`を作成します。2回目以降は、このマーカーがある場合だけ`markdown/`と`html/`を丸ごと置き換えます。出力先の直下にあるほかのファイルは変更しません。

マーカーがない出力先に`markdown/`または`html/`が存在する場合は、利用者のファイルを誤って失わないようエラーにします。既存のディレクトリを再利用するときは、その内容を確認したうえで別の場所へ退避するか、空の出力先を指定してください。

入力ディレクトリと出力ディレクトリは分けてください。同じディレクトリを指定した場合や、シンボリックリンクを経由する場合を含め、一方が他方の配下にある場合はエラーにします。

生成先の入れ替えに失敗した場合は、以前の生成結果の復元を試み、調査できるよう一時出力を保持します。

## 開発

リポジトリを取得したあとは、ビルドせずにそのまま実行できます。

```sh
go run ./cmd/metago \
  --input /path/to/force-app/main/default/objects \
  --output /path/to/output
```

変更を加えたら次を確認します。

```sh
gofmt -l .
go vet ./...
go test -race ./...
go build ./cmd/metago
```

## ライセンス

[MIT License](LICENSE)
