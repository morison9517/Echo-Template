// =============================================================================
// デモ用のAPI(画面ではなくデータを返す受付)
//
//	★これは動作確認用です。自分たちのAPIは internal/handlers/api.go に書きます。
//	  書き方の見本としてどうぞ。
//
// ▼ 画面を返す受付との違い
//
//	page.go … HTMLを丸ごと返す。ページが切り替わる。
//	api.go  … データだけ返す。ページを切り替えずに一部だけ書き換えられる。
//
//	使い分けの目安:
//	  ページ移動を伴う操作(ログイン、詳細ページへ移動) → page.go
//	  その場で追加・削除・チェック(いいね、Todo追加)   → api.go
//
// ▼ JavaScript側との対応
//
//	internal/demo/page.html の中のJSから呼ばれている。
//	送受信の作法(整理券を付ける、エラーを拾う)は main.js の api がやるので、
//	自分たちのページでは api.post("/api/todos", { title: "牛乳" }) と書くだけでよい。
//
// =============================================================================
package demo

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v5"

	"case_echo/internal/database"
)

// registerAPIRoutes = デモ用のURLを登録する。demo.go の Register() から呼ばれる。
func registerAPIRoutes(e *echo.Echo) {
	api := e.Group("/__demo/api")

	// ★ログイン必須にしたい場合は、この1行を足す:
	//     api.Use(middleware.RequireLoginAPI())
	//   今はログインしなくても試せるようにしてある。

	api.GET("/todos", listTodos)
	api.POST("/todos", createTodo)
	api.PATCH("/todos/:id", toggleTodo)
	api.DELETE("/todos/:id", deleteTodo)
}

// listTodos = 一覧を返す。
func listTodos(c *echo.Context) error {
	var todos []Todo

	// Order(...) で新しい順。Limit で取りすぎを防ぐ。
	// ★上限が無いと、データが増えたときに画面が固まる。
	if err := database.DB.Order("id DESC").Limit(100).Find(&todos).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, errorJSON("取得に失敗しました。"))
	}

	// ★ここで返す形が、そのままJavaScript側で受け取る形になる。
	//   項目名(id / title / is_done)は demo/todo.go の json タグで決まっている。
	return c.JSON(http.StatusOK, map[string]any{"todos": todos})
}

// createTodo = 1件追加する。
func createTodo(c *echo.Context) error {
	// ▼ 送られてきたJSONを受け取るための入れ物
	//
	//	{"title": "牛乳を買う"} という形で届くので、
	//	json:"title" で「JSONの title をここに入れる」と指示している。
	//	この関数の中でしか使わないので、ここで定義している。
	var input struct {
		Title string `json:"title"`
	}

	// Bind = 届いたJSONを上の入れ物に流し込む。
	// 形が違えばエラーになるので、変な値のまま進まずに済む。
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, errorJSON("送信内容の形式が正しくありません。"))
	}

	title := strings.TrimSpace(input.Title)

	// ★入力チェックは必ずサーバー側でもやる。
	//   画面側(HTMLのrequiredやJS)のチェックは、開発者ツールから素通りできる。
	if title == "" {
		return c.JSON(http.StatusBadRequest, errorJSON("内容を入力してください。"))
	}
	if len([]rune(title)) > 200 {
		return c.JSON(http.StatusBadRequest, errorJSON("200文字以内で入力してください。"))
	}

	// ★デモのTodoは持ち主を持たない(本番のUserに紐付けない)。
	//   理由は todo.go のコメント参照。紐付けの書き方も同じ場所に見本がある。
	todo := Todo{Title: title}

	if err := database.DB.Create(&todo).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, errorJSON("保存に失敗しました。"))
	}

	// 201 = 「新しく作った」を表す返事。
	return c.JSON(http.StatusCreated, map[string]any{"todo": todo})
}

// toggleTodo = 済み / 未済 を切り替える。
func toggleTodo(c *echo.Context) error {
	// c.Param("id") = URLの :id の部分。/__demo/api/todos/3 なら "3"。
	todo, err := findTodo(c)
	if err != nil {
		return c.JSON(http.StatusNotFound, errorJSON("見つかりませんでした。"))
	}

	todo.IsDone = !todo.IsDone

	// Save = 中身をまるごと上書き保存する。
	if err := database.DB.Save(&todo).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, errorJSON("更新に失敗しました。"))
	}

	return c.JSON(http.StatusOK, map[string]any{"todo": todo})
}

// deleteTodo = 1件消す。
func deleteTodo(c *echo.Context) error {
	todo, err := findTodo(c)
	if err != nil {
		return c.JSON(http.StatusNotFound, errorJSON("見つかりませんでした。"))
	}

	if err := database.DB.Delete(&todo).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, errorJSON("削除に失敗しました。"))
	}

	return c.JSON(http.StatusOK, map[string]any{"deleted": todo.ID})
}

// findTodo = URLの番号から1件取り出す。見つからなければエラーを返す。
//
// ★Goの関数は値を2つ返せる。「結果」と「エラー」を並べて返すのが定番の形で、
//
//	呼び出し側は if err != nil { ... } で失敗を先に片付ける。
func findTodo(c *echo.Context) (Todo, error) {
	var todo Todo

	// ★URLの :id は文字として届くので、必ず数字に変換してから使う。
	//   変換せずにそのままDBへ渡すと、URLに細工をされてDBを操作される
	//   (SQLインジェクション)危険がある。
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return todo, err
	}

	// Where("id = ?", id) の ? に値を当てはめる書き方なら、
	// 何が入っていても「値」として扱われるので安全。
	err = database.DB.Where("id = ?", id).First(&todo).Error
	return todo, err
}

// errorJSON = 失敗したときの返事の形 {"error": "..."} を作る。
//
// ★この形は main.js との約束。項目名を変えると、画面にエラー文が出なくなる。
func errorJSON(message string) map[string]string {
	return map[string]string{"error": message}
}
