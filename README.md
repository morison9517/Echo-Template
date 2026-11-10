# Echo Template Demo

Echo(Go)でWebアプリを作るための開発テンプレート。
**`docker compose up` だけで、アプリとDBが揃った開発環境が立ち上がります。**

ログイン・エラーページ・本番用の構成まで入っているので、**作りたいものの中身から書きはじめられます。**
コードには「なぜそう書くか」の説明を付けてあり、Goが初めての人でも読み進められます。

> リポジトリ名は `case_echo`、画面上の表示名は `Echo Template Demo` です。
> プロダクト名が決まったら、表示名は `internal/view/page.go` の
> `siteName` の1か所を直してください(全ページのタイトルとヘッダーに反映されます)。

---

## いきなり動かす

```bash
# 1. 金庫を作る(初回だけ)
cp .env.example .env          # Windows: Copy-Item .env.example .env

# 2. 起動する
docker compose up --build
```

→ <http://localhost:1323> を開く

**DBに表を作るコマンドはありません。** 起動するたびに自動で用意されます。

うまくいかないときは **[docs/SETUP.md](docs/SETUP.md)** を見てください。困ったときの対処が全部書いてあります。

---

## 最初に出るデモページについて

起動して `/` を開くと「セットアップ完了 🎉」というデモページが出ます。
**これは消さなくて大丈夫です。** DjangoやRailsの初期画面と同じ仕組みで、条件を満たすと自動で出なくなります。

| | |
| --- | --- |
| 出る条件 | 開発モード **かつ** `internal/handlers/page.go` にまだ `"/"` が無いとき |
| 消える条件 | `page.go` の `e.GET("/", index)` のコメントを外す(それだけ) |
| 本番 | `APP_ENV=production` では最初から出ない。デモ用の表(`demo_todos`)もDBに作られない |
| あとで見たい | `/__demo` で開ける(開発モードのときだけ) |

```go
// internal/handlers/page.go のこの行のコメントを外した瞬間、デモは出なくなります
func RegisterPageRoutes(e *echo.Echo) {
	e.GET("/", index)
	e.GET("/health", health)
}
```

### ⚠ Echo版だけの注意点:同じURLを2回登録すると、黙って上書きされる

**Echoは同じURLを2回登録しても、エラーを出しません。後から登録したほうが勝ちます。**

| | 同じURLを2回登録すると |
| --- | --- |
| Flask / Django | 先に登録したほうが勝つ |
| Gin | 起動した瞬間に落ちる |
| **Echo** | **後に登録したほうが、何も言わずに上書きする** |

デモは最後に取り付けるので、何もしなければ**デモが自分たちのトップページを上書きします。**
そのため、デモ側は「`/` が空いているかどうか」を必ず先に確認してから登録しています。

```go
// internal/demo/demo.go
if !hasRoute(e, http.MethodGet, "/") {
	e.GET("/", index)   // ← 空いているときだけ引き受ける
}
```

この `hasRoute()` の確認を消すと、**自分たちのトップページを作ってもデモが出続けます。**
エラーが出ないぶん原因に気づきにくいので、`internal/demo/` を触るときはここだけ残してください。

> 同じことは自分たちのコードでも起きます。「URLを足したのに、前からある画面が出なくなった」
> ときは、**同じURLをどこかでもう一度登録していないか**を疑ってください。

### デモの画面は1枚で完結しています

デモの画面(`internal/demo/page.html`)は、`base.html` も `style.css` も `main.js` も使いません。見た目も動きも全部そのファイルの中に入っています。

デモの役目は「セットアップが動いているか」を見せる**計器**です。計器が `base.html` に頼っていると、`base.html` を自分たちの見た目に作り替えた日に、この画面まで一緒に壊れます。しかも穴(block)の名前を変えただけだとエラーも出ず、真っ白になるだけなので「アプリが壊れた?」と勘違いします。1枚完結にしておけば、`base.html` を好きなだけ作り替えても、この計器だけは最後まで正しく動きます。

そのため `page.html` は `web/templates/pages/` ではなく `internal/demo/` に置いてあります(あそこに置くと共通の仕組みが自動で拾って `base.html` と合体させてしまうため)。読み込みは `go:embed` でアプリ本体に取り込んでいます。

ページの書き方の見本は `web/templates/pages/login.html` を見てください(`base.html` を継いだ本物のページです)。

### デモが本当に不要になったら(2か所)

1. `internal/demo/` フォルダを削除
2. `internal/router/router.go` の「デモ」の数行を削除

---

## 使っている技術

| 分類 | 技術 |
| --- | --- |
| フロント | 素のHTML / CSS / JavaScript(Goの標準テンプレート) |
| バック | Go 1.26 / Echo v5 / GORM |
| DB | MySQL 8.4(確認は DBeaver、ポートは **3310**) |
| 環境 | Docker Compose / air(保存したら自動で作り直す道具) |
| 本番 | Nginx / AWS(**`compose.prod.yml` に構成済み。本番イメージは約32MB**) |

---

## フォルダの地図

「アプリ = 1軒のお店」だと思って読んでください。

```
case_echo/
├── cmd/server/main.go      お店の開店作業(起動の入口)
│
├── internal/               お店の裏側(★他のプロジェクトからは入れない場所)
│   ├── config/             設定を1か所に集める
│   ├── database/           DBに繋ぐ共用の道具
│   ├── models/             データの形(DBの表)を決める
│   ├── middleware/         入口に置く仕掛け(セッション・ログイン確認・成りすまし対策・記録)
│   ├── handlers/           受付。URL → 処理
│   │   ├── page.go           画面(HTML)を返す
│   │   ├── auth.go           ログイン・新規登録
│   │   ├── api.go            JavaScript向けにデータだけ返す
│   │   ├── error.go          エラー画面(404など)。文言の表はここ
│   │   └── loginlimit.go     ログインの失敗回数を数えて締め出す
│   ├── demo/               動作確認用のデモ(開発モード限定・触らない)
│   │                       画面(page.html)も1枚完結でここに入っている
│   ├── router/             受付をまとめて組み立てる(土台)
│   └── view/               型紙(base.html)を使う仕組み(土台)
│
├── web/                    お客さんの目に入るもの
│   ├── templates/
│   │   ├── layouts/base.html   全ページ共通の型紙
│   │   ├── pages/              各ページの中身(ここに置くだけで使える)
│   │   │   └── error.html        404などで使う共通の1枚
│   │   └── partials/           型紙が長くなったら部品を切り出す置き場
│   └── static/                 CSS / JS / 画像
│       └── robots.txt          検索エンジンへの案内(サイト直下に出している)
│
├── media/                  利用者が上げたファイル(★中身はGitHubに上げない)
│
├── docs/                   手順書(SETUP / Gohelp / DEPLOY)
├── tools/                  開発中だけ使う小道具スクリプト
│
├── compose.yml             アプリとDBをまとめて動かす段取り表(開発用)
├── compose.prod.yml        本番用の段取り表(★開発中は使わない)
├── docker/nginx/           本番のNginxの設定(3枚)
│   ├── app.inc               中身の設定。下の2枚から読み込む1枚
│   ├── prod.conf             HTTPのまま動かすとき(最初はこちら)
│   └── prod-https.conf       証明書を取った後
├── Dockerfile              箱を組み立てるレシピ
├── .air.toml               保存したら自動で作り直す設定
├── go.mod / go.sum         買い物リストとレシート(全員が同じ部品を使うための記録)
│
├── .github/dependabot.yml  部品の新しい版が出たら知らせてもらう設定
├── .vscode/                共通のエディタ設定
│
├── LICENSE                 使ってよい条件(MIT)
├── .env                    金庫(★GitHubに上げない)
└── .env.example            金庫の中身の見本(こちらは上げる)
```

> **`internal` という名前について**
> Goでは `internal` という名前のフォルダは「このプロジェクトの中からしか使えない」という
> 決まりになっています。お店のバックヤードのようなもので、外から勝手に触られません。
> 特別な設定は不要で、名前を付けるだけでそうなります。

---

## 担当ごとに触る場所

複数人で作るときに、**他の人と同じファイルを触らずに済むよう分けてあります。** これでコンフリクト(変更の取り合い)がほぼ起きません。

| 担当 | 触る場所 |
| --- | --- |
| 見た目 | `web/templates/` `web/static/css/` |
| 画面の動き | `web/static/js/` |
| データの形 | `internal/models/` |
| URLと処理(画面) | `internal/handlers/page.go` |
| URLと処理(データ) | `internal/handlers/api.go` |
| ログイン | `internal/handlers/auth.go` |
| エラーページの文言 | `internal/handlers/error.go` |
| サイト名・説明文 | `internal/view/page.go` |

`main.go` `router/` `view/` `config/` `database/` `middleware/` `compose.yml` `Dockerfile` は**土台**です。
複数人で作っているときは、**先に共有してから**変更してください(全員に影響します)。

---

## ページを1枚増やす手順

1. **`web/templates/pages/` にHTMLを1枚置く**(`login.html` をコピーするのが早い)

   ```html
   {{ define "content" }}
   <h1>マイページ</h1>
   {{ end }}
   ```

   > ヘッダーとフッターは書きません。`base.html` から自動的に付きます。

2. **`internal/handlers/page.go` に2つ足す**

   ```go
   func RegisterPageRoutes(e *echo.Echo) {
       e.GET("/health", health)
       e.GET("/mypage", myPage)   // ← 1行足す
   }

   func myPage(c *echo.Context) error {  // ← 関数を書く
       return c.Render(http.StatusOK, "mypage.html", view.Page(c, view.Data{
           "Title": "マイページ",
       }))
   }
   ```

3. 保存して数秒待つ → <http://localhost:1323/mypage>

**登録作業はこれだけです。** テンプレートの一覧に書き足す必要はありません(自動で見つけます)。

ログインしている人だけに見せたいときは、後ろに1つ足すだけです。

```go
e.GET("/mypage", myPage, middleware.RequireLogin())
```

「見せられない」「見つからない」ときは、エラーを返すだけでエラーページになります。

```go
return echo.ErrNotFound   // 404 の画面が出る
```

---

## base.html(型紙)の仕組み

**初めてだと戸惑いやすい考え方なので、ここだけ先に押さえてください。**

ヘッダーとフッターを全ページにコピペすると、直すときに全ファイルを回ることになります。
そこで**共通部分を1枚の「型紙」にまとめ、各ページは真ん中の中身だけを書く**形にしています。

```
base.html(型紙)                    pages/index.html(中身)
┌──────────────────┐
│ ヘッダー          │
├──────────────────┤
│                  │  ←──  {{ define "content" }}
│  ここが穴         │            <h1>ホーム</h1>
│                  │        {{ end }}
├──────────────────┤
│ フッター          │
└──────────────────┘
```

- 型紙側:`{{ block "content" . }}{{ end }}` と書いた場所が**穴**になる
- ページ側:`{{ define "content" }} ～ {{ end }}` に書いた中身が**その穴に入る**
- 名前(`content`)が両者で一致していることが条件

穴は3つ用意してあります。

| 穴の名前 | 用途 |
| --- | --- |
| `content` | ページの本体(必須) |
| `head_extra` | そのページだけで使うCSSを足したいとき |
| `scripts` | そのページだけで使うJSを足したいとき |

**ヘッダーを直したいときは `base.html` を1枚直すだけ**で、全ページに反映されます。

> ★つまずきやすい点が2つあります。詳しくは `web/templates/layouts/base.html` の
> 冒頭のコメントに書いてあるので、最初に一度読んでください。
>
> 1. `{{ .Title }}` のように書いても、**渡していない情報は空になる**(エラーも出ない)
> 2. `{{ range }}` の中では**ドットの意味が変わる**(全体の情報は `{{ $.SiteName }}`)

---

## よく使うコマンド

| やりたいこと | コマンド |
| --- | --- |
| 起動する | `docker compose up` |
| 裏で起動する | `docker compose up -d` |
| 止める | `docker compose down` |
| エラーを見る | `docker compose logs -f web` |
| DBを作り直す(データは消える) | `docker compose exec web go run ./cmd/server -reset-db` |
| 箱の中に入る | `docker compose exec web bash` |
| ライブラリを追加する | `docker compose exec web go get <ライブラリ>` |
| 書き方を自動で整える | `docker compose exec web go fmt ./...` |
| 怪しい書き方を調べる | `docker compose exec web go vet ./...` |

---

## 公開するときに要るものは、入れてあります

**手元で動かすだけなら要らないが、外に出すと必ず要るもの**を最初から入れてあります。
どれも**設定しなくても動き**、**要らなければファイルを消せば外れます。**

| 入っているもの | 設定しないとどうなるか | いじる場所 |
| --- | --- | --- |
| **エラーページ**(404/500) | そのまま使える | 文言の表1つ(`internal/handlers/error.go`) |
| **ログイン試行制限** | そのまま使える(5回失敗で15分) | 回数と時間の定数(`internal/handlers/loginlimit.go`) |
| **検索・SNSでの見え方** | そのまま使える。説明文は書き換える | サイト名と説明文(`internal/view/page.go`) |
| **成りすまし対策(CSRF)** | そのまま使える(Echo付属のもの) | 設定(`internal/middleware/csrf.go`) |
| **CSS・JSの版の印** | そのまま使える。直したCSSが利用者の画面に必ず届く | `{{ static "css/style.css" }}` と書く(`internal/view/static.go`) |
| **HTTPS** | HTTPで動く。設定は用意済み | `.env` に `HTTPS_DOMAIN` を1行([docs/DEPLOY.md](docs/DEPLOY.md)) |
| **部品の更新の知らせ** | GitHubに上げれば月1回届く | `.github/dependabot.yml`(要らなければ消す) |

> エラーページは、存在しないURL(例 `/hogehoge`)を開けば確認できます。

---

## 3つのテンプレートの対応表

同じ構成・同じ画面で作ってあるので、1つ分かれば他も読めます。

| やること | Gin版 | Echo版 | Django版 |
| --- | --- | --- | --- |
| 起動の入口 | `cmd/server/main.go` | `cmd/server/main.go` | `manage.py` + `config/` |
| 設定 | `internal/config/` | `internal/config/` | `config/settings.py` |
| データの形 | `internal/models/` | `internal/models/` | `main/models.py` |
| 画面を返す | `handlers/page.go` | `handlers/page.go` | `main/views.py` |
| ログイン | `handlers/auth.go` | `handlers/auth.go` | `accounts/views.py` |
| ログイン試行制限 | `handlers/loginlimit.go` | `handlers/loginlimit.go` | `accounts/loginlimit.py` |
| エラーページ | `handlers/error.go` | `handlers/error.go` | `main/errors.py` |
| 検索・SNS対応 | `internal/view/page.go` | `internal/view/page.go` | `main/context_processors.py` |
| 型紙 | `base.html`(Go) | `base.html`(Go) | `base.html`(Django) |
| 成りすまし対策 | 自前で実装 | Echo付属 | Django付属 |
| 表を作る | 起動時に自動 | 起動時に自動 | 起動時に自動 |
| 表の形を変える | 列の追加のみ可 | 列の追加のみ可 | **データを保ったまま変更可** |
| 管理画面 | 無い | 無い | **`/admin/`** |
| アプリのポート | 8080 | 1323 | 8000 |
| DBのポート | 3308 | 3310 | 3309 |
| 本番イメージ | 48MB | **32MB** | 913MB |

> ポートをずらしてあるので、**同時に起動しても衝突しません**(Flask版はアプリ 5000 / DB 3307)。

> **Flask版は入門用として別扱いです。** はじめてのWebアプリを短期間で作ることに絞ってあり、
> この表とは構成が揃っていません(<https://github.com/morison9517/Flask-Template>)。

> **Gin版とEcho版は、フォルダ構成も画面(HTML/CSS/JS)もまったく同じです。**
> 違うのは Go のコードの書き方だけなので、2つを並べるとフレームワークの違いだけが見えます。
> 違いの早見表は [docs/Gohelp.md](docs/Gohelp.md) の最後にあります。

---

## ドキュメント

- **[docs/SETUP.md](docs/SETUP.md)** — 環境構築、日々の操作、DBeaverでの接続、困ったときの対処
- **[docs/Gohelp.md](docs/Gohelp.md)** — Goの書き方(コーディング経験者向けの早わかり)
- **[docs/DEPLOY.md](docs/DEPLOY.md)** — 本番に出す手順

---

## ライセンス

MIT License([LICENSE](LICENSE))

自由に使って、改造して、公開して構いません(商用も可)。
条件は「著作権表示を残すこと」だけです。
