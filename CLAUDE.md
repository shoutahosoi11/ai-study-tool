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

