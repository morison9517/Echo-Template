# Go早わかり — コーディング経験者向け

このファイルは**Goを一から学ぶ教科書ではありません。**

他の言語でプログラムを書いたことがあるなら、変数・条件分岐・繰り返し・クラスといった土台はもう持っています。足りないのは「Goだとどう書くか」だけです。ここはその**書き換え表**です(比べる相手には、学校で習うことの多いJavaとPHPを選んでいます)。

> 引っ越しに例えると、家具（考え方）はもう持っていて、新しい部屋の**間取りに合わせて置き直すだけ**です。家具を買い直す必要はありません。

**Javaをやった人には、Goはかなり近いです。** 型を書く、コンパイルする、という点が同じだからです。むしろPythonより馴染みやすいかもしれません。

分からなくなったら、この順で見てください。

1. まずこのファイルで書き方を確認する
2. 実際に動いているコードを読む（`internal/demo/` の中が全部お手本です）
3. それでもダメなら人に聞く

---

## 0. 最初の3つのショック

Goのコードを初めて見ると、たいてい同じ3つで面食らいます。**慣れれば全部いいことです。**

### ① 型を**後ろ**に書く

```java
String name = "太郎";        // Java:型が先
int count = 3;
```

```go
var name string = "太郎"     // Go:名前が先、型が後ろ
var count int = 3
```

慣れると読みやすくなります。「これは name という変数で、中身は文字列」と、日本語の語順に近くなるためです。

### ② `:=` で型を省略できる

上の書き方は実は長すぎて、普段は使いません。

```go
name := "太郎"       // 型は "太郎" から自動で決まる
count := 3
```

**関数の中ではほぼ全部これです。** `var` を書くのは、最初に値を入れないときだけです。

### ③ ★使っていない変数と import は**エラーになる**

ここが一番びっくりします。

```go
name := "太郎"       // ← 使わないとコンパイルが通らない
```

```
declared and not used: name
```

Javaなら警告止まりですが、**Goはコンパイル自体を拒否します。**

> 厳しく見えますが、これのおかげで「消し忘れたゴミコード」が絶対に残りません。
> 複数人で書いていると、消し忘れは必ず起きます。それを機械が止めてくれます。
>
> ★デバッグ中に一時的に困ったら、`_ = name` と書けば「使ったこと」になります。

---

## 1. 変数

| | Java | Go |
| --- | --- | --- |
| 宣言 | `String name = "太郎";` | `name := "太郎"` |
| 型を明示 | `int count = 3;` | `var count int = 3` |
| 値なしで宣言 | `String name;` | `var name string` |
| 定数 | `final int MAX = 10;` | `const MAX = 10` |

### ★「空」がありません(ゼロ値)

Javaの `String name;` は `null` ですが、**Goは型ごとの初期値が入ります。**

| 型 | 初期値 |
| --- | --- |
| `string` | `""`（空文字） |
| `int` | `0` |
| `bool` | `false` |
| ポインタ・スライス・マップ | `nil` |

> `null` を踏んで落ちる、という事故が起きにくい仕組みです。
> その代わり「未入力」と「空文字」を区別したいときは工夫が要ります（後述のポインタ）。

---

## 2. 条件分岐

```go
if count > 10 {
    ...
} else if count > 5 {
    ...
} else {
    ...
}
```

Javaとほぼ同じですが、**2つ違います。**

- **条件をかっこで囲まない**
- **`{` は必ず同じ行**（次の行に書くとエラー）

`&&` `||` `!` はJavaと同じです。Pythonのように英語にはなりません。

### ★条件の前に1行書ける

Goでよく見る書き方です。

```go
if err := database.DB.Save(&todo).Error; err != nil {
    ...
}
```

これは「保存して、その結果を `err` に入れて、それが `nil` でなければ」という意味です。
`err` はこの `if` の中だけで生きるので、後ろの行を汚しません。

> `internal/demo/api.go` にたくさん出てきます。**Goで一番よく見る形**なので、
> これが読めるようになると一気に読めるようになります。

### `switch` は `break` が要りません

```go
switch category {
case "success":
    ...
case "error":
    ...
default:
    ...
}
```

Javaのように次のcaseへ落ちないので、`break` を書き忘れる事故がありません。

---

## 3. 繰り返し

**Goには `for` しかありません。** `while` も `do-while` もありません。

```java
for (int i = 0; i < 5; i++) { ... }      // Java
while (count < 10) { ... }
```

```go
for i := 0; i < 5; i++ { ... }           // Java と同じ形
for count < 10 { ... }                   // これが while の代わり
for { ... }                              // 無限ループ
```

### 中身を1つずつ取り出す

```go
for _, todo := range todos {
    fmt.Println(todo.Title)
}
```

`range` が「1つずつ取り出す」の意味です。**2つ受け取る**のが特徴で、
1つ目が番号、2つ目が中身です。

```go
for i, todo := range todos {     // 番号も欲しいとき
    fmt.Println(i, todo.Title)
}
```

> ★`_`（アンダースコア）は「これは使いません」の印です。
> 番号が要らないときにこう書きます。**Goは使わない変数を許さない**ので、
> 捨てると宣言するためのものです。

---

## 4. スライスとマップ ← ★ここが一番よく使う

Javaの配列・HashMapに当たるものです。

### スライス（Javaの `ArrayList`）

```go
todos := []string{"みかん", "りんご", "ぶどう"}

todos[0]                      // "みかん"
len(todos)                    // 3
todos = append(todos, "かき") // 末尾に追加
```

> ★`append` は「足した結果を返す」ので、**受け取り直す必要があります。**
> `append(todos, "かき")` だけ書いても `todos` は変わりません。
> これはよく踏みます。

### マップ（JavaのHashMap）

```go
counts := map[string]int{"みかん": 3, "りんご": 5}

counts["みかん"]              // 3
counts["ぶどう"]              // 0 ★無いキーでも落ちない。ゼロ値が返る
```

### ★「無い」と「0」を区別する

上の通り、無いキーを引いてもエラーになりません。**区別したいときは2つ受け取ります。**

```go
value, ok := counts["ぶどう"]
if !ok {
    // 無かった
}
```

> この「2つ目に ok を受け取る」形はGoの定番です。
> `internal/demo/api.go` の `findTodo(c)` も `(todo, ok)` を返しています。

---

## 5. 関数

```java
public String greet(String name) {       // Java
    return "こんにちは " + name;
}
```

```go
func greet(name string) string {         // Go:戻り値の型が最後
    return "こんにちは " + name
}
```

### ★戻り値を2つ返すのが普通です

Javaならクラスを作るしかない場面ですが、Goは並べるだけです。

```go
func checkDB() (string, string) {
    return "ok", ""
}

status, message := checkDB()
```

`internal/demo/demo.go` の `checkDB()` がこの書き方です。

**そして、これがGoのエラー処理の土台になります**（次の項）。

### ★大文字始まりが「公開」

Javaの `public` / `private` に当たるものが、Goには**ありません。**
代わりに**名前の1文字目**で決まります。

```go
func Register(...)     // 大文字 = 他のファイルから呼べる(public)
func hasRoute(...)     // 小文字 = このパッケージの中だけ(private)
```

> ★これはGoで**最も多い詰まり方**です。
> 「呼べるはずなのに undefined と言われる」ときは、まず1文字目を見てください。
>
> 変数も構造体の項目も同じルールです。項目名が小文字だと、
> **DBにも保存されず、JSONにも出ず、HTMLにも出ません。**
> `internal/models/user.go` の冒頭にも書いてあります。

---

## 6. 構造体とメソッド（Javaのクラス）

**Goにクラスはありません。** 代わりに「データの入れ物（構造体）」と「それに付ける関数（メソッド）」を別々に書きます。

```java
// Java
class User {
    String username;
    String show() { return this.username; }
}
```

```go
// Go
type User struct {
    Username string
}

func (u User) Show() string {     // ← (u User) が「Userに付ける」の印
    return u.Username
}
```

対応はこうです。

| Java | Go |
| --- | --- |
| `class` | `type ... struct` |
| `this` | `u`（自分で名前を決める） |
| メソッドはクラスの中 | **メソッドは外に書く** |

### ★ `*` が付くかどうかで意味が変わります

```go
func (u User) Show() string { ... }        // コピーを触るだけ
func (u *User) SetPassword(...) { ... }    // ★本人を書き換える
```

`internal/models/user.go` の `SetPassword` には `*` が付いています。
**付け忘れると「なぜかパスワードが空のまま」になります。** コピーを書き換えて捨てているためです。

> 書類に例えると、`*` 無しは「コピーを渡されて赤入れした」、
> `*` 付きは「原本を渡されて赤入れした」です。
> 原本を直したいときは `*` が要ります。

### バッククォートの部分（タグ）

構造体の項目の後ろに付いている、この部分です。

```go
Username string `gorm:"size:80;uniqueIndex;not null" json:"username"`
```

**項目に貼る付箋**です。中身のデータには関係なく、道具に指示を出しています。

| | 誰への指示か |
| --- | --- |
| `gorm:"..."` | DBの表を作るとき |
| `json:"..."` | JavaScriptに渡すとき |

> ★`json:"-"` は「JSONに絶対に含めない」の意味です。
> `PasswordHash` に付いています。**これを外すと、APIの返事に
> パスワードが混ざって画面まで流れ出ます。**

---

## 7. 文字列

```go
"こんにちは " + name                       // + で繋げる(Javaと同じ)
fmt.Sprintf("こんにちは %s さん", name)    // 埋め込む
len(name)                                  // ★バイト数。日本語は3倍になる
len([]rune(name))                          // ★文字数はこちら
```

> ★日本語で `len()` を使うときは注意してください。
> 「200文字まで」のつもりが「200バイトまで（日本語なら66文字）」になります。
> `internal/demo/api.go` では `len([]rune(title))` と書いています。

よく使うもの:

```go
strings.TrimSpace(title)     // 前後の空白を削る
strings.ToLower(title)       // 小文字に
strconv.Itoa(3)              // 数字 → 文字
strconv.Atoi("3")            // 文字 → 数字(エラーも返る)
```

---

## 8. `nil` とエラー処理 ← ★Goで一番の違い

**Goには例外（try-catch）がありません。**

```java
try {                                    // Java
    save();
} catch (Exception e) {
    ...
}
```

```go
if err := save(); err != nil {           // Go
    ...
}
```

エラーは**戻り値として返ってきます。** だから `err != nil` のチェックが、コードのあちこちに出てきます。

```go
if err := database.DB.Save(&todo).Error; err != nil {
    // ★return を付けて、ここで必ず抜ける
    return c.JSON(http.StatusInternalServerError, errorJSON("更新に失敗しました。"))
}
```

> 最初は「同じ形が多すぎる」と感じますが、**エラーを握りつぶせない**という利点があります。
> Javaのように catch でまとめて受けると、どこで何が失敗したのか分からなくなります。
> Goは失敗しうる場所が一目で分かります。
>
> ★`return` を書き忘れると、エラーの返事を出したあとに処理が続いてしまい、
> 「返事が2回送られる」という警告が出ます。**必ずセットで書いてください。**

### `nil` の使いどころ

Javaの `null` に近いですが、使えるのは**ポインタ・スライス・マップ・エラー**だけです。

```go
Email *string      // ← * が付くと「未入力」を表せる
```

`internal/models/user.go` の `Email` がこれです。
メールアドレスは任意入力なので、**空文字とも違う「未入力」**を表せるようにしています。

---

## 9. パッケージと import

```go
package handlers          // ★ファイルの1行目。フォルダ名と揃える

import (
    "net/http"                          // Goに最初から入っているもの
    "github.com/labstack/echo/v5"       // 外から借りてきたもの
    "case_echo/internal/database"       // 自分たちのコード
)
```

> ★**使っていない import はコンパイルエラーになります。**
> VSCodeの拡張機能を入れていれば、保存時に自動で消してくれます。
>
> ★ライブラリを足すときは、手で書かずにコマンドを打ってください。
> `go.mod` と `go.sum` が自動で更新されます。
>
> ```bash
> docker compose exec web go get <ライブラリ名>
> ```

---

## 10. Echoのコードで見かける、Goっぽい書き方

### `&` と `*`

```go
database.DB.Save(&todo)     // & = 「原本の場所」を渡す
```

`&` は「この値がある場所」を渡す印です。**コピーではなく本体を渡したいときに使います。**
DBに保存するときは、保存後にIDを書き戻してもらう必要があるので `&` が要ります。

> 住所に例えると、`todo` は家そのもの、`&todo` は住所です。
> 「この家を改装して」と頼むときは、家を郵送するのではなく住所を伝えます。

### `defer`

```go
defer file.Close()
```

「**この関数が終わるときに実行して**」という予約です。
Javaの try-with-resources に当たり、閉じ忘れが起きません。

### `map[string]any{...}` と `view.Data{...}`

```go
return c.JSON(http.StatusOK, map[string]any{"status": "ok"})
```

`map[string]any` は**その場で作る辞書**です。JSONの形をそのまま書けます。
「名前は文字(`string`)、中身は何でも(`any`)」という意味です。

画面に値を渡すときの `view.Data{...}` は、これに短い別名を付けただけのものです
(`internal/view/page.go`)。

```go
return c.Render(http.StatusOK, "index.html", view.Page(c, view.Data{"Title": "ホーム"}))
```

### `c *echo.Context` と `error`

Echoのすべてのハンドラは、この形をしています。

```go
func health(c *echo.Context) error {
    return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}
```

`c` は**「今の1回のやり取り」が全部入った箱**です。誰が何を送ってきたか、
どう返事をするか、全部ここから触ります。

| よく使うもの | 意味 |
| --- | --- |
| `c.FormValue("username")` | フォームで送られてきた値 |
| `c.QueryParam("page")` | URLの `?page=2` の部分 |
| `c.Param("id")` | URLの `:id` の部分 |
| `c.Bind(&input)` | 送られてきたJSONを入れ物に流し込む |
| `c.JSON(200, ...)` | JSONで返事する |
| `c.Render(200, "index.html", ...)` | HTMLで返事する |
| `c.Redirect(302, "/")` | 別のURLへ飛ばす |

> ★**返事を作る行には必ず `return` を付けます。**
> Echoのハンドラは最後に `error` を返す決まりで、`c.JSON(...)` や `c.Render(...)` は
> 「うまく返せたか」をその `error` として返してくるからです。
> 付け忘れるとビルドが通らないので、書き忘れたまま動いてしまうことはありません。

### エラー画面を出したいときは、エラーを `return` するだけ

```go
if post.UserID != user.ID {
    return echo.ErrForbidden      // 403 の画面が出る
}
```

自分でエラー画面を組み立てる必要はありません。**返したエラーは1か所
(`internal/handlers/error.go`)に集まり、そこが画面にして返します。**
`echo.ErrNotFound`(404)、`echo.ErrBadRequest`(400)なども同じように使えます。

### ミドルウェアは「関数の後ろ」に書く

```go
e.GET("/mypage", myPage, middleware.RequireLogin())
```

「このURLはログイン中の人だけ」のような仕掛けは、**ハンドラの後ろ**に並べます。
(Ginを触ったことがある人へ:Ginは前に書きますが、Echoは後ろです)

---

## 11. ★つまずきポイント集

実際に一番よく起きる順です。**詰まったらまずここを見てください。**

### 1. `declared and not used`

**使っていない変数**があります。消すか、`_ = name` と書いてください。
import も同じです。

### 2. `undefined: xxx`

**関数名や項目名の1文字目が小文字**になっています。
他のファイルから呼ぶものは**大文字始まり**にしてください。

### 3. DBに保存されない / 画面に出ない / JSONに出ない

これも**1文字目が小文字**です。GORMもJSONもHTMLも、小文字の項目は見えません。

### 4. 保存したのに中身が変わっていない

メソッドに `*` が付いていません。

```go
func (u *User) SetPassword(...)     // ← この * が必要
```

### 5. `{` を次の行に書いた

```go
if count > 10
{                    // ★エラーになる
```

Goでは `{` を必ず同じ行に書きます。

### 6. `append` したのに増えていない

戻り値を受け取り直していません。

```go
todos = append(todos, "かき")    // ← 左辺が必要
```

### 7. 日本語の文字数が合わない

`len()` はバイト数です。文字数は `len([]rune(s))` を使ってください。

### 8. 画面が真っ白 / 途中で切れる

Goの話ではなくテンプレートの話ですが、**一番多い事故**です。
`{{ define "content" }}` の**外に書いたものは表示されません。**
エラーも出ないので気づきにくいです。詳しくは `web/templates/layouts/base.html` の冒頭を見てください。

---

## 12. 練習にちょうどいい場所

**読むだけで一番勉強になるのは `internal/demo/` の中です。** 動いているコードなので、間違いがありません。

| 見るファイル | 何が分かるか |
| --- | --- |
| `internal/demo/todo.go` | 構造体の書き方 / タグ / DBの表の作り方 |
| `internal/demo/api.go` | 関数 / エラー処理 / マップ / `c *echo.Context` |
| `internal/demo/demo.go` | 戻り値を2つ返す書き方 / HTMLに値を渡す方法 |
| `internal/models/user.go` | メソッド / ポインタ / 大文字小文字のルール |
| `internal/handlers/page.go` | 自分たちの画面を書く場所（見本がコメントで入っています） |

### 最初の練習メニュー

1. `internal/demo/api.go` の `listTodos` を読んで、**何をしているか日本語で説明できるようにする**
2. `internal/models/user.go` のコメントを参考に、構造体を1つ作ってみる
3. `internal/database/database.go` の `Migrate()` に1行足して、**DBeaverで実際に表ができたのを見る**
4. `internal/handlers/page.go` の `e.GET("/", index)` のコメントを外して、画面を1枚出す

**3番が特に大事です。** 自分の書いた構造体がDBの表になって現れるのを一度見ておくと、
その後の話が全部すっと入ります。

---

## おまけ:GoとPythonのどちらが向いているか

**同じものを作るなら、Goのほうがコードは長くなります。** エラー処理を毎回書くからです。

その代わり、**間違いの多くをコンパイラが止めてくれます。** 型の食い違い、使っていない変数、
returnの書き忘れ。Pythonなら動かしてみて初めて分かるものが、書いている途中で分かります。

- **短期間で形にするなら** Python（Django / Flask）
- **長く動かす・人数が増える・速度が要るなら** Go（Echo / Gin）

**どちらかが劣っているわけではありません。** このテンプレートは Flask版・Gin版・Django版と
同じ画面・同じ機能で作ってあるので、読み比べると違いだけが浮かび上がります。

### EchoとGinの違い(同じGoの中で)

| | Echo | Gin |
| --- | --- | --- |
| ハンドラの形 | `func(c *echo.Context) error` | `func(c *gin.Context)` |
| 返事の書き方 | `return c.JSON(...)` | `c.JSON(...)` のあと `return` |
| エラー画面 | エラーを `return` すると1か所に集まる | その場で自分で返す |
| 成りすまし対策(CSRF) | 最初から付いている | 自分で書く |
| 同じURLを2回登録 | 後のほうで黙って上書き | 起動時に落ちる |
| 使う部品の数 | 少ない | 多い |

**書き忘れに気づきやすいのはEcho、解説記事が多いのはGin**です。
