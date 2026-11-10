// =============================================================================
// static.go = CSS・JS・画像のURLに「版の印」を付ける仕掛け
//
//	{{ static "css/style.css" }}  →  /static/css/style.css?v=3f9a2c1b
//
// ▼ 何のためのものか
//
//	ブラウザは一度読んだCSSやJSを手元に取っておき、次からはサーバーに聞かずに
//	それを使う(本番では30日間)。速くなる代わりに、直して出し直しても
//	利用者の画面は古いままになる。
//	そこでURLの後ろにファイルの中身から作った印を付ける。中身が変われば印が変わり、
//	ブラウザは「別のファイルだ」と判断して取り直す。変わらなければ手元の分を使い続ける。
//
// ▼ 使い方(HTMLの中で)
//
//	<link rel="stylesheet" href="{{ static "css/mypage.css" }}" />
//	<script src="{{ static "js/mypage.js" }}"></script>
//	<img src="{{ static "images/logo.png" }}" />
//
//	★"/static/..." と直接書いても表示はされるが、印が付かないので
//	  直したのに古いまま、が起きる。
//	★CSSの中の url(...) で読む画像には印が付かない。差し替えるときはファイル名を変える。
//
// ▼ 消したいとき
//
//	このファイルを消し、base.html の {{ static "..." }} を "/static/..." に戻す。
//
// =============================================================================
package view

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
)

// 静的ファイルの置き場所。router.go の staticDir と同じ場所を指すこと。
const staticDir = "web/static"

// 一度作った印を覚えておく棚。sync.Map = 同時に使われても壊れない辞書。
var staticVersions sync.Map

// init = このファイルが読み込まれたとき自動で1回だけ動く関数。
// HTMLの中で {{ static "..." }} と書けるように、関数を登録している。
func init() {
	funcMap["static"] = staticURL
}

// staticURL = 「css/style.css」を「/static/css/style.css?v=印」に直す。
func staticURL(name string) string {
	name = strings.TrimPrefix(name, "/")
	url := "/static/" + name

	if version := staticVersion(name); version != "" {
		return url + "?v=" + version
	}

	// ★ファイルが無いときは印を付けずに返す(エラーにしない)。
	//   そのファイルだけ404になり、ブラウザの開発者ツールで気づける。
	return url
}

// staticVersion = ファイルの中身から8文字の印を作る。
func staticVersion(name string) string {
	// ★本番では1回作ったら使い回す(ファイルは出し直すまで変わらないため)。
	//   開発中は毎回作り直す。そうしないと、保存しても古い印のままになる。
	cache := appConfig != nil && appConfig.IsProduction()

	if cache {
		if v, ok := staticVersions.Load(name); ok {
			return v.(string)
		}
	}

	// path.Clean で "../" を片付けてから繋ぐ(置き場所の外を読ませないため)。
	file, err := os.Open(filepath.Join(staticDir, filepath.FromSlash(path.Clean("/"+name))))
	if err != nil {
		return ""
	}
	defer file.Close()

	// sha256 = 中身が1文字でも違えば、まったく別の値になる計算。
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return ""
	}
	version := hex.EncodeToString(hash.Sum(nil))[:8]

	if cache {
		staticVersions.Store(name, version)
	}
	return version
}
