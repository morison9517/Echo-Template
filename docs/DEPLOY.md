# 本番に出す手順

このファイルは**サーバーにアプリを載せるとき**に見る紙です。
日々の開発は [SETUP.md](SETUP.md) を見てください。

> ★**本番前に一度、必ず練習してください。**
> ここに書いてある手順を、公開する当日に初めて打つのは危険です。
> 一度通しておけば、当日は同じことを繰り返すだけになります。

---

## 開発と本番の違い(先に頭に入れておくこと)

| | 開発 | 本番 |
| --- | --- | --- |
| 使うファイル | `compose.yml` | **`compose.prod.yml`** |
| 箱の中身 | Go本体 + air(1GB超) | **実行ファイル1個(30MB程度)** |
| コードの反映 | 保存したら即 | **build し直したとき** |
| CSS・画像を配る人 | Echo | **Nginx** |
| DBのポート | PCから見える(3310) | **開けない** |
| エラー画面 | 詳しく出る | **出ない(内部情報が漏れるため)** |

**「保存しても本番に反映されない」のは正しい動きです。** 本番は実行ファイルに固めたもので動きます。

---

## 1. 準備(初回だけ)

### ① サーバーに Docker を入れる

```bash
docker version
docker compose version
```

### ② コードを置く

```bash
git clone <リポジトリのURL>
cd case_echo
```

### ③ 金庫(`.env`)を作る

```bash
cp .env.example .env
```

**そして必ず中身を書き換えます。** 見本のままだと危険です。

| 項目 | 本番で入れる値 |
| --- | --- |
| `SECRET_KEY` | **長いランダムな文字列**(下のコマンドで作る) |
| `APP_ENV` | `production` |
| `DB_PASSWORD` / `DB_ROOT_PASSWORD` | **開発用と違うパスワード** |

割り印(SECRET_KEY)はこれで作れます。

```bash
python3 -c "import secrets; print(secrets.token_urlsafe(50))"
```

> ★`APP_ENV=development` のまま本番に出すと、起動ログが冗長になり、
> 内部の情報が出やすくなります。必ず `production` にしてください。
>
> ★`APP_ENV=production` で `SECRET_KEY` が見本のままだと、アプリは**起動を拒否します。**
> 見本の値はGitHubで誰でも読めるので、そのまま動かすとログイン状態を偽造されるためです。

---

## 2. 起動する

```bash
docker compose -f compose.prod.yml up -d --build
```

**`-f compose.prod.yml` を毎回付けます。** 忘れると開発用が起動します。

初回は数分かかります。終わったら状態を見ます。

```bash
docker compose -f compose.prod.yml ps
```

`db` `web` `nginx` の3つが `running` なら成功です。

**DBの表を作るコマンドは要りません。** アプリが起動時に自分で用意します。

### 開いて確認

ブラウザで `http://<サーバーのアドレス>/auth/login` を開きます。

> ★`/`(トップページ)は、まだ自分たちで作っていないうちは **404 になります。**
> 開発中は `/` にデモページが出ますが、**本番ではデモを登録しないため**です。
> 「デモが本番に出ない」ことの裏返しなので、故障ではありません。
> `internal/handlers/page.go` の `e.GET("/", index)` のコメントを外せば解消します。
>
> ★同じ理由で、**新規登録とログインは成功しても飛び先が404に見えます。**
> 成功したかどうかは、404の画面ではなくログで判断してください
> (`[アクセス] 302 POST /auth/login` のように 302 が出ていれば成功しています)。

### 最初の利用者を作る

Echo版には管理画面がありません。**普通に新規登録の画面から作ってください。**

```
http://<サーバーのアドレス>/auth/register
```

> DBの中身を直接見たいときは、下の「本番のDBをDBeaverで見たいとき」を参照。

---

## 3. コードを直したとき

```bash
git pull
docker compose -f compose.prod.yml up -d --build
```

**この2行だけです。** DBの表づくりは起動時に自動で走ります。

> ★GORMの `AutoMigrate` は、列を**増やす**ことはできますが、
> 型を変えたり列を消したりはしません。
>
> 本番でどうしても変える必要が出たら、DBeaverで直接ALTERするか、
> 一度データを捨てて作り直すことになります。
> **本番に出す前にモデルを固めておくのが安全です。**

---

## 4. HTTPSにする

練習の段階では `http://` のままで構いません。**本番では必ずHTTPSにします。**

やり方は2つあります。

| | Let's Encrypt(サーバーの中で取る) | AWSのロードバランサー |
| --- | --- | --- |
| 費用 | **無料** | 動かしているだけで月20ドル前後 |
| 更新 | 90日ごと(自動にできる) | AWSが自動でやる |
| 設定 | **このテンプレートに用意済み** | VPC・サブネット2つ・ターゲットグループ |
| 向き | サーバー1台 | 複数台に増やす前提 |

**サーバー1台なら Let's Encrypt です。以下はその手順です。**

> ロードバランサーを使う場合、アプリ側は何も変えません。
> HTTPSはAWS側で終わらせて、サーバーにはHTTPで届きます。
> `prod.conf`(HTTP版)のままで正しく動きます。
> `X-Forwarded-Proto` が「元はHTTPSだった」と Echo に伝えているからです。

### 先に確かめること(★ここを飛ばすと必ず失敗します)

- **ドメインのAレコードが、このサーバーのIPを指していること**

  証明書は「そのドメインを開いたら本当にあなたのサーバーが出るか」を
  確かめてから発行されます。向いていなければ、何度試しても取れません。

  ```bash
  nslookup 自分のドメイン
  ```

- **80番と443番が外から入れること**(AWSならセキュリティグループ)

  80番は「もうHTTPSだから要らない」と思って閉じがちですが、
  **証明書の確認はHTTPで来ます。**閉じると取得も更新もできません。

### ① HTTPのまま起動しておく

```bash
docker compose -f compose.prod.yml up -d
```

`http://自分のドメイン` が開けることを確認してください。
**ここが開けない状態では、証明書は取れません。**

### ② まず練習で取ってみる(★これを先にやる)

```bash
docker compose -f compose.prod.yml run --rm certbot certonly   --webroot -w /var/www/certbot   -d 自分のドメイン   --email 自分のメールアドレス --agree-tos --no-eff-email   --dry-run
```

`--dry-run` は練習用の指定です。**本物の証明書は作られませんが、
本番と全く同じ確認が走ります。**

なぜ練習を挟むかというと、Let's Encrypt には
**「同じ内容は1週間に5回まで」という回数制限**があるからです。
設定が合っているか分からないまま本番で何度も失敗すると、
**その週はもうHTTPSにできません。**`--dry-run` は制限に数えられないので、
ここで何度でも試せます。

`The dry run was successful` と出たら次へ進みます。

### ③ 本物を取る

上のコマンドから `--dry-run` を外して、もう一度実行するだけです。

```
Successfully received certificate.
```

と出れば成功です。証明書は `certbot_conf` の保管庫に入りました。

### ④ HTTPS版の設定に切り替える

**2か所です。**

1. `docker/nginx/prod-https.conf` の `example.com` を**自分のドメインに書き換える**

   ```nginx
   ssl_certificate     /etc/letsencrypt/live/example.com/fullchain.pem;
   ssl_certificate_key /etc/letsencrypt/live/example.com/privkey.pem;
   ```

   ★ここの書き換え忘れが、HTTPS化でいちばん多いつまずきです。
   証明書はあるのに Nginx が起動せず、原因が設定側にあると気づきにくい。

2. `compose.prod.yml` の nginx の**1行を差し替える**

   ```yaml
   # 変更前
   - ./docker/nginx/prod.conf:/etc/nginx/conf.d/default.conf:ro
   # 変更後
   - ./docker/nginx/prod-https.conf:/etc/nginx/conf.d/default.conf:ro
   ```

そして Nginx を作り直します。

```bash
docker compose -f compose.prod.yml up -d nginx
```

`https://自分のドメイン` を開いて、鍵マークが付けば成功です。
`http://` で開いても、自動でHTTPSへ移ります。

> **起動しなかったら、先にログを見てください。**
>
> ```bash
> docker compose -f compose.prod.yml logs nginx
> ```
>
> `cannot load certificate` と出ていれば、1番のドメインの書き換え忘れです。
> **HTTP版(`prod.conf`)に戻せばサイトはすぐ復帰します。**落ち着いて直せます。

### ⑤ 自動更新を仕掛ける

証明書は90日で切れます。**切れるとサイトが警告だらけになり、
多くのブラウザは開くことすら止めます。**手で更新し続けるのは現実的ではないので、
サーバー側の `cron`(決まった時刻に自動で実行する仕組み)に登録します。

```bash
crontab -e
```

開いたら、次の1行を足します(`/path/to/app` は自分の置き場所に変える)。

```
0 3,15 * * * cd /path/to/app && docker compose -f compose.prod.yml run --rm certbot renew --quiet && docker compose -f compose.prod.yml exec -T nginx nginx -s reload
```

- `0 3,15 * * *` = 毎日3時と15時。**1日2回で構いません**
- `renew` は**期限が30日以内に迫ったときだけ**実際に更新します。
  それ以外は何もせずに終わるので、毎日走らせても制限に当たりません
- 更新しただけでは Nginx は古い証明書を持ったままです。
  最後の `nginx -s reload` で読み直させます。**これを忘れると、
  更新は成功しているのにサイトは期限切れのまま**になります

### ⑥ 数日待ってから HSTS を有効にする

`docker/nginx/prod-https.conf` の最後に、コメントアウトされた1行があります。

```nginx
# add_header Strict-Transport-Security "max-age=31536000" always;
```

有効にすると、ブラウザが「このサイトは今後HTTPSでしか開かない」と覚えます。
安全になりますが、**一度覚えさせると1年間取り消せません。**

その間に証明書が切れると、ブラウザはHTTPへ逃げることすら拒むので、
**サイトが完全に開けなくなり、HTTPに戻して直すこともできません。**

自動更新が実際に1回回ったのを確かめてから外してください。急ぐ理由はありません。

### ★HTTPSにしたら必ず戻すこと

`.env` の1行です。

```
SECURE_COOKIES=true
```

練習中に `false` にしていた場合、**戻し忘れるとログイン状態が
HTTPSでない経路でも持ち歩けてしまいます。**
---

## 5. よく使うコマンド

| やりたいこと | コマンド |
| --- | --- |
| 起動 | `docker compose -f compose.prod.yml up -d --build` |
| 停止 | `docker compose -f compose.prod.yml down` |
| 状態を見る | `docker compose -f compose.prod.yml ps` |
| ログを見る | `docker compose -f compose.prod.yml logs -f web` |
| Nginxのログ | `docker compose -f compose.prod.yml logs -f nginx` |
| 箱の中に入る | `docker compose -f compose.prod.yml exec web bash` |

> ★`down -v` は**絶対に打たないでください。**
> `-v` はDBと画像の保管庫ごと消す指定です。利用者のデータが全部消えます。

---

## 6. 本番のDBをDBeaverで見たいとき

本番では**DBのポートを開けていません。** 開けると世界中からログインを試されます。

代わりに、SSHのトンネルを通して見ます。DBeaverの接続設定で
「SSH」タブを開き、サーバーへのSSH情報を入れてください。
そのうえで、ホストは `127.0.0.1`、ポートは `3306` にします。

> トンネル = 自分のPCとサーバーの間に専用の通路を1本引くイメージです。
> 通路の中を通るので、外からは見えません。

---

## 7. 困ったとき

### 画面が真っ白 / デザインが崩れている

Nginxが `web/static/` を見つけられていない可能性があります。
`compose.prod.yml` の nginx に、この行があるか確認してください。

```yaml
- ./web/static:/var/www/static:ro
```

> ★Django版と違い、Echoには「CSSを1か所に集めるコマンド」がありません。
> ソースのフォルダをそのままNginxに見せる作りになっています。
> **サーバー上に `git clone` したソースが必要**なのはこのためです。

### ログインできない(ログイン画面に戻される)

**`SECURE_COOKIES` が原因です。**

`http://` でアクセスしているのに `True` になっていると、
ログイン自体は成功しているのにログイン状態が保存されません。
エラーも出ないので、まずここを疑ってください。

練習中は `.env` に `SECURE_COOKIES=false` を入れてください。

### ボタンを押すと 400(整理券が正しくありません)/ 403

**400** は、フォームに整理券が入っていません。HTMLにこの1行があるか確認してください。

```html
<input type="hidden" name="csrf_token" value="{{ .CSRFToken }}" />
```

JavaScriptから送っている場合は、`main.js` の `api` を使えば自動で付きます。

**403** は、別のサイト(別のドメインやサブドメイン)から送られた、とブラウザが申告した送信です。
Echoが整理券を見る前に断っています。自分たちの別ドメインから送りたい場合は、
`internal/middleware/csrf.go` の設定に `TrustedOrigins` を足して、送り元を名指しで許可します。

### アクセス元のIPが全部同じに見える

`.env` の `TRUSTED_PROXIES` が空です。Nginxを通すと、Echoから見た相手は
Nginxになります。本当のアクセス元はヘッダーに書いてありますが、
**無条件に信じると詐称できてしまう**ので、既定では無視する作りです。

`compose.prod.yml` で `TRUSTED_PROXIES: 172.16.0.0/12` を渡しています。
Dockerの内部ネットワークの範囲です。

### 502 Bad Gateway と出る

Nginxは動いているが、Echo が返事をしていません。
web のログを見てください。だいたい起動時のエラーです。

```bash
docker compose -f compose.prod.yml logs web
```

### **時々**502になる / しばらく待つと勝手に直る

**★DBがメモリ不足で強制終了されています。**アプリのバグではありません。

メモリ1GBのサーバー(AWSの `t3.micro` など無料枠でよく使うもの)で起きます。
MySQL 8 は既定のままだと800MB近く使うので、アプリと合わせて足りなくなり、
**OSが「いちばんメモリを食っている奴」としてMySQLを終了させます。**

たちが悪いのは、この症状の出方です。

- アプリのログにはエラーが出ない(アプリは正常に動いている)
- 少し待つと `restart: always` でDBが戻ってくるので、勝手に直る
- **再現しないので、何が悪いのか分からない**

確かめ方はこれです。

```bash
docker compose -f compose.prod.yml ps
```

db の `STATUS` が `Up 30 seconds` のように**短い時間**になっていたら、
さっき落ちて起き直した直後です。`docker compose -f compose.prod.yml logs db`
に `Out of memory` や `killed` が出ていれば確定です。

**このテンプレートの `compose.prod.yml` には、対策の4行を最初から入れてあります**
(db の `command:` の部分)。それでも起きる場合は、サーバーのメモリを
2GBに上げるか、スワップ(メモリが足りないときにディスクを借りる仕組み)を
用意してください。

```bash
# スワップを2GB用意する(サーバー側で1回だけ)
sudo fallocate -l 2G /swapfile
sudo chmod 600 /swapfile
sudo mkswap /swapfile
sudo swapon /swapfile
echo '/swapfile none swap sw 0 0' | sudo tee -a /etc/fstab
```

### 鍵マークが付かない / 「保護されていない通信」と出る

まず、どちらの状態か切り分けます。

```bash
docker compose -f compose.prod.yml logs nginx
```

| ログの様子 | 原因 |
| --- | --- |
| `cannot load certificate` | `prod-https.conf` のドメインの書き換え忘れ |
| 何も出ず、80番で普通に動いている | `compose.prod.yml` のマウント行が `prod.conf` のまま |
| `certificate has expired` | 自動更新が回っていない(`nginx -s reload` 忘れが多い) |

**証明書そのものの期限は、これで見られます。**

```bash
docker compose -f compose.prod.yml run --rm --entrypoint openssl certbot   x509 -enddate -noout -in /etc/letsencrypt/live/自分のドメイン/fullchain.pem
```

どの場合も、`compose.prod.yml` のマウント行を `prod.conf` に戻して
`up -d nginx` すれば、**HTTPのサイトとしてはすぐ復帰します。**
慌てて設定をいじる前に、先にサイトを生かしてから直してください。

### プロフィールアイコンが表示されない

`media` の保管庫がNginxから見えていない可能性があります。
`compose.prod.yml` の nginx に `media_files:/var/www/media:ro` があるか確認してください。

**なお、開発用と本番用で画像の保管庫は別です。** 開発中に入れた画像は本番にはありません。

### 起動しない / 起動してすぐ落ちる

`volumes:` に `- .:/app` を書いていないか確認してください。
**本番でこれを書くと、固めた実行ファイルが隠れて起動しません。**
本番でいちばん多い事故です。

`SECRET_KEY` を見本のままにしている場合も、アプリは起動を拒否します。
ログに `[起動失敗]` で始まる行が出ているので確認してください。

---

## 8. 本番に出す前のチェックリスト

デプロイの練習のときに、この順で確認してください。

- [ ] `.env` の `APP_ENV` が `production`
- [ ] `.env` の `SECRET_KEY` を見本から変えた(★変えないと起動しません)
- [ ] `.env` の `DB_PASSWORD` を開発用から変えた
- [ ] `/auth/login` が開ける
- [ ] **CSSが当たっている**(見た目が崩れていない)
- [ ] **新規登録 → ログイン → ログアウトが通る**
- [ ] **`https://` で開けて、鍵マークが付いている**
- [ ] **`http://` で開いたら、自動でHTTPSに移る**
- [ ] **証明書の自動更新を `crontab` に登録した**(★これが無いと90日後に止まります)
- [ ] `docker compose -f compose.prod.yml restart` して、データが残っている

**最後の1つが特に大事です。** 再起動でデータが消えるなら、保管庫の設定が間違っています。
