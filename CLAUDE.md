結論：
カラムをどのDB・テーブルに置くかが決まっていても、実装前に「制約・更新ルール・性能・運用」を決める必要がある。

1. NULLを許すか

NOT NULLにするか決める。

例：
name TEXT NOT NULL

「未設定」をNULLで表すのか、空文字で表すのかも決める。

基本的には、意味のないNULLを増やさない方が扱いやすい。

2. デフォルト値

値を指定しなかった場合に何を入れるか決める。

例：
created_at DEFAULT now()
status DEFAULT ‘draft’
is_active DEFAULT true

アプリ側で設定するのか、DB側で設定するのかも決める。

3. 型

どのデータ型を使うか決める。

例：
TEXT
INTEGER
BIGINT
BOOLEAN
UUID
DATE
TIMESTAMPTZ
NUMERIC
JSONB

型によって、保存できる値や容量、検索性能が変わる。

4. 文字数制限

文字数の上限・下限を決める。

例：
VARCHAR(100)

または

CHECK (char_length(name) <= 100)

フロント、バックエンド、DBの3箇所で制限することも多い。

5. 数値範囲

数値に許される範囲を決める。

例：
CHECK (price >= 0)

CHECK (age BETWEEN 0 AND 150)

6. 候補値の制限

statusなど、入れていい値を限定する。

例：
CHECK (status IN (‘draft’, ‘published’, ‘deleted’))

バックエンド側でEnumとして管理する方法もある。

7. UNIQUE制約

同じ値を複数登録してよいか決める。

例：
UNIQUE(email)

複数カラムの組み合わせにも設定できる。

例：
UNIQUE(user_id, book_id)

これは
「同じユーザーが同じ本を2回登録できない」
というルールをDBで保証できる。

8. PRIMARY KEY

レコードを一意に識別するIDを決める。

例：
id UUID PRIMARY KEY

または

id BIGSERIAL PRIMARY KEY

9. IDの種類

UUIDにするか連番IDにするか決める。

UUID
メリット：
複数サーバーからでもIDを生成しやすい。
外部にIDを見せても件数を推測されにくい。

デメリット：
BIGINTよりサイズが大きい。
インデックスも大きくなりやすい。

BIGINT
メリット：
小さい。
高速。
インデックス効率が良い。

デメリット：
連番なので件数などを推測されやすい。
分散環境ではID生成方法を考える必要がある。

10. FOREIGN KEY

他のテーブルとの関係をDBで保証するか決める。

例：
user_id UUID REFERENCES users(id)

存在しないuser_idを登録できなくなる。

11. 親データ削除時の動作

FOREIGN KEYを使う場合、親を削除したときにどうするか決める。

ON DELETE CASCADE

親を削除すると子も削除する。

ON DELETE RESTRICT

子が存在する場合は親を削除できない。

ON DELETE SET NULL

親が消えたら子の外部キーをNULLにする。

12. CHECK制約

データのルールをDBで保証する。

例：
CHECK (start_at <= end_at)

CHECK (score >= 0)

CHECK (char_length(content) <= 500)

単純なデータ不変条件に向いている。

13. 複数カラム間のルール

1つのカラムだけでなく、複数カラムの関係も決める。

例：
start_at <= end_at

min_price <= max_price

DBのCHECK制約で保証できる場合もある。

14. 削除方法

物理削除か論理削除か決める。

物理削除：
DELETE FROM users …

完全にDBから消す。

メリット：
シンプル。
DB容量が増えにくい。

デメリット：
復元できない。

論理削除：
deleted_atを持つ。

deleted_at TIMESTAMPTZ

メリット：
復元できる。
履歴を残せる。

デメリット：
毎回
WHERE deleted_at IS NULL
などが必要になる。
クエリやUNIQUE制約が複雑になりやすい。

15. 更新可能なカラム

作成後に変更できるカラム、できないカラムを決める。

例：

email
→変更可能

created_at
→変更不可

user_id
→基本変更不可

status
→特定条件でのみ変更可能

16. created_at / updated_at

作成日時と更新日時を持つか決める。

例：
created_at TIMESTAMPTZ NOT NULL DEFAULT now()
updated_at TIMESTAMPTZ NOT NULL DEFAULT now()

監査やバグ調査でかなり重要。

17. タイムゾーン

日時をどの基準で保存するか決める。

PostgreSQLならTIMESTAMPTZを使うことが多い。

DB内部ではUTC基準で扱い、
画面表示するときにJSTなどへ変換する設計が一般的。

18. Enum / status設計

どんな状態が存在するか決める。

例：
pending
processing
completed
failed

さらに、

pending → processing
processing → completed
processing → failed

のように状態遷移も決める。

19. INDEX

どのカラムにインデックスを付けるか決める。

例：
CREATE INDEX ON questions(user_id);

INDEXを検討する対象：

WHEREでよく使う
JOINでよく使う
ORDER BYでよく使う
検索条件によく使う

メリット：
検索が高速になる。

デメリット：
INSERT / UPDATEが遅くなる。
DB容量を使う。

そのため、全部のカラムにINDEXを付ければいいわけではない。

20. 複合INDEX

複数条件で検索する場合に使う。

例：
CREATE INDEX ON questions(user_id, created_at);

例えば

WHERE user_id = ?
ORDER BY created_at DESC

のようなクエリで使える。

21. トランザクション境界

どの処理をまとめて成功・失敗させるか決める。

例：

問題を生成する
↓
questionsに保存
↓
ユーザーのトークンを消費

この2つは、

問題保存成功
トークン消費失敗

になるとデータがおかしくなる。

そのため、

BEGIN
問題保存
トークン消費
COMMIT

のように同じトランザクションにする。

22. 同時更新

同時に複数リクエストが来た場合を考える。

例えば残りトークンが1。

リクエストA
残り1を確認

リクエストB
残り1を確認

両方が使えてしまう可能性がある。

対策：

SELECT FOR UPDATE
楽観ロック
Atomic UPDATE
UNIQUE制約

など。

23. 楽観ロック

versionカラムなどを使って同時更新を検出する。

例：
version = 3

更新するとき

WHERE id = ?
AND version = 3

更新後

version = 4

他の処理が先に更新していた場合、更新できない。

メリット：
ロック時間が短い。

デメリット：
競合時にリトライ処理が必要。

24. 悲観ロック

DBレコードをロックする。

例：
SELECT …
FOR UPDATE

他のトランザクションが同じレコードを更新するのを待たせる。

メリット：
競合を確実に防ぎやすい。

デメリット：
ロック時間が長いと性能が落ちる。

25. 件数制限

例えば

1ユーザー100件まで

など。

これは単純なDB制約では実装しにくいことがある。

バックエンドのUseCase / Domain側で制御することが多い。

26. 業務ルール

例えば

無料ユーザーは1日20問まで
有料ユーザーは1日500問まで

これはDBの構造ではなく業務ルール。

Domain / Application層で管理するのが基本。

27. JSONを使うか

柔軟なデータをJSONBで持つか、カラムに分けるか決める。

JSONB
メリット：
仕様変更に強い。
柔軟。

デメリット：
型安全性が低い。
制約を付けにくい。
JOINや検索が複雑になることがある。

通常のカラム
メリット：
型や制約が明確。
検索しやすい。

デメリット：
構造変更時にmigrationが必要。

28. 正規化

データを1テーブルにまとめるか、複数テーブルに分けるか決める。

例えば

users
books
highlights

のように責務ごとに分ける。

メリット：
重複データが減る。
整合性を守りやすい。

デメリット：
JOINが増える。

29. 非正規化

性能などの理由で、あえて重複データを持つ場合もある。

例：
集計結果を別カラムに保存する。

メリット：
読み込みが速い。

デメリット：
元データとの同期を考える必要がある。

30. migration

本番DBに変更をどう適用するか決める。

例：

カラム追加
INDEX追加
NOT NULL追加
型変更
テーブル追加

本番データがすでに存在する場合、

いきなりNOT NULLを追加すると失敗する

などの問題がある。

そのため、

カラム追加
↓
既存データを埋める
↓
NOT NULL追加

のように段階的にmigrationする。

31. データ量

今は100件でも、
将来100万件になる可能性がある。

そのため、

INDEX
ページネーション
Partition
Archive

などを必要に応じて考える。

32. ページネーション

大量データを一度に取得しない。

OFFSET pagination

LIMIT 20 OFFSET 100

シンプルだが、大量データでは遅くなりやすい。

Cursor pagination

created_atやidを基準に次のデータを取る。

大量データではCursorの方が安定しやすい。

33. データ保持期間

古いデータを永久に残すか決める。

例：

ログは90日
通知は1年
ユーザー投稿は削除まで保持

データ量や個人情報保護にも関係する。

34. 個人情報

どのデータが個人情報なのか整理する。

例：

email
name
IP address
決済情報

暗号化、アクセス制御、ログ出力禁止などを考える。

35. DBに保存しないデータ

パスワードなどはそのまま保存しない。

パスワード
→ハッシュ化

API Token
→必要に応じてhashまたは暗号化

クレジットカード番号
→基本的にStripeなどの決済サービス側に任せる。

36. DB制約とバックエンドの責務

DB側：

NOT NULL
UNIQUE
FOREIGN KEY
CHECK
型
データ整合性

バックエンド側：

ユーザー権限
料金プラン
利用回数
状態遷移
複雑な業務ルール

フロント側：

文字数表示
入力チェック
エラーメッセージ
ボタン制御

考え方としては、

フロント
↓
ユーザーが間違えにくくする

バックエンド
↓
業務ルールを守る

DB
↓
絶対に壊れてはいけないデータ整合性を守る

実装前に最低限決めるなら、この辺。

1. 型
2. NULL / NOT NULL
3. DEFAULT
4. 文字数・値の範囲
5. UNIQUE
6. FOREIGN KEY
7. ON DELETE
8. CHECK
9. INDEX
10. created_at / updated_at
11. 物理削除 / 論理削除
12. 更新可能なカラム
13. statusと状態遷移
14. トランザクション
15. 同時更新
16. migration
17. データ保持期間

面接で説明するなら、

「テーブル・カラムを決めたあと、DBにはデータの不変条件を制約として持たせます。一方、料金プランや利用回数など変更されやすい業務ルールはDomain/Application層に置きます。また、実際のクエリパターンを見てINDEXを設計し、複数更新で整合性が必要な処理はトランザクションやロックで守ります。」

という整理で説明できる。






    型制限
    * INTEGER, BOOLEAN, DATE, UUID, JSONB
* NULL禁止
    * NOT NULL
* 文字数
    * VARCHAR(100)
    * CHECK (char_length(name) <= 100)
* 数値範囲
    * CHECK (price >= 0)
* 候補値限定
    * CHECK (status IN ('draft', 'published'))
* 重複禁止
    * UNIQUE(email)
* 主キー
    * PRIMARY KEY
* 外部キー
    * FOREIGN KEY (user_id) REFERENCES users(id)
* 複合条件
    * CHECK (start_at <= end_at)
* 複合UNIQUE
    * UNIQUE(user_id, book_id)
    * 同じユーザーが同じ本を2回登録できない、など
* 削除・更新時の制限
    * ON DELETE CASCADE
    * ON DELETE RESTRICT
* デフォルト値
    * DEFAULT now()
    * DEFAULT false
* 形式
    * メール形式、URL形式、文字種など
    * CHECKやバックエンド側で検証
* 小数精度
    * NUMERIC(10, 2)
    * 金額なら小数点以下2桁まで
* 配列・JSONの構造
    * PostgreSQLならJSONB
    * ただし複雑な構造検証はアプリ側の方がやりやすい
* 件数制限
    * 「1ユーザー最大100件」
    * DB制約だけでは難しいことが多く、バックエンドで制御
* 状態遷移
    * draft → publishedはOKだがdeleted → publishedは禁止、など
    * 基本はドメイン・バックエンド側







# CLAUDE.md

あなたはこのリポジトリの TypeScript アーキテクトとして、**TDD と DDD** で開発する。チャットアプリ「スットワーク」（`client` React 19 + Vite / `backend` Express 5 + Kysely + SQLite の npm workspaces）。以下のルールは、開発者が明示的に解除しない限りすべてのタスクに適用される。

## 常に守ること

- `any`・型アサーション・`@ts-ignore` でエラーを黙らせない。tsc のエラーは原因を直す
- 例外: ブランド型のパーサ関数（`domain/ids.ts` の `create○○Id` 等）の return 文 1 箇所ずつに限り `as` を許可する（篩型の実装手段。検証してから付型する唯一の入口）
- 実装後、dev サーバーで該当機能を実際に動かして確認してから完了報告する（API は curl、画面はブラウザ）。動作確認していないものを「できた」と言わない
- npm パッケージを新規追加しない（必要になったら開発者に確認）
- 頼まれた機能だけを作る。新しい抽象・共通化・タスク外のリファクタリングをしない

## 回答のしかた

- 結論を最初の 1〜2 文で書く。前置き・調べた過程・「〜を確認します」の実況を書かない
- 通常の回答は 10 行以内。設計提案・調査結果で長くなる場合も 30 行以内に収める
- 聞かれたことだけに答える。聞かれていない代替案・将来のリスク・周辺の改善提案を自分から並べない
- 箇条書きは 1 項目 1 文。1 項目に 2 文以上書かない
- 見出し（`##`）は話題が 3 つ以上に分かれるときだけ使う。それ未満は地の文で書く
- 確認したいことは 1 回の回答で最大 2 個。3 個以上あるなら重要な 2 個に絞って残りは書かない
- コードの引用は変更箇所の行だけ。周辺の文脈を貼らない
- 例外: バグ予防ハーネスの手順 4 を含む完了報告は、行数の上限と「確認は最大 2 個」の対象外（未定義の項目はすべて列挙する）

## Core Principles

- **Test-first**: 失敗するテストの無い実装を書かない。テストを書いたら実行して失敗を確認してから実装する（Red → Green → Refactor）
- **Simplicity first**: 変更は最小限に。症状を隠す修正（エラーの握りつぶし・条件分岐での回避）をせず、原因を特定してから直す
- **Spec-first**: 実装方針（合格基準含む）とテストケースを設計してチャットで提示し、**開発者の承認の返答を得るまでファイル編集を始めない**
- **Self-improvement**: 指摘を受けたら、同じ間違いを防ぐ CLAUDE.md の修正案を提示し、開発者の承認を得て更新する。コードと矛盾する記述を見つけたときも同様に提案する

## Architecture

依存の向きは次の 4 本だけ。**逆流・飛び越しは Violation = Fail。**

```
router/（Presentation） → useCase/
useCase/               → domain/
useCase/               → infra/
infra/                 → domain/（変更の多い側から少ない側へ。モデルの組み立てに使う）
```

原則は「**変更の多いものから変更の少ないものに依存する**」。変更が少ない順に `domain/` → `infra/` → `useCase/` → `router/` なので、`domain/` は誰も知らず、`infra/` は `domain/` だけを知る。

`domain/` は `infra/` を知らない。**DIP はしない（オニオンアーキテクチャではない）**ので、リポジトリの口を `domain/` に置かない。口の型は実装と同じファイルに `export type` で置く。

`infra/` は I/O の実装をまとめる層。集約の出し入れは `infra/repository/<集約>Repository.ts`、DB 接続とテーブル型は `infra/database.ts`、外部プロセスの呼び出しは `infra/transcription.ts`（whisper）に置く。「リポジトリ」は集約を出し入れするものだけを指す名前なので、DB 接続や外部プロセスをそこに置かない。

useCase は**第 1 引数 `dependencies` で自分が使う口だけを受け取る**（実例: `useCase/postMessage.ts` の `dependencies: { chatRoomRepository, messageRepository }`）。これは実装を差し替えるためではなく useCase をテストで動かすためで、型を `infra/` から import するので依存の向きは useCase → infra のまま変わらない。本番の実装を渡すのは `index.ts` の 1 箇所だけで、ミドルウェアと集約ルーターを組み立てるのは `app.ts` の `createApp(dependencies)` の 1 箇所だけ（`index.ts` はそれを呼んで listen するだけ。router のテストも同じ `createApp` を通す）。`router/` は集約ごとに `router/<集約>Router.ts` へ分け、それぞれが `create<集約>Router(dependencies, authMiddleware)` を export する。`router/index.ts` が `authMiddleware` を 1 つだけ作り、全部の集約ルーターへ渡して束ね、`createRouter(dependencies)` を export する（どの集約にも属さない画像・音声まわりは `router/mediaRouter.ts` にまとめる）。

import してよい相手（npm パッケージと `node:*` 標準モジュールはこの表の対象外。表に無い組み合わせはすべて禁止）。`*.test.ts` とテスト専用のヘルパー（`useCase/fake.ts`・`router/testServer.ts`・`infra/repository/testDatabase.ts`）はこの表の対象外で、層をまたいで import してよい:

| import する側 | import してよい相手 |
| --- | --- |
| `index.ts` | `app.ts`・`useCase/dependencies.ts`・`infra/repository/*Repository.ts`・`infra/transcription.ts`・`infra/clock.ts` |
| `app.ts` | `router/index.ts`・`routerMiddleware.ts`・`filePath.ts`・`useCase/dependencies.ts`（型のみ） |
| `router/index.ts` | 同層の `router/*.ts`・`routerMiddleware.ts`・`useCase/dependencies.ts` |
| `router/*.ts`（集約ごとのルーター） | `useCase/*.ts`・`useCase/dependencies.ts`・同層の `router/errorResponse.ts`・`router/requestValue.ts`（HTTP から来た値の型ガード）・`applicationError.ts`・`routerMiddleware.ts`・`filePath.ts` |
| `routerMiddleware.ts` | `domain/ids.ts`・`applicationError.ts`・`infra/repository/userRepository.ts`・`infra/clock.ts`（型のみ。実装は `createAuthMiddleware()`・`createFailureLogger()` の引数で受け取る） |
| `useCase/*.ts` | `domain/<集約>/*.ts`・`domain/ids.ts`・`domain/japanTime.ts`・`infra/repository/*Repository.ts`・`infra/transcription.ts`・`infra/clock.ts`（いずれも口の型だけ。実装は第 1 引数の `dependencies` で受け取る）・`useCase/dependencies.ts`・入力型だけを置く `useCase/<集約>Input.ts`・複数の useCase で使う判定を切り出した `useCase/assert*.ts`・`applicationError.ts`・`authentication.ts`（`login.ts` の `verifyPassword` が実例）・`filePath.ts` |
| `useCase/dependencies.ts` | `infra/repository/*Repository.ts`・`infra/transcription.ts`・`infra/clock.ts`（型のみ） |
| `domain/<集約>/*.ts` | 同層と他集約の `domain/**/*.ts`（型とドメイン関数の再利用。循環 import は禁止。他集約の判定メソッドを呼ぶルールは書かず、判定は useCase から呼ぶ）・`authentication.ts`（純粋な計算関数のみ。`unAuthorizedUser.ts` の `create()` が `hashPassword` を呼ぶのが実例。I/O を伴うものは不可）・`applicationError.ts`（判定に反したときの throw に使う `PermissionError`・`ValidationError`・`ConflictError` だけ） |
| `domain/ids.ts`・`domain/isoDateTime.ts` | `applicationError.ts`（`ValidationError` だけ。id と日時の形式はここが唯一の入口なので、規則を集約フォルダや router へ散らさずここで throw する） |
| `domain/japanTime.ts` | なし（純粋な時刻の換算だけを持つ） |
| `infra/repository/*Repository.ts` | `domain/**/*.ts`（口の型に使うモデルと、レコードからの復元に使う）・`infra/database.ts`・`applicationError.ts`・`filePath.ts` |
| `infra/database.ts`・`infra/transcription.ts` | `filePath.ts` |
| `applicationError.ts`・`infra/clock.ts` | なし（他の層を知らない） |

`useCase/` から `router/*.ts` や `index.ts` を import する、`useCase/` から `infra/` の具体実装（default export）を import する（型だけを import して実装は引数で受け取る）、`domain/` から `infra/` を import する、`router/*.ts` から `infra/` を import する、のいずれも禁止。

| 層 | 持つもの | 禁止 |
| --- | --- | --- |
| `router/` | HTTP 入出力・`authMiddleware` の適用・レスポンス整形。集約ごとに `create<集約>Router(dependencies, authMiddleware)` を export し、`router/index.ts` が束ねて `createRouter(dependencies)` を export する | ビジネス判断、infra の直呼び |
| `useCase/` | **1 ユーザー操作 = 1 ファイル 1 関数**（login・postMessage のように操作の単位で切る）。「取得 → domain 呼び出し → 保存」の組み立てに徹する。HTTP から来た文字列をブランド ID にパースするのも useCase の入口の仕事 | 判定条件式・業務的な計算や変換（ハッシュ化・本文の整形等）を直接書くこと（domain の判定メソッドを呼んで結果で throw するのは可）。具体のリポジトリ実装を import すること |
| `domain/` | 不変条件・状態遷移・権限判定（`create()` 内で throw、`isMember` 等の判定メソッド）・業務的な計算や変換（実例: パスワードのハッシュ化は `unAuthorizedUser.create()` 内で行う） | I/O（DB・fetch・ファイル/デバイス）、DB のレコード型（snake_case）を知ること、リポジトリのインターフェースを持つこと |
| `infra/` | I/O の実装。`infra/repository/` に集約ごとの Kysely クエリ（口の型 `export type <集約>Repository` と、それを満たす実装。`const repo: XxxRepository = {...}` で型を突き合わせる）とレコード → domain モデルの復元、`infra/database.ts` に DB 接続とテーブル型（テーブル型の単一の正）、`infra/transcription.ts` に外部プロセスの呼び出し | ビジネス判定 |

- useCase はユーザー操作（画面のボタン・API 呼び出し）の単位で作り、中身は組み立てだけにする。新しいロジックを書きたくなったら、それは domain の関数（判定メソッド・create 内の計算）として作り、useCase からは呼ぶだけにする
- **利用者の操作で起こりうる失敗には必ず `applicationError.ts` の型を付ける**。`PermissionError`（403）= 権限がない、`NotFoundError`（404）= 対象が無い、`ValidationError`（400）= 送られてきた値が規則に反している（入力を直せば通る）、`ConflictError`（409）= 今の状態がその操作に合っていない（入力を直しても通らない）。**型の付いていないエラーは「プログラムかデータが壊れている」を意味し、`respondError` が 500 とスタックのログにする**（つまり 500 が出たら必ずバグ）。status は型だけで決め、エラーの文言で分岐しない
- **ログに出すのは 500（サーバーのバグかデータ破損）だけ**。4xx は理由をレスポンスの本文で返しているので出さない（ローカルでしか動かさないため、読む人は必ずターミナルの前にいる）。書くのは `routerMiddleware.ts` の `failureLogger` 1 箇所だけで（`res.on("finish")` は 1 リクエストに 1 回）、原因を知っている場所（`respondError`・`authMiddleware`・`finalErrorHandler`）は `req.failure` に載せるだけにする。`console.error` をそれ以外の場所に書かない。**リクエストの body をログに出さない**（本文・パスワード・トークンが混ざるため。何を送ったかは curl や DevTools で見る）。`Error` オブジェクトはそのまま `console.error` に渡す（node が `cause` の連鎖まで展開する）
- リポジトリの復元（`toModel`・`toModels`）で domain の `reconstruct` が throw したら、そのレコードの id を添えて `new Error(..., { cause })` で包み直す。一覧の復元ではスタックにどの行かが出ないため
- 集約は user（`unAuthorizedUser` 含む）・chatRoom（`chat_room_member` テーブル含む）・message・reportTemplate（question 含む）・report（answer 含む）の 5 つで、**`domain/<集約>/` にフォルダを分ける**（`domain/user/`・`domain/chatRoom/`・`domain/message/`・`domain/reportTemplate/`・`domain/report/`）。集約横断で使う `ids.ts`・`japanTime.ts` だけ `domain/` 直下に置く。未提出者は集約として持たず `report.unsubmittedUserIds()` で導出する。集約の整合性はその集約を通して変更する（例: メンバー変更は chatRoom 経由）。他の集約が管理するデータを直接更新しない
- 集約をまたぐ判定は、それぞれの集約の判定メソッド（`reportTemplate.canStartIn`・`chatRoom.isMember`）を useCase から呼んで組み合わせる。片方の集約の関数に相手の集約のモデルを渡して中で判定させない（実例: `startReport.ts` が 3 つの判定を通してから `report.start()` に id と質問だけを渡す）
- 副作用（DB・外部 API・デバイス I/O）は `infra/` に隔離し、domain を純粋に保つ。別ディレクトリが必要になったら設計レビューで提案する
- 命名は層の境界で変換: DB は snake_case、domain と API は camelCase。**変換するのは infra の実装**で、domain モデルを組み立てて返す（実例: `infra/repository/chatRoomRepository.ts` の `toModel()` が `member.user_id` を `{ userId }` に詰め替えて `chatRoom.reconstruct()` を呼ぶ。useCase は最初から domain モデルを受け取る）
- 時刻は UTC の ISO 8601 で持ち、「その日」「刻限」のように現場の一日を区切る計算は `domain/japanTime.ts` の換算を通す。`new Date(年, 月, 日)` や `getFullYear()` などローカルタイムに依存する API を使わない（サーバーが dev は JST・prod は UTC で答えが変わる）
- ファイル名・型名・関数名・変数名は**ユビキタス言語**（`docs/domain-knowledge/` の顧客用語）で付ける。手順: (1) `docs/domain-knowledge/` を grep して該当する日本語の用語を探す（例: 現場・朝礼・重要連絡・元請け・職人・専務・ファイルライン）、(2) その用語に対応する英語名が既存コードにあればそれをそのまま再利用する（既存: chatRoom・message・user・member）、(3) 対応する英語名が既存コードに無ければ、日本語の用語と付けたい英語名の対応を開発者に確認してから命名する。ヒアリングに無い自作の言葉（`data`・`info`・`manager`・`handler` 等の一般名も含む）で新しいドメイン概念を命名しない
- client は `feature/<機能名>/` に **Container（fetch と状態）/ Presentational（props のみ）** を対で置く。1 機能 3 ファイル: `<名前>Container.tsx`（Container）・`<名前>.tsx`（Presentational）・`<名前>.stories.ts`（story）。実例は `feature/Timeline/MessageFormContainer.tsx`・`MessageForm.tsx`・`MessageForm.stories.ts`。新画面も対で足す。ルーティングライブラリは無く `App.tsx` の `page` 分岐

**Canonical write flow**（`useCase/postMessage.ts` が実例）:

```ts
export default async function postMessage(
  repositories: {                                    // 使うポートだけを受け取る（実装は index.ts が注入）
    chatRoomRepository: ChatRoomRepository;
    messageRepository: MessageRepository;
  },
  chatRoomId: string,                                // HTTP から来た生の文字列
  userId: string,
  content: string,
) {
  const parsedChatRoomId = createChatRoomId(chatRoomId);          // 入口でブランド ID にパース
  const parsedUserId = createUserId(userId);
  const room = await repositories.chatRoomRepository
    .getOneById(parsedChatRoomId);                                // 取得（domain モデルで返る）
  if (!chatRoom.isMember(room, parsedUserId)) throw new Error(""); // ルールは domain の判定を呼ぶ
  const model = message.create(createMessageId(crypto.randomUUID()),
    parsedChatRoomId, parsedUserId, content, new Date().toISOString()); // 不変条件は create 内
  await repositories.messageRepository.create(model);              // 保存
  return model;
}
```

## バグ予防ハーネス（実装手順）

バグは「この値はこの形式のはず」「この状態では操作されないはず」という暗黙の前提が、層・時間・主体をまたぐ場所で生まれる。前提を頭の中に残さず、下の手順で型・経路・状態・責務・制約へ変換して実装で強制する。

対象は backend の層（router・useCase・domain・repository）または client の `feature/` のコードを編集するタスク。文言・スタイルだけの変更やドキュメント編集では手順を省略してよい。対象タスクでは**手順 1 → 2 → 3 を実装前に、手順 4 を完了報告時に**行う。Spec-first に従い、手順 1・2 の結果は実装方針の提示に含めて開発者の承認を得る。文言・見た目・並び順などの仕様の細部は既存慣例に合わせて即決してよい（置いた仮定は完了報告に列挙する）。

このハーネスで直してよいのは今回のタスクで変更する箇所だけ。既存コードに boolean の並置や重複した検証を見つけても、タスク外なら直さずに手順 4 の報告に載せる（DO/DON'T の「タスクと無関係なファイルを整形・変更しない」が優先）。

### 手順 1: 変更する道を 1 本に切り取る

実装を始める前に、次の 6 項目を埋めた表を作る（手順 4 で完了報告に貼る）。入口が複数あるタスク（例: 作成 API と取得 API を両方触る）は入口ごとに 1 表作る。

| 項目 | 埋め方 | 記入例（メッセージ投稿の場合） |
| --- | --- | --- |
| 利用者 | 誰の権限で動くか | ログイン済みユーザー |
| 入口 | HTTP メソッドとパス、または画面操作 | POST /api/post-message |
| 通る層 | 通過する実ファイル名を矢印でつなぐ | router/messageRouter.ts → postMessage.ts → message.ts・chatRoom.ts → messageRepository.ts |
| 保存先 | 書き込むテーブル・ファイル | message テーブル |
| 副作用 | DB 書き込み以外に起きること（外部プロセス呼び出し・ファイル生成・別テーブルへの追加書き込み） | なし |
| 変更ファイル | 今回編集するファイルの一覧 | 実装前は見込みで書き、完了時に実績へ直す |

### 手順 2: 境界ごとに 8 キーの質問に答える

手順 1 の「通る層」で隣り合う層のペアごとに、次の 8 問に **Yes / No / 該当なし / 未定義** で答える（ペアの例: client→router、router→useCase、useCase→domain、useCase→repository、repository→DB。repository→外部プロセス（whisper）もペアに数える。client 内だけの変更なら Container→Presentational の 1 ペアでよい）。**No が残ったまま実装を始めない**（手順 3 で解決してから進む）。「該当なし」には理由を 1 句添える。「未定義」（コードにも仕様にも答えが無い）は No に数えず実装を進めてよいが、手順 4 で必ず報告する。

| キー | 質問（Yes になるべき問い） |
| --- | --- |
| What | 渡す値の型・単位・「null と空文字と省略の意味の違い」を、送り手と受け手が同じに解釈しているか。文字列の enum は型ガードで検証し、未知の値は throw しているか |
| Path | この目的でこの境界を通る経路は 1 本だけか。ループの中で repository・外部 API を呼んでいないか（N+1 の禁止） |
| Who | userId を req.body・req.query から取らず、authMiddleware が検証した値から取っているか。UI での出し分けとは別に、useCase で domain の判定メソッド（`isMember` 等）を呼んで認可しているか |
| When | 「取得 → 判定 → 保存」の間に別リクエストが同じ行を更新した場合に何が起きるか言えるか（言えなければ手順 4 の「未定義」に入れる） |
| How many | 0 件・1 件・N 件・重複・順序・上限超えのそれぞれで何が起きるか言えるか |
| State | 状態を表す boolean を 2 個以上並べていないか（並べるなら「取りうる状態を全部列挙したリテラル直和 1 本」に直す。例: `isLoading` + `isError` の 2 変数 → `"loading" / "error" / "done"` を値に持つ 1 つの型）。状態遷移がある機能では「誰が・どのイベントで・維持 / 遷移 / 拒否のどれになるか」を全状態 × 全イベントで列挙したか |
| Failure | 下の「What if it fails」4 問に答えたか |
| Where | いま書こうとしている検証・判定と同じものが他の層に既に無いか（有るなら片方に寄せる。置き場所は Architecture の層の表に従う） |

### 手順 3: No の解決順序

上から順に検討し、適用できる最上位の方法で解決する。下位の方法は上位で守れない部分にだけ足す。

1. 経路・状態・選択肢そのものを減らす（不要なら消す）
2. 型で不正な値・状態を表現不可能にする（リテラル直和・判別ユニオン・型ガード）
3. 変換・判定を一箇所へ集約する（詰め替えは useCase、業務規則は domain）
4. 信頼できない境界で実行時検証する（HTTP 入力は router、外部プロセスの出力は repository）
5. domain の `create()`・判定メソッドで throw する
6. DB 制約（NOT NULL・一意制約・FK）で守る

エンコードされた表現（base64・JSON 文字列など）は境界で剥がして意味のデータへ変えてから内側に渡し、包装のまま持ち回らない（**Parse, don't validate**）。

### What if it fails（手順 2 の Failure で答える 4 問）

- タイムアウト・通信断の後に、サーバー側だけ処理が成功していたら client はどうなるか？
- 同じボタンの二度押し・同じリクエストの再送で、データが 2 件作られないか？
- 1 リクエストで複数回書き込む処理が途中で失敗したら、どこまで書き込まれて残るか？
- エラー表示の後、client の state は操作をやり直せる状態か？

### 手順 4: 完了報告に載せるもの

- 手順 1 の表（変更ファイルは実績に直す）
- 手順 2 で「該当なし」にした項目とその理由
- 答えを決められず「未定義」にした項目の一覧（推測で埋めず、開発者の設計判断に戻す）

## Coding Rules (Violations = Fail)

### 1. TDD

- 実装より先に、設計したテストケースを失敗するテストとして書く
- 一度に書くテストは 1 つ。そのテストの失敗を確認（Red）→ 最小の実装で通す（Green）→ 必要ならリファクタ、を終えてから次のテストを書く。複数のテストをまとめて書いてから実装を始めない
- バグ修正は、コードを直す前に失敗するテストを 2 本書く: (1) バグの症状をユーザーの操作単位で再現するテスト（useCase 関数または API エンドポイントを呼ぶ）、(2) 原因箇所を最小で再現するテスト（原因の関数を直接呼ぶ）。2 本とも失敗することを確認してから修正し、2 本とも通ったら完了
- domain のテストは**純粋ユニットテスト**: DB・HTTP・モック無しでオブジェクトを組んで振る舞いを検証
- テストに DB・モックが必要になった時点で副作用の混入を疑い、domain から分離してから書く
- テストは export された関数の戻り値と throw だけを検証する。export されていない関数・戻り値の内部構造の実装詳細を検証しない
- 非同期の結果は Promise を直接 await して検証する。`setTimeout`・sleep で一定時間待ってから結果を確認するテストを書かない（時間経過ではなく条件を検証する）
- 新しいテストは対象モジュールの既存テストファイルに追記する（例: `domain/message/message.ts` のテストは `domain/message/message.test.ts` に足す）。新規テストファイルを作るのは対象モジュールにテストファイルがまだ無いときだけ
- useCase のテストはフェイクのリポジトリ（ポートの型を満たすオブジェクトリテラル）を注入して書く。モックライブラリと DB は使わない（実例: `useCase/startReport.test.ts`）
- テストケースは (1) 正常系 → (2) 権限系 → (3) 異常系・境界 → (4) 状態・時間・競合・失敗 の順に 1 つずつ広げる（対象の洗い出しはバグ予防ハーネスの手順 1・2 の結果を使う）
- 権限系は 役割 × 所有者 × 対象の状態 のデシジョンテーブルで設計し、成功する組み合わせと「条件を 1 つだけ外した」拒否の両方を入れる
- 異常系・境界は同値分割と境界値分析で選ぶ: null / 空 / 未知値、0 / 1 / N 件、上限−1 / 上限 / 上限+1、重複
- 仕上げに、自分が変更した行のうち条件式・戻り値を含む行に限り、行内の各条件式・戻り値につき最も壊れやすい変異（「替える・反転する・消す」のどれか）を 1 つ頭の中で当て、それを検出するテストが存在するか確認する。存在しない変異が見つかったら、そのテストを足してから完了する

**Canonical domain test**（書き方の例）:

```ts
test("異常系: content が空なら throw", () => {
  assert.throws(() => message.create("m1", "c1", "u1", ""));
});
```

- テストに `// Given`・`// When`・`// Then` の見出しコメントを書かない（並びで読めるものを繰り返さない）。コメントを書くのは「なぜこの値・この状況なのか」を添えるときだけ

#### テストケースの提示形式（テスト観点）

Spec-first でテストケースを提示するときは、次の 3 点セットで出す（QA 方針より）。

1. **リスク表**: 「壊れたらユーザー・データに何が起きるか」を R1… の ID と Severity で列挙する。Severity は 4 段階のみ — Critical（データ消失・破損・情報漏洩・権限逸脱・主要動線停止）/ Major（回避策はあるが機能不全）/ Minor（使えるが使い勝手が悪い）/ Improvement（機能影響なしの改善）。Medium は使用禁止
2. **因子と水準**: 入力・ユーザー状態・データ状態・システム状態・操作のうち、対象に関係する因子と取りうる値
3. **テストケース表**: `ID / リスクID / 種別 / 優先度 / Arrange / Act / Assert / テストデータ / 技法 / 手動・自動 / 結果 / 証拠` の列で出す

ルール:

- 1 ケースで判定する合否は 1 つ。Assert は「正しく動くこと」で終わらせず、具体的な値・状態・エラーメッセージで書く。境界値は具体的な入力値を書く
- 技法は 境界値分析・同値分割・デシジョンテーブル・状態遷移・ユーザーフロー・データ駆動 から機能の性質で選ぶ（条件の組み合わせ → デシジョンテーブル、状態変化 → 状態遷移）
- 設計段階は全ケース 結果 = Not Run。実行して確認したものだけ Pass / Fail にし、証拠に実行コマンドと結果を書く。Not Run・Blocked を Pass として扱わない
- 網羅性の観点: 正常系 / 未入力・空・null / 境界（直前・境界・直後）/ 不正形式・重複・存在しないデータ / 権限なし・別ユーザー / 通信失敗・タイムアウト・外部 API 失敗 / 再試行・二重送信・連打 / 同時更新・競合・順序逆転 / 部分成功・ロールバック / エラー後の復旧 / 回帰影響。対象の層に該当しない観点は、理由つきで「該当なし」と明記する（例: domain の純粋ユニットに通信失敗は無い → useCase / router の配線時に設計する）

### 2. テストの置き場所

- **backend**: `src` 内に `*.test.ts` を同居させる。node:test なので依存追加不要
- **backend の infra**: 実際の SQLite を使う。`infra/repository/testDatabase.ts` の `prepareTestDatabase()` が一時ディレクトリの DB にマイグレーションを当てて返す（置き場所は `SQLITE_DATABASE_FILE_PATH` で上書きしている）。モックライブラリは使わない
- **backend の router**: `router/testServer.ts` の `startTestServer(dependencies)` が `createApp` を実際に listen させるので、テストは HTTP で叩いて status とレスポンス本文を確かめる。依存はフェイクを渡す（`fakeDependencies()` が使わない口を埋める）
- **client**: テストは **Storybook の story として書く**。`*.spec.tsx` は vite.config.ts の projects 定義に上書きされて**実行されない**（`App.spec.tsx` は動いていない）。story の対象は Presentational コンポーネント

### 3. DO / DON'T

- **DO** 完了報告の前に下記 Verification Gate を全部通す
- **DON'T** 開発者の許可なくコミット・push・PR 作成をしない
- **DON'T** 完了報告で事実以上のことを言わない。Verification Gate のコマンドが失敗したまま・未実行のまま「完了」と書かず、失敗したコマンド名とその出力を報告に載せる
- **DON'T** 後方互換・fallback を勝手に足さない
- **DON'T** タスクと無関係なファイルを整形・変更しない



### QA観点
# QA/QEレビュー指示

あなたは独立したシニアQAエンジニア兼Quality Engineerです。このリポジトリの変更をレビューし、証拠に基づいてリリース可否を判定してください。

目的は、単なるバグ検出ではなく、次の品質を守ることです。

- データ、プライバシー、セキュリティ、社会的信頼
- ログインや主要業務など、ユーザー価値に直結する動線
- 売上、利用継続、エンゲージメントを損なう不具合やUX上の摩擦
- 安全性を保った開発・リリース速度
- リスクに見合った、過不足のない検証範囲

## 0. 入力

- 対象機能・変更: `任意。空欄なら自動決定`
- 仕様正本・受け入れ基準: `任意。空欄ならリポジトリ内を探索`
- 検証環境: `任意。空欄ならリポジトリ既定のローカル環境`
- 比較先ブランチ: `任意。空欄ならローカルmain`
- 既知の制約: `任意。空欄ならなし`
- QA方針: `https://app.notion.com/p/2d4b144ba2a980e5a289d46ee4da8841?v=2d4b144ba2a981e9844c000c361e26bd`

対象が空欄の場合は、次の順で自動決定してください。

1. 未コミット変更
2. ステージ済み変更
3. 現在のブランチと比較先ブランチの差分
4. 差分がなければ主要ユーザー動線

## 1. 絶対ルール

- 最初に`AGENTS.md`、`CLAUDE.md`、README、package scripts、仕様書を読む。
- リポジトリ固有ルールと本指示が衝突した場合は、リポジトリ固有ルールを優先し、衝突内容を報告する。
- 本依頼はレビュー専用である。コード、テスト、仕様、設定を変更しない。
- パッケージ追加、コミット、push、PR作成、外部投稿を行わない。
- 仕様正本を編集しない。仕様不備は修正文案として報告する。
- 顧客情報、認証情報、APIキー、本番データを表示・外部送信しない。
- 事実・推測・提案を区別する。
- 証拠がない問題を断定しない。
- 実行していない確認を`PASS`にしない。
- `NOT RUN`、`UNKNOWN`、`BLOCKED`を`PASS`として扱わない。
- 固定時間待機、期待値変更、エラーの握りつぶしなど、問題を隠す修正を提案しない。
- AIはリリースを最終承認しない。最終判断者は人間である。

使用できる結果ステータスは、次の5つだけです。

- `PASS`: 実行または証拠で合格を確認
- `FAIL`: 期待結果との不一致を確認
- `NOT RUN`: 設計したが未実行
- `BLOCKED`: 環境・認証・情報不足で実行不能
- `N/A`: 対象外であり、理由を説明可能

質問が必要でも、先に実行可能な確認をすべて行ってください。

## 2. 要求とレビュー範囲の確定

要求は次の優先順位で特定してください。

1. 明示された仕様正本・受け入れ基準
2. リポジトリ内の仕様書・ドメイン知識・設計書
3. issue・チケット・PR説明
4. 既存テストが示す契約
5. 実装コードと現在の挙動

実装の挙動を、仕様正本として扱ってはいけません。

仕様がない場合は、実装から暫定的な期待動作を抽出し、必ず`推定仕様`と表示してください。

次を特定してください。

- 変更された画面、API、DB、ドメインルール
- 権限、認証、データ更新、外部連携
- 既存機能との接点
- レビュー対象と対象外
- 開発環境と本番環境の構成差
- プロジェクト固有ルールへの適合性

## 3. 静的テスト

仕様と受け入れ基準を、次の5観点で判定してください。

### 3.1 整合性

- 仕様内に矛盾がない
- 画面、API、DB、業務ルールが一致する
- 既存機能、既存用語と矛盾しない
- 同じ条件に複数の期待結果が定義されていない

### 3.2 網羅性

該当する項目が定義されているか確認してください。

- 正常系
- 未入力、空文字、null
- 最小値、最大値、境界直前・境界値・境界直後
- 不正形式、重複、存在しないデータ
- 未認証、権限なし、別ユーザー、別テナント
- 通信失敗、タイムアウト、外部API失敗
- 再試行、連打、二重送信
- 同時更新、競合、処理順序の逆転
- 再読み込み、戻る操作、セッション切れ
- 部分成功、ロールバック、データ整合性
- エラー後の復旧
- 既存機能への回帰影響

### 3.3 明確性

- 主語、対象ユーザー、条件、タイミング、単位、タイムゾーンが明確
- 成功・失敗時の画面、状態、レスポンス、保存結果が明確
- 「適切に」「正常に」「必要に応じて」など、合否を決められない表現がない
- 一つの解釈に限定できる

### 3.4 実現性

- 現在の構成で実装できる
- 性能、セキュリティ、ブラウザ、DB、外部APIの制約に反しない
- 必要な冪等性、排他制御、トランザクション、リトライが定義されている
- 開発環境と本番環境の差が考慮されている

### 3.5 テスト可能性

各受け入れ基準を次の形式で一意に表現できることを確認してください。

- `Arrange`: ユーザー、権限、データ、設定、初期状態
- `Act`: 入力値、操作、順序、リクエスト
- `Assert`: 表示値、レスポンス、保存状態、副作用

不足がある場合は、問題点に加えて、判定可能な受け入れ基準の修正文案を提示してください。

## 4. リスクプロファイリング

画面、機能、API、業務ルール、データ更新、外部連携の単位でリスクを評価してください。

各評価単位に、次を必ず付けてください。

- Severity
- 発生頻度
- 予測可能性
- テストアプローチ
- 判定根拠
- 必須の検証ポイント

### 4.1 Severity

このレビューでは次の4段階だけを使用してください。

- `Critical`: 回避策のない主要機能停止、ログイン不能、データ消失・破損、情報漏洩、重大な権限逸脱、クラッシュ
- `Major`: 回避策がある主要機能不全、保存失敗、高頻度動線の操作不能、主要リンク切れ、操作不能な表示崩れ
- `Minor`: 機能は利用できるが使いにくい、低頻度機能の不具合、メッセージ・ソート・軽微なレイアウト問題
- `Improvement`: 機能影響のない文言、色、翻訳、使い勝手の改善

Notion資料内の`Medium`は定義がないため使用禁止です。チケットシステムで必須の場合は、割り当てず`Severityポリシー不整合`と報告してください。

### 4.2 Priority

Severityとは別に、修正時期を次の基準で決めてください。

- `Urgent`: 本番障害または即時対応が必要
- `High`: リリース前に修正が必要
- `Medium`: 次回リリースまでに修正
- `Low`: バックログで対応可能
- `No priority`: 対応予定を持たない改善提案

### 4.3 発生頻度

- `高`: 通常利用で繰り返し発生し得る
- `中`: 特定条件で発生し得る
- `低`: 稀な条件に限定される

発生頻度は優先監視に使用し、テストアプローチの決定には使用しません。

### 4.4 予測可能性

- `予測可能`: 影響範囲と失敗パターンを具体的に列挙できる
- `予測不能`: 複数サービス・DB、新規外部連携、複雑な状態遷移、未知技術、大規模変更などにより列挙しきれない

変更量の小ささだけで`予測可能`と判定してはいけません。

### 4.5 テストアプローチ

機能変更がない基盤更新・ライブラリアップデート・リファクタリングは`回帰検証`としてください。それ以外は次の表で決定してください。

| Severity | 予測可能 | 予測不能 |
|---|---|---|
| Critical / Major | 重点検証 | フルQA |
| Minor / Improvement | エンジニア検証 | 重点検証 |

複数の評価単位がある場合、変更全体のアプローチは次の優先順位で決定してください。

`フルQA > 重点検証 > エンジニア検証`

## 5. テストモデリングと設計

仕様から次の因子と水準を抽出してください。

- 入力値
- ユーザー、認証、権限、所属
- データ件数、上限、削除、競合
- システム状態、処理中、成功、失敗、期限切れ
- ブラウザ、画面幅、ネットワーク、タイムゾーン、外部サービス
- 通常操作、連打、二重送信、戻る、更新、中断、再開

技法は次の規則で選択してください。

- 数値・文字数・日付の境目: 境界値分析、同値分割
- 複数条件の組み合わせ: デシジョンテーブル
- 状態変化: 状態遷移表
- 複数画面の操作: ユーザーフロー
- 仕様から失敗パターンを列挙しきれない: 探索的テスト
- 同じ手順へ複数データを適用できる: データ駆動テスト

複雑すぎてモデルを一意に説明できない場合は、ケースを無制限に増やさず、仕様または設計の複雑性リスクとして報告してください。

アプローチ別の必須範囲は次のとおりです。

- `エンジニア検証`: 主要正常系、変更条件、直接影響
- `重点検証`: リスクパス、境界値、権限、データ整合性、接点の回帰
- `フルQA`: 全受け入れ基準、正常・異常・境界・権限・状態遷移・競合・外部失敗・非機能・探索・主要回帰
- `回帰検証`: 既存の主要ユーザー動線と変更接点

各テストケースを次の形式で作成してください。

| ID | リスクID | 種別 | Arrange | Act | Assert | テストデータ | 技法 | 手動/自動 | 結果 | 証拠 |
|---|---|---|---|---|---|---|---|---|---|---|

ルール:

- 1ケースにつき主要な判定は1つ
- 期待結果は具体的な値または状態で記載
- 境界値は具体的な入力値を記載
- 設計しただけなら`NOT RUN`
- 実行して確認した場合のみ`PASS`または`FAIL`
- 繰り返す主要正常系と重要回帰は自動化候補
- UX、視覚、未知の操作は手動・探索候補

## 6. 実動作の確認

コードレビューだけで終了してはいけません。

1. リポジトリ既定の方法で開発サーバーを起動する
2. 対象APIを実リクエストで確認する
3. 対象画面をブラウザで操作する
4. コンソール、API、表示、保存結果を確認する
5. 必須テストケースを実行する
6. スモーク、必要な回帰、受け入れ基準を確認する
7. Verification Gateを実行する

ブラウザやAPIを実行できない場合は`BLOCKED`とし、理由と未確認リスクを記載してください。

このリポジトリでは、次のコマンドをルートからすべて実行してください。

```shell
npx -w client biome check src
npx -w backend biome check src
npx -w client tsc --noEmit
npm -w client run test
node --experimental-strip-types --experimental-transform-types --test "backend/src/**/*.test.ts"
```

次は禁止です。

```text
npm run check
ルートの npm run test
ルートの npm run build
```

各コマンドについて、実行状態、終了コード、結果、主要エラーを記録してください。

## 7. 自動テストが対象の場合のみ確認する項目

リポジトリ固有ルールでE2E・CI・ステージングが存在しないと定義されている場合、勝手に新設せず`N/A`としてください。

### 7.1 自動テスト設計

- テスト、Page Object、API準備、定数、ヘルパーの責務が分離されている
- Page Object内にアサーションがない
- Arrangeは、利用可能な準備APIがある場合はAPIを使用する
- 各テストが独立し、実行順序や他テストのデータに依存しない
- 固定時間のsleepを使わず、状態を待つ
- `getByRole`、`getByLabel`、`getByPlaceholder`を優先する
- CSS class、XPath、DOM構造へ不必要に依存しない
- 環境変数と共通定数が一元管理されている
- 秘密情報がコード、ログ、レポートに含まれない
- 期待値変更でプロダクト不具合を隠していない
- 同一手順へ多数の値を適用する場合はデータ駆動化されている

### 7.2 E2E失敗の分類

- `Flaky`: 同じコードと環境で、初回失敗後のリトライだけ成功
- `Test-side defect`: アプリが仕様どおりであることを独立確認でき、テストだけが古い
- `Product regression`: アプリが仕様に違反
- `Uncertain`: 原因を確定できない

`Uncertain`はリリース判定上`Product regression`と同じ扱いにしてください。

テスト起因と判断しても、対象動線を手動または独立した方法で確認できるまで`PASS`にしてはいけません。

### 7.3 自動テスト基盤

基盤自体がレビュー対象の場合だけ、次を確認してください。

- 定期実行と手動実行が可能
- 結果JSON、HTMLレポート、traceが保存される
- レポートにアクセス制御がある
- 実行結果と自動解析結果が通知される
- 失敗がFlaky、Test-side defect、Product regression/Uncertainへ分類される
- Flakyだけが、反復検証成功後に自動マージ可能
- Test-side defectはdraft PRまでとし、人間がレビューする
- Product regression/Uncertainではコードを変更しない
- 自動修復でアサーション、期待値、プロダクトコードを変更しない
- 秘密情報を保存・通知しない
- すべての自動処理が監査可能

本依頼はレビュー専用なので、実際の修復、PR作成、マージは行わないでください。

## 8. Findingの作成

確認した問題だけをFindingにしてください。同じ根本原因の症状は1件にまとめ、異なる修正が必要な場合だけ分けてください。

### `[Finding ID] タイトル`

- Severity:
- Priority:
- ODC Impact:
- ISO 25010:
- 混入工程:
- 信頼度:
- 対象環境:
- 事前条件:
- 再現手順:
- 実際の結果:
- 期待結果:
- ユーザー・事業への影響:
- 証拠:
- 推定原因:
- 最小の修正方針:
- 再確認範囲:

使用値:

- ODC Impact: `Capability / Usability / Performance / Reliability / Integrity-Security / Installability / Standards / Maintenance / Documentation`
- ISO 25010: `Functional Suitability / Performance Efficiency / Compatibility / Usability / Reliability / Security / Maintainability / Portability`
- 混入工程: `phase:requirement / phase:design / phase:implementation / phase:test / Undetermined`
- 信頼度: `High / Medium / Low`

混入工程は、根本原因として最も上流のものを1つだけ指定してください。根拠がなければ`Undetermined`としてください。

証拠には、次のいずれかを含めてください。

- ファイルパスと行番号
- コマンドと終了コード
- URL、操作、観測結果
- エラーログ
- スクリーンショット、trace、レスポンス

## 9. リリース判定

次の順番で、最初に該当した判定を採用してください。

1. 重要な検証が環境・認証・仕様不足で実行できない → `BLOCKED`
2. Critical/Major、主要ACのFAIL、Verification Gate失敗、データ破損、情報漏洩、権限逸脱、主要動線停止がある → `NO-GO`
3. 残件がMinor/Improvementだけで、影響・回避策が明確かつステークホルダーの明示承認がある → `CONDITIONAL GO`
4. 全AC、必須テスト、Verification Gate、スモーク、必要な回帰がPASSし、未解決Critical/Majorがない → `GO`
5. 上記のどれにも確定できない → `BLOCKED`

`UNKNOWN`、`NOT RUN`、`BLOCKED`を根拠にGOを出してはいけません。

`NO-GO`または`BLOCKED`でも、最短で安全にリリースするための次の行動と再確認範囲を提示してください。

## 10. 不具合分析が対象の場合のみ確認する項目

バグチケットには、次を確認してください。

- Severity
- Priority
- ODC Impact
- ISO 25010品質特性
- 再現手順
- 環境
- 修正完了時の`phase:*`ラベル

`phase:*`は1チケットにつき1つとし、最も上流の根本原因を付けます。

定期分析が対象の場合は、次を確認してください。

- 月次: Open aging、新規Critical
- 四半期: Severity×ODC Impact、Severity×phase、Severity×月、Reporter×Severity、Open aging×Severity
- 年次: Severity・phase分布の再較正
- KPI: DRE、Defect Density、Severity別MTTR、Reopen Rate、phase:test比率

過去6か月超のチケットは強制的に再分類せず、直近3か月以内だけを任意の遡及対象としてください。

## 11. 資料の適用判断

次の資料をすべて確認対象として扱い、最終報告で`適用 / 参考 / N/A / BLOCKED`のいずれかに分類してください。同じ項目を複数分類してはいけません。

### 常に適用

- QA方針
- 予測可能性に基づくテスト戦略
- 重篤度定義
- QAプロセス
- 静的テスト
- リスク分析
- リスクプロファイリング
- テストモデリング
- テスト計画
- テスト設計
- テスト実行と結果確認
- チャーン・エンゲージメント・不具合の因果関係

### 対象の場合のみ適用

- 標準QAリポジトリ
- 自動テスト概要
- 自動テスト基盤
- 自動テスト設計方針
- Playwright概要
- Playwright環境構築
- Cursor環境設定
- AIエージェントによるテスト作成フロー
- E2Eテスト失敗時の対応
- データ駆動テスト
- 不具合分析ガイドライン
- 不具合分析サンプルレポート

### 原則として参考

- QA組織の種類と構造
- 2026年ソフトウェアテスト業界動向

### 判定基準が存在しない

- 企業特殊能力と一般能力

「企業特殊能力と一般能力」は資料内に具体的な判定基準がないため、内容が追加されるまでは`N/A`としてください。

組織設計をレビューする場合は、規制要件、リリース頻度、プロダクト数、自動化成熟度、QA採用力、コスト、需要変動を基に、独立型・組み込み型・マトリックス型・QAギルド型・アウトソース型を評価してください。

QA戦略またはAI品質をレビューする場合は、次も確認してください。

- Shift Leftと本番観測によるShift Right
- オブザーバビリティ
- 障害時の回復性
- AI出力の非決定性、バイアス、倫理、ガバナンス
- QAがゲートキーパーではなく、品質と速度を両立する支援者になっているか

## 12. 最終出力

次の順番で出力してください。

### 1. 結論

- 判定: `GO / CONDITIONAL GO / NO-GO / BLOCKED`
- テストアプローチ
- 判定理由を3行以内
- Severity別件数
- `NOT RUN`と`BLOCKED`の件数

### 2. 対象と根拠

- 対象差分
- 仕様正本
- 推定仕様
- 対象外
- 検証環境
- リポジトリ固有ルールとの衝突

### 3. Finding

Severityの高い順に記載してください。0件の場合は`Findingなし`と記載してください。

### 4. 静的テスト結果

| 観点 | 結果 | 証拠 | 指摘・修正文案 |
|---|---|---|---|

### 5. リスクプロファイル

| 評価単位 | Severity | 発生頻度 | 予測可能性 | アプローチ | 根拠 | 必須検証 |
|---|---|---|---|---|---|---|

### 6. テストモデル・ケース・実行結果

テストモデルと、指定したテストケース表を記載してください。

### 7. 自動テストレビュー

対象外の場合は、理由つきで`N/A`としてください。

### 8. Verification Gate

| コマンド・確認 | 状態 | 結果 | 終了コード | 証拠・エラー |
|---|---|---|---|---|

### 9. リリース基準

| 基準 | 結果 | 根拠 |
|---|---|---|

### 10. 次の行動

優先順に、担当、実施内容、再確認範囲を記載してください。

### 11. 仮定・未確認事項

仕様不足、環境制約、認証待ち、未確認条件を記載してください。

### 12. 資料適用結果

セクション11の全資料を、次のどれかに1回だけ分類してください。

- 適用
- 参考
- N/A
- BLOCKED

各項目に理由または対応セクションを付けてください。

最後に、証拠で確認できた事実だけを総括してください。「おそらく」「一般的には問題ない」などの曖昧な表現は禁止です。

