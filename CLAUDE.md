仕様書などからテストすべき項目を洗い出し、システムの内部構造を考慮せずに実施するテスト技法
代表的なものに、以下のような技法があります
同値分割法(equivalence partitioning)
境界値分析(boundary value analysis)
デシジョンテーブル(decision table testing)
状態遷移テスト(state transition testing)
ドメイン分析(domain analysis)
ユースケーステスト(use case testing)
組み合わせテスト(combinatorial testing)
クラシフィケーションツリー法(classification tree method)
原因結果グラフ法(cause-effect graphing)



https://jira.atlassian.freee.co.jp/browse/APAR-4206Can't find link  あなたは図書館の管理人です。本の管理システムを作りたいと思っています。 以下の機能を実現させ、本の管理システムを作成してください。
* 本（Book）の情報登録
    * カテゴリー
    * タイトル
    * 作者
    * 発売日
    * 金額
* 本の情報更新
* 本の情報検索
* 本の貸し出し記録。誰が、いつ借りて、いつ返したか
[memo]
開発を進める前に何をやるべきなのかをまず相談をする
* 誰が使うのかなど
    * 管理者
* 機能を作るのにはどんなデータが必要？
    * カテゴリー
    * 哲学・芸術・文学 

カテゴリ

* 実装する前に要求を詰めて不明点をなくす
    * 実装する機能
        * 本の情報登録・更新・検索・貸し出し記録
    * 同じ本が複数冊ある場合
        * 一意のIDで識別する
            * 管理用の番号を振る方が良いのでは？
            * createdatはDBの持ち物
            * DBを移行した時に値が変わってしまう
            * 本の管理番号をつけて一意になる番号を作る
    * 利用者の識別方法
        * 借りる人をどう識別するか？
        * 会員ID、電話番号、メアド
        * 基本的に個人情報を持ちたくない
        * 会員ID (整数)　UUIDとは別 1-10 
        * ニックネーム
        * 貸し出し時はニックネーム
    * 返却ルールのルール
        * 貸出期間の上限
        * 延滞したい時は？
        * 一人で同時に何冊まで借りれるか
    * どうやって検索したい？
        * カテゴリ・本のタイトル・作者の名前・発売日で並べ替えたい
        * and条件
正規化
カテゴリをマスタデータとして切り出さなくて良いか？
[memo]
上が終わったら
* DBの設計をする <- 相談
* 実装に詰まったら相談するでは遅いのでつまらないように相談する
* 何をするのにどれくらい時間がかかったのかの感覚を養う
 
必要なデータ(仮)
 本（蔵書）データ：Books
ID：一意の識別子（同じ本が複数あっても1冊ずつ識別するため）
タイトル
作者
カテゴリ
発売日
金額
ステータス：貸出可能、貸出中、修理


CREATE TABLE `books` (
  `id` varchar(26) NOT NULL COMMENT 'ULID'
  `title` varchar(255) NOT NULL,
  `author` varchar(100) NOT NULL,
  `category_id` varchar(26) DEFAULT NULL COMMENT 'カテゴリID',
  `release_date` date,
  `status` varchar(16) NOT NULL COMMENT '利用状況: available/lending/suspend'
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  PRIMARY KEY (`id`)
)
* id ← ULIDにするaparにあわせる
* ENUMは使わない
    * 仕様変更に対する柔軟性が低い
        * 値を変更しずらい(ALTER TABLEでマイグレーションする必要がある)
        * enumの定義をDBからだけでは確認できない
        * アプリケーション側から値の候補を取得するのが難しい
* 発売日は分からない本もあるのでNULLを許容する
created_at
updated_at
index、unique_key
カテゴリ : Category


CREATE TABLE `categories` (
  `id` varchar(26) NOT NULL COMMENT 'ULID',
  `name` varchar(26) NOT NULL COMMENT 'カテゴリ名: 哲学/芸術/文学 ',
  `created_at` DATETIME NOT NULL,
  `updated_at` DATETIME NOT NULL,
  PRIMARY KEY (`id`)
  UNIQUE KEY `uk_name` (`name`)
)
正規化
利用者データ：Users
ID
ニックネーム


CREATE TABLE `users` (
  `id` varchar(26) NOT NULL COMMENT 'ULID'
  `nickname` varchar(30) NOT NULL,
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  PRIMARY KEY (`id`)
)
 
貸出・返却記録データ：rental_records
貸出ID：記録ごとの一意の識別子
蔵書ID：どの本か
利用者ID：誰が借りたか
貸出日：いつ借りたか


CREATE TABLE `rental_records` (
  `id` varchar(26) NOT NULL COMMENT 'ULID'
  `book_id varchar(26) NOT NULL COMMENT 'ULID'
  `user_id` varchar(26) NOT NULL COMMENT 'ULID'
  `rented_at` datetime NOT NULL COMMENT '貸出日時'
  `returned_at` datetime DEFAULT NULL COMMENT '返却日時'
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  PRIMARY KEY ('id')
)
 
マイグレーション
ドメインモデル
設計 → 要件聞く-> どんなデータを保持しないといけないかを聞く → 要件を満たすテーブル設計 → データを使って何をするのか(UC) → 左をするのにどんな作業が発生するのか洗い出してみる
 
ドメインモデル・API ・画面を作る
ドメインモデルを業務として考える
 
概念を検討する ← レビュー
* 消込とかプランの利用状況・消込ルールのところを見てみるとヒントになるかも
* 業務を表現するためのdb
* 集約ルート調べる
*  
次はAPI の設計




Owner	service-infra
Author	@Yu Usami (Unlicensed)
Status	LIVING
Last Update	2024年4月24日
Original	https://docs.google.com/document/d/1OzPIvJ_3h0KdjXq1zIWe6n6THgz9TGqrcebdMY5fRhI/editチームの 80% が Google Drive のプレビューを表示しています接続
 
* このドキュメントは？
    * 対象とするサービス
    * フィードバック方法
* 原則
    * ISCG-1: サービス間はInternal/Admin APIもしくはイベントで連携する
    * ISCG-2: サービス間での相互依存・循環依存をしない
    * ISCG-3: 分散トランザクションを実装しようとしない
    * ISCG-4: 連携先のサービスが常に正常に動作することを前提としない
    * ISCG-5: バリデーションはデータを所有するサービスが責任を持って行う
    * ISCG-6: データを大量に返すList/Bulk系APIには適切な上限を定める
* 詳しい方針・標準が未定なもの
    * パフォーマンス標準
    * PubSubで用いるイベント定義標準
    * Create API 方針
    * Update API 方針
    * 共通マスタサービス利用ガイドライン
    * 分散ロックの利用方針
* ケース別実装例
    * 相互・循環依存を解決する
        * データ連携時に整合性を優先する（非推奨）
        * 結果整合で整合性をとる（データ連携時に可用性を優先する）
            * 1. Batch処理で行う
            * 2. 非同期処理で行う
            * 3. PubSub基盤を使う
            * 4. Frontend側で結果だけ先に見せる
        * サービス間でファイルをやりとりする
このドキュメントは？
freeeにおいて複数のサービスとの連携を前提としたサービスを実装する際のガイドラインを策定する。各サービス開発者が他サービスとの連携を実装する際に指針となることを目的とする。
対象とするサービス
* ドメインロジックを持つバックエンドサービス
* フロントエンドの有無は問わない
フィードバック方法
本ガイドラインは随時改訂するものとして広く改善のためのフィードバックを募集している。ドキュメントへのコメントやSlackにおける意見や質問等を歓迎する。
方針の改訂においては必要に応じてkikanchosの意見を求めつつ、サービス基盤にて決定した事項を反映するものとする。

原則
各サービスが基本的に守るべき原則を説明する。厳密に守ることが難しいケースであっても、必要性を理解しなるべく理念に沿った実装を目指すこととする。現状、これらに反する実装がなされているサービスについては、今後適切なタイミングで変更していくこととする。どうしても原則に反する実装にする必要がある際や実装に困った際には、サービス基盤（#service_infra）に相談すること。DDレビューの依頼も推奨する。
ISCG-1: サービス間はInternal/Admin APIもしくはイベントで連携する
 
* 別サービスを呼ぶ際にInternal APIを使っている
* サービスAと連携するためにサービスAのイベントをsubscribeしている
連携先サービスの利用したい機能がPrivate APIで実装されていたのでそのAPIを使うことにした
 
API呼び出しについては freee API標準 に定められている標準に従う。API標準においてはサービス間のAPI呼び出しは Internal API もしくは Admin API のみが許容されている。ユーザー起因である場合は Internal API、そうでないものは Admin API を適切に使い分けるようにする。
またPubSubを用いたイベント経由のサービス間連携も推奨する。API呼び出しとどう使い分けるべきかは循環依存を解消する目的や、自然な依存の向きを考慮して決めることが望ましい。イベント定義に関する標準については今後詳しく定められる予定。
 
ISCG-2: サービス間での相互依存・循環依存をしない
 
サービスAからサービスBの連携はAPIを呼ぶことで実現し、サービスBからサービスAの連携はサービスBのイベントをサービスAがsubscribeすることで実現した
サービスAからサービスBの連携、サービスBからサービスAの連携、どちらもAPI呼び出しで実現した
 
サービス間の依存はソフトウェアでのコードの依存関係と同様に、相互依存・循環依存をしないようにしておくことが望ましい。相互依存・循環依存があると依存の結合度が強くなり、それぞれのサービスを開発する際に考慮する事項が増えたり、デプロイ順序が問題になったりと複数チームでの高速な開発を阻害する強い要因になりうるので、基本的に排除することが望ましい。
ここでいうサービス間の依存とは、他サービスの機能を利用して実現する機能があったり、他サービスのことを知っていることが前提の動作をしていることなどを指す。「サービスAがサービスBのAPI呼び出しをしている」「サービスAがサービスBのイベントをsubscribeしている」などは「サービスAがサービスBに依存している」と言える。この依存の向きを相互に向かない、複数サービス間で循環させないようにすることを原則とする。完全に取り払うことが難しいケースであれば、相互の依存をイベントのsubscribeに置き換える、などの依存の中でもなるべく弱い依存で留められないか検討をすることを求めたい。
依存の強弱については明確な定義をしているわけではないので、同期的なAPI呼び出しよりも非同期のAPI呼び出しの方が依存としては弱い、といった概念的なものにすぎません。イベントをsubscribeするのと非同期でAPI呼び出しをするのは依存の強さとしては同程度と考えられます。
サービスがどのような依存関係を持っているかは現状確認することが簡単ではないので、ここは何らかの確認しやすくなる手段を今後用意したい。
 
ISCG-3: 分散トランザクションを実装しようとしない
 
サービスAのデータ更新で非同期処理をenqueueし、サービスAの非同期処理でサービスBにAPI経由でデータを追加する
* サービスAのデータ更新で整合性を保ってサービスBのデータを作成したいので、両サービス間の分散トランザクションを頑張って実装した
* サービスAのデータ更新トランザクション内にて同期的にサービスBのデータを更新したいので、サービスAのデータ更新トランザクションの beginーcommit 間でサービスBのデータ更新APIを呼び出した
 
分散トランザクションは正しく実装することが難しく、容易に障害の原因となり得るためfreeeでは実装を推奨しない。サービス間でのデータ整合性を保つ必要があるケースにおいては、一時的な不整合を許容しつつ、後述する結果整合で整合性をとる実装パターンから要件に沿った方法を選択することを推奨する。
また、整合性をもったトランザクション内で他サービスを呼び出したい、という意図で beginーcommit 間にて他サービスのAPI呼び出しを行う、というパターンは推奨しない。これはそもそもトランザクションのロールバックが適切に行われる実装になっていないこと、コミット待ちの間にネットワーク越しの他サービスを呼ぶことでロックが取られたまま長い待ちが発生していることなど、良くない特性がいくつかあるため。
ACID特性を保ったトランザクションが必要な操作については、そもそもサービスを分割させないといったドメイン境界の判断から再度確認することが大切である。
 
ISCG-4: 連携先のサービスが常に正常に動作することを前提としない
 
* サービスAのデータXからサービスBのデータYへの参照があり、サービスAの画面表示でデータYの詳細を表示する際に、サービスBへの参照リクエストが失敗してもfallbackが表示できるようにしておく
* サービスAの非同期処理からサービスBのデータを作成する際、なんらかの理由でデータ作成に失敗した場合は成功するまでretryする（最大retry数は適切に設定し、最大retry数に達して失敗した際はbugsnag等で通知する）
* サービスAからサービスBを呼び出す際、10sのタイムアウトを設定して呼び出した
* サービスAの操作Xをする際にサービスBのデータを更新するAPIを同期呼び出しをしており、サービスBに障害が起きた際サービスAの操作Xが使用不可能となる
* サービスAからサービスBのデータを作成するAPIを同期呼び出しをし、その結果をサービスAのデータにも反映している時、サービスBに障害が起きた際にタイミングによってデータの不整合が発生する
* サービスAからサービスBを呼び出す際、タイムアウトは設定せず呼び出した
 
他サービスとネットワーク越しに連携する以上、「サービスBのみメンテナンスに入っている」などの理由によって連携先のサービスが正常に動作しない、正常なレスポンスが返ってこないことを想定する必要がある。この点から、連携先サービスへのAPI呼び出しは基本的にretryをすることになるが、Create API (POSTなどで新しくデータを作成するAPI）においては、後述する冪等性(idempotency)の担保がなされていることを確認する必要がある。
また、他サービスの呼び出しの際は、必ずタイムアウトを設定することとする。gPRC 呼び出しであれば Omega のクライアントを用いれば60sのタイムアウトがデフォルトで設定されている。HTTP 呼び出しの場合は各サービスで用いるクライアントにて適切に設定することが必要となる。Faradayを使っているのであれば、デフォルト値としてopen_timeout 60sとread_timeout 60sが設定されている。合計で最大60s以下になるように適宜設定することを推奨する。 
 
ISCG-5: バリデーションはデータを所有するサービスが責任を持って行う
 
サービスAのデータを作成するAPIは単一のInternal APIに統一されており、サービスAのデータに関してはそのAPIにて作成する際にサービスAのドメインロジックによりバリデーションが適切にされる
フロントエンドサービスBがサービスAのデータをAPIを介して作成する際、フロントエンドサービスB側できっちりデータのバリデーションを行う前提なので、サービスA内部でのデータバリデーションは最低限にした
 
各サービスのデータは各々がオーナーとなり責任をもつ。データの作成はオーナーのサービスを介してしかできないことは前提として、データのバリデーションについてもオーナーのサービスが必ず行うこととする。
この時データを作成するためのAPIが複数存在すると統制が取りにくくなるため、基本的にはデータ作成のためのAPI については汎用的なものを用意して他サービスから適切に利用させる方針を推奨する。
 
ISCG-6: データを大量に返すList/Bulk系APIには適切な上限を定める
 
サービスAのデータをlistで取得するAPIは、最大でも1リクエストにつき500件のデータを返すことを上限とし、必要があればpage番号を指定することで順次取得できるpagenationが実装されている
サービスAのデータをlistで取得するAPIは、指定パラメータに合致するデータは全て1リクエストで返却する実装になっている
 
freee API標準 サービス基盤 FY22Q3 に推奨する実装方針が定められている。
List/Bulk APIに適切な上限やpagenationが実装されておらず、大量のデータが一度に送られることにより元のリクエストがタイムアウトを起こす障害が発生したことがある。2023-10-13 [DEV_LV3->Lv2][INCID-331] エラーが表示され請求書がcsvインポートできない
適切に上限を設けることで意図しないデータ量がサービス間で送信されることを防ぐと共に、上記のタイムアウトを設定する意図と合致させるためにも、適切な時間内に送信完了できる量のデータを上限に定めておくべきである。その際には、全データを取得する目的にも正しく使えるように、pagenationを実装し必要があれば呼び出しサービス側で適切に扱えるようにする。
 

詳しい方針・標準が未定なもの
今後このガイドラインもしくは別の標準化ドキュメントにて規定していきたいが、標準化するに至っていない観点についてこちらに列挙する。
パフォーマンス標準
現状は各サービスのパフォーマンスについては各サービスオーナーのチームに任されており、共通化されたSLOなどは定められていない。サービス呼び出しの際のタイムアウトの設定については言及したが、具体的にそれぞれのサービスの全てのエンドポイントが推奨する60s以内にレスポンスを返せることは現状保証されていない。このあたりを含めてSLOとして整備し、ダッシュボード等でモニタリングできるようにしていく必要性があると考えている。
PubSubで用いるイベント定義標準
PubSubの基本的な使い方や導入方法についてはPubSub基盤導入Doc サービス基盤 FY23Q4を参照する。イベントの定義についてはこちらに基本方針が記載されている。連携ガイドラインとして使いやすいイベント定義標準を今後定める予定。
Create API 方針
Create API（POSTでレコードを作成するAPI）については、冪等性（idempotency）を担保するように作る方針にしていきたい。現状はまだ実装標準が定まっていないが、idempotency keyをparamとして受け取れるようにし、そのkeyが一致するリクエストに関しては新たにレコードを作成しないようにするなどが考えられる。これは連携先サービスが正常に動作することを前提としない作りこみをする際に、retry機構などを組み込むことが考えられるが、正常にレコードは作成したがレスポンスが正しく到達しなかったケースなどにおいて再度同じリクエストがretryされた場合に重複してレコードが作成されることを防ぐために必要な方針となる。過去に同様のケースで障害が発生したことがある 2023-04-28 [DEV_LV2][INCID-51] freee請求書経由で取引が重複して登録されている
Update API 方針
Update API（現状PUTでレコード更新をしているAPI）についても、互換性担保の観点からPATCHによる部分更新を基本にするべきではないか、という議論もあり、今後推奨する方法が定まったらガイドラインを更新する。
共通マスタサービス利用ガイドライン
共通マスタサービスが今後実装されていくが、これらはnest-authなどの基盤サービスと同様に他のサービスから依存されることが多く、無秩序に利用されることは避けたいため、ルールに則った利用が推奨される。具体的なガイドラインについては今後共通マスタサービスの実装が具体的になってから整備していく予定。
分散ロックの利用方針
Dynamolockを用いた分散ロックによってサービス利用の排他制御を行うことが可能だが、実装時に気をつけるべきことが多い。デッドロックの発生に繋がることも考えられるため、気軽に利用をすることは避けたいが、必要な部分に効果的に使うことは重要だと考える。今後利用の幅が広がり、ある程度ルールとして整備することが可能になった際に改めて方針を定めたい。

ケース別実装例
原則を実際にどう適用して実装すれば良いのか迷った時のため、よくある実装パターンについて実装例を提示する。実装方針を決める際には
* 各実装パターンにおけるトレードオフ
* 要件によって最適な実装は異なること
* そもそもの要件（許容される不整合や不整合な期間等）が妥当であるか
といった点を留意する必要がある。
必要があれば要件を再度PdMと相談したり、実装方針について #kikanchos で相談してみましょう。
相互・循環依存を解決する
 
例
freee会計とfreee請求書の依存関係を考える。 freee請求書で作成した請求書に紐づくfreee会計の取引を作成し、freee会計の取引のステータスが変更された際に、freee請求書側の取引ステータスも更新する処理を行う。
 
この例の連携を実装する際に、単純に実装するのであれば以下の４つの方法が考えられる。
 
実装案1（会計と請求書が相互依存: 非推奨）
https://www.figma.com/board/6kgqNn3k772ZTVBeBqSCLn/%E3%82%B5%E3%83%BC%E3%83%93%E3%82%B9%E9%96%93%E9%80%A3%E6%90%BA%E3%82%AC%E3%82%A4%E3%83%89%E3%83%A9%E3%82%A4%E3%83%B3?node-id=5-234&t=S1VVO5S7uKOJvZVo-4
 freee請求書から請求書に紐づいた取引を作成する際は、freee会計APIを呼び出して取引を作成する（請求書から会計への依存）。freee会計では取引ステータスを更新する際に、取引が請求書idを持っていた場合に、freee請求書APIを呼び出してfreee請求書内の取引ステータスを更新する（会計から請求書への依存）。
 
実装案2（会計から請求書への依存: 非推奨）
https://www.figma.com/board/6kgqNn3k772ZTVBeBqSCLn/%E3%82%B5%E3%83%BC%E3%83%93%E3%82%B9%E9%96%93%E9%80%A3%E6%90%BA%E3%82%AC%E3%82%A4%E3%83%89%E3%83%A9%E3%82%A4%E3%83%B3?node-id=5-164&t=S1VVO5S7uKOJvZVo-4
 freee請求書でpubsubを用いて、請求書に紐づく取引が作成された際に連携取引作成イベントをpublishする。freee会計はその連携取引作成イベントをsubscribeし（会計から請求書への依存）、会計内に請求書idとの紐付けと共に取引を作成する。freee会計にて取引を更新する際に、請求書idをもつ取引だった場合はfreee請求書の取引ステータス更新APIを呼び出して、freee請求書側の取引ステータスを更新する（会計から請求書への依存）。
 
実装案3（請求書から会計への依存）
https://www.figma.com/board/6kgqNn3k772ZTVBeBqSCLn/%E3%82%B5%E3%83%BC%E3%83%93%E3%82%B9%E9%96%93%E9%80%A3%E6%90%BA%E3%82%AC%E3%82%A4%E3%83%89%E3%83%A9%E3%82%A4%E3%83%B3?node-id=5-48&t=S1VVO5S7uKOJvZVo-4
 freee請求書から請求書に紐づいた取引を作成する際は、freee会計APIを呼び出して取引を作成する（請求書から会計への依存）。
freee会計ではpubsubを用いて、取引ステータスを更新する際に取引更新イベントをpublishする。freee請求書はその取引更新イベントをsubscribeし（請求書から会計への依存）、freee請求書から連携していた取引のステータスが更新されていた場合に、freee請求書側の取引ステータスを更新する。
 
実装案4（会計と請求書の直接の依存はなし）
https://www.figma.com/board/6kgqNn3k772ZTVBeBqSCLn/%E3%82%B5%E3%83%BC%E3%83%93%E3%82%B9%E9%96%93%E9%80%A3%E6%90%BA%E3%82%AC%E3%82%A4%E3%83%89%E3%83%A9%E3%82%A4%E3%83%B3?node-id=5-75&t=S1VVO5S7uKOJvZVo-4
 freee会計とfreee請求書の連携を行うための取引連携サービスを新しく導入する。freee請求書から請求書に紐づいた取引を作成する際は、取引連携サービスのAPIを呼び出して取引連携を作成する（請求書から取引連携サービスへの依存）。freee会計では取引を更新する際に、取引連携サービスのAPIを呼び出して取引連携更新を通知する（会計から取引連携サービスへの依存）。
取引連携サービスはpubsubを用いて、取引連携作成イベントと取引連携更新イベントをpublishする。freee会計では取引連携作成イベントをsubscribeし（会計から取引連携サービスへの依存）、請求書に紐づけられた取引を作成する。freee請求書では取引連携更新イベントをsubscribeし（請求書から取引連携サービスへの依存）、請求書に紐づけられた取引の更新であればfreee請求書内の取引ステータスを更新する。
 
実装案1はシンプルな実装ではあるが、相互依存することとなるため非推奨。実装案2はこの形式も可能である、というだけで依存の方向性として無理がある。請求書はそもそも会計の周辺ドメインであるのに、会計側が請求書に依存するという依存の方向性は歪であるため非推奨。
この例であれば、実装案3と実装案4のどちらかを、さらにそのほかのサービスとの依存関係・イベント流量・実装コストなどを勘案して選択することが望ましい。
データ連携時に整合性を優先する（非推奨）
 
（注記）下記の結果整合のパターンと対になる形で整合性を優先するとこうなる、というパターンを記載したが、実装例で示したケースにおいても整合性が保てない可能性が残るため基本的には推奨しないこととした。
 
例
サービスAのデータに紐づく取引をfreee会計に作成する
 
実装例
サービスA側でユーザーによりデータXに紐づく取引を作成する操作がなされた際、同期的なAPI呼び出しにてfreee会計の取引を作成するAPIを呼ぶ。freee会計の取引作成が失敗することを念頭に、適切な回数リトライを行う。この際、冪等性を考慮して同一のデータXに紐づく取引が複数作成されないようにする。freee会計への取引作成と共に、サービスAのデータも取引連携済み、にステータス更新をするが、freee会計へのAPI呼び出しはサービスAのDBトランザクション内で行わない。
簡易的なシーケンスは以下のようになる。
 
1. サービスAにてユーザーからのデータXに紐づく取引の作成リクエストを受ける
2. サービスAからfreee会計の取引作成APIを呼び出す
3. freee会計が取引を作成
4. サービスAがfreee会計の取引作成成功レスポンスを受ける
5. サービスAがデータXに取引idなどを含めDB内容を更新
6. サービスAがユーザーにレスポンスを返す
 
この実装例においては、サービスAの紐づく取引を作成する機能はfreee会計が利用不可能の際は同様に利用できなくなる（必ず失敗する）。
このシーケンスにおいて、3と4の間になんらかの理由（メンテナンス等）によりfreee会計からのネットワークが遮断された際に、freee会計内にはデータXに紐づいた取引が作成されたが、作成の成功レスポンスが適切にサービスAに到達せず、サービスA側では失敗処理となってしまい不整合が発生する可能性がどうしても残る。このため、この実装パターンは非推奨となっている。
冪等性を考慮した実装がなされており、さらには一方のサービスに紐づいたデータが残ってしまっても問題がない、などの条件が重ならない限り、基本的にはこの実装パターンの採用は難しい。
結果整合で整合性をとる（データ連携時に可用性を優先する）
ここで紹介する実装パターンは、結果整合で整合性を取るため、実際の連携には連携先のサービスが必ずしも動作している保証を必要としないため、サービス・機能単体での可用性は高い実装となる。トレードオフとして、それぞれのパターンによって不整合の起こり方や不整合の状態が続く時間が異なるため、サービスとしてどこまでが許容できるかによって実装方針を決める必要がある。
 
1. Batch処理で行う
 
例
freee販売での販売に紐づくfreee会計の取引について、freee会計で決済ステータスが更新されたものについてfreee販売側での決済ステータスも同様に更新する
 
実装例
freee販売にてBatch処理を実装し、決済ステータスが未決済になっているものについてはfreee会計に問い合わせ、決済完了になっていた場合にはfreee販売側の決済ステータスも決済完了に変更する。Batch処理はkubernetesのCronJob機構を用いて日に数回実行する。
 
許容されている不整合
実際にfreee会計にて決済完了に変更されてからBatch処理がfreee販売側の決済ステータスを決済完了に変更するまでの数時間は両者のデータが不整合となる。CronJobが実行される間隔が不整合が起こりうる最長の時間となるので、CronJobを頻繁に実行することで不整合である時間を短くすることは可能。
 
2. 非同期処理で行う
このパターンはfreee内ではまだ実装例がないと思われるため、データ連携時に整合性を優先するパターンの例を非同期処理で行う実装案を記載する。
 
例
freee請求書の請求書に紐づく取引をfreee会計に作成する
 
実装案
freee請求書側でユーザーにより請求書に紐づく取引を作成する操作がなされた際、非同期処理経由でfreee会計の取引を作成するAPIを呼ぶ。非同期処理のJobはfreee請求書側に実装され、resque等を利用して実行される。非同期処理ではAPI呼び出しが失敗した際も適切にリトライを行う。この際、整合性を優先する実装例と同じく、冪等性を考慮する必要がある。freee請求書側では請求書に紐づく取引を作成する操作がなされた段階で取引連携済み、とするが、実際には非同期処理にて遅れてfreee会計側に連携された取引が作成されることとなる。また、freee会計で作成した取引のid等をfreee請求書側にも登録する必要がある場合は、pubsubなどの仕組みで取引作成後にfreee請求書へ通知する必要がある。
簡易的なシーケンスは以下のようになる。
 
1. freee請求書のfrontendにてユーザーが請求書に紐づく取引を作成する
2. freee請求書がfreee会計の取引を作成するAPIを呼び出す非同期処理をenqueue
3. freee請求書が請求書のステータスを取引連携済みに更新
4. freee請求書がユーザーにレスポンスを返す
5. (以降background処理で2以降のタイミングで実行）freee請求書で非同期処理が実行され、freee会計の取引作成APIを呼び出す
6. freee会計が請求書に紐づいた取引を作成
7. freee会計が取引作成イベントをpublish
8. freee請求書が取引作成イベントをsubscribeし、請求書に紐づいた取引が作成されていたら取引idなどを含めてDB内容を更新
 
許容されている不整合
実際にfreee請求書にてユーザーが取引作成のアクションをとってレスポンスを受け取ってから、非同期処理にてfreee会計に取引が作成されるまでの数秒から数分間、連携済みとなっていても対応する取引が存在しない。さらに連携した取引idがfreee請求書に同期されるまでは請求書側のデータも会計の取引と関連付けることができない。
 
 
この実装パターンでは、freee会計がサービス利用不可の状態でもfreee請求書の連携取引を作成する機能自体は使うことができ、リトライ等が適切に実装されていればfreee会計が復旧した後にそのままデータ連携が可能となる。
最終的にfreee請求書側に請求書に紐づく取引idが登録されるまでは、連携する取引へのリンク等が使えないので、適切に実装しておく必要がある。
 
3. PubSub基盤を使う
PubSub基盤は基本的には依存関係の解消が本来の目的ではあるが、整合性を結果整合で取るパターンに利用することもできる。
 
例
新規の事業所が作成された際にebisにてzuoraのアカウントを作成する
 
実装例
新規の事業所が作成されると、nest-authがPubSub基盤を用いてServiceEventを発行する。ebisにてServiceEventをsubscribeし、新規の事業所作成であればzuoraに当該事業所用のアカウントを作成する。
nest-authのPubSub基盤は、outbox-capturerパターンを利用してイベントが消失しないことを保証している。マスタ系やnest-authのようなマスタに準ずる基盤サービスにおける重要なイベントには、outbox-capturerパターンを用いることが推奨される。ebis側はpush-subscriberを用いてイベントのsubscribe、ebisのInternal API呼び出しを行う。zuoraのアカウント作成はそのInternal API内にて行われる。
 
許容されている不整合
各プロダクトもしくはfreeeアカウント管理にて新規事業所が作成されてから、実際にebisにてnest-authのイベントがsubscribeされebis側での処理がなされる数秒から数分間、データとしては不整合な状態となる。
 
4. Frontend側で結果だけ先に見せる
このパターンの実装例はfreeeにはないため、実装案のみ記載する。
 
例
freee会計の画面にて取引先を編集し、ユーザーは編集の保存ボタンを押すと、即座に画面に反映される
 
実装案
取引先マスタへの更新依頼を非同期処理で行い、本来は取引先マスタから取得した最新の状態を表示するfrontend側で、あたかも更新が反映されているようにユーザーに見せる。
簡易的なシーケンスは以下のようになる。
 
1. freee会計のfrontendにてユーザーが取引先編集の保存ボタンを押す
2. freee会計が取引先マスタの更新APIを呼び出す非同期処理をenqueue
3. freee会計のfrontendでは取引先の変更が完了したものとして表示部分を編集内容に入れかえて描画
4. (background処理で厳密には3の後と限らない）freee会計で非同期処理が実行され、取引先マスタの更新APIを呼び出す
5. 取引先マスタがDB内容を更新
 
許容されている不整合
ユーザーが操作をしてから、実際に非同期処理が取引先マスタのAPIを呼び出しDBが更新されるまでの数秒間は表示画面と取引先マスタのDBの内容が不整合となる。
DB更新が失敗した際は、ユーザーが操作をした、という行動とDB内容が一致しなくなる。
 
 
実際のDB内でのデータ更新は裏で数秒の時間をかけて行うが、数秒待たせるのはユーザー体験が悪く、即座に変更が反映されているように見せたい場合に利用できるパターン。実際の実装では、DB側の更新が失敗した際にはどう見せるか、リロードをした際にはどう見せるか、などを考慮する必要があることや、ある程度複雑な実装を行う必要があるため、本当にクリティカルな体験部分にのみ導入の検討をすること。
 
サービス間でファイルをやりとりする
 
例
取引先マスタサービスにてユーザーからcsvファイルを受け取り取引先をインポートする機能を実装する。 取引先マスタはcommon-settingsがフロントを担うサービスとなるので、ファイルの中身をユーザーからcommon-settingsが受け取り、取引先マスタサービスがその後に何らかの形式で受け取る必要がある。
 
実装例
S3を一時的なファイル置き場として利用し、サービス間で大きなファイルを直接送り合うことを避ける。ファイルの中身についてはドメインロジックを持つべきサービスがバリデーションを行う。
簡易的なシーケンスは以下のようになる。
 
1. ユーザーがcsvファイルをアップロード
2. common-settingsがユーザーからのファイルを受ける
3. common-settingsがファイル形式の簡易的な確認（csvヘッダなど）
4. common-settingsがS3にファイルアップロード
5. common-settingsがアップロードされたS3のファイルパスと共に取引先マスタのインポートAPIを呼び出し
6. 取引先マスタがS3からアップロードされたファイルをダウンロード
7. 取引先マスタがファイルの中身をバリデーションする
8. 取引先マスタが取引先をインポートする
 
 
freee API標準
Owner: アプリ基盤サービス基盤ヨット
Author: Yui Terashima
Last Updated: 2026/03/04
Status: living document
Translation Ticket: TRAN-14 (English version is here)
Original: https://docs.google.com/document/d/142dAD2_J6kxtxkKZxOKbJmYpTvjNzpd_KapYS2toTVw/editチームの 80% が Google Drive のプレビューを表示しています接続 
 
* freee API標準
    * このドキュメントは？
        * ここで定めるAPI標準のスコープ
        * スコープ外のトピック
    * APIの分類
        * 種別
        * プロトコル
        * エンドポイント
    * 認証
        * ユーザー認証
        * サービス認証
    * メタデータ
        * リクエストメタデータ
        * レスポンスメタデータ
    * エラー処理
        * 汎用エラー情報
        * エラーフォーマット・構造
        * Severity
    * APIドキュメント
    * OpenAPI
    * gRPC
        * プロトコル定義
        * 周辺ツール
        * クライアント
        * 生成ファイル
    * その他
        * omegaによるサポート
        * テスト
        * 日付時刻
        * ページング
        * GraphQL
このドキュメントは？
freeeにおけるサービスのAPIの標準を提案する。
ここでいうサービスのAPIとはfreeeが開発・運用するサーバーへのHTTPベースのリクエスト＆レスポンスのことを指す。
 
API Working Groupにて議論してまとめたものをLiving Standardとして公開し随時改善中。
詳細を決めきれていない部分もあるが、方向性に対する意見やこの辺どうなっていくのか知りたいなどドキュメントへのコメントやSlackでの議論などでフィードバックをしてもらえるとありがたい。
 
English version: freee API Standard Service Platform FY22Q3
ここで定めるAPI標準のスコープ
* APIの分類
* 認証手段
* APIのプロトコル定義方法（OpenAPI / gRPC）
* リクエストレスポンスの形式
* APIのための周辺ツール
スコープ外のトピック
* APIサーバーの構成
* AWSリソース
* Gateway, Service Discovery, Service Mesh
* パフォーマンス・SLI/SLO
* サービス分割
* ドメインロジックの実装
* テスト手法




APIの分類
APIは用途による種別と、呼び出しのためのフレームワークで分類する。
種別
* Public API
    * 一般ユーザー、外部開発者、連携先がfreee外から操作を行うためのAPI
* Private API
    * freee製アプリ（Web、モバイル、デスクトップ）がfreee外から操作を行うためのAPI
    * （古いPrivate APIは内部からのアクセスもあるが、これらはInternal APIに寄せていく）
* Internal API
    * freee内部のサービス間でユーザー起因の操作を行うためのAPI
    * 非同期処理やadminがbecome/assume（ユーザー操作のエミュレート）する場合も含む
    * （古いInternal APIはユーザー起因でないものも含むが、これらはAdmin APIに分離していく）
* Admin API
    * freee内部よりシステム起因の操作を行うためのAPI
    * admin画面、adminタスク、バッチ、PubSubなどから各サービスを呼ぶ
* Admin Web API
    * admin画面のブラウザから操作を行うためのAPI
    * （現状は一般サービスにもあるが、central-adminのようなadmin専用サービスのみが提供する形にしていく）
￼
 
プロトコル
各種APIは形式的なプロトコル定義によってそのエンドポイントとリクエスト・レスポンスを定める。
プロトコル定義の形式としてはOpenAPIとgRPCを用いる。
* OpenAPI
    * https://swagger.io/specification/
    * Public APIは全てこれで定義されている
    * JSON basedなAPIに仕様定義と型付けができる
        * 既存のAPIには順次定義を付けていく
    * 新しいものはスキーマ駆動開発を推奨したい
* gRPC
    * https://grpc.io/
    * マイクロサービス間の通信に適した特性
        * 常時接続&バイナリ化で通信オーバーヘッドが少ない
        * 互換性を維持したままインターフェースを変更するためのプラクティスが豊富
 


	OpenAPI	gRPC
Public API	◎（必須）	✕（ユーザーへの直接提供はしない）
Private API	◎（推奨）	✕（非推奨）
Internal API	◯（Railsならこちら）	◎（gRPCに寄せていきたい）
Admin API	◯（Internal APIに合わせる）	◎（Internal APIに合わせる）
Admin Web API	◎（推奨）	✕（非推奨）


エンドポイント
OpenAPIの場合はNginxのルールで制御しやすいようにまずAPI種別でパスの先頭を分ける。その上でモジュールまたはクライアントによってパスを切る。
 


API種別	パスprefix	用途
Public API	-	（別途規定）
Private API	/api/p/	そのプロダクトのWebクライアント向けBFF
	/api/m/	Mobileクライアント向けのBFF
	/api/:module/	モジュラモノリスとして共通moduleが乗っている場合の、プロダクト横断機能用
Internal API	/api/internal/	独立したDomain Serviceの場合
	/api/internal/:module/	モジュラモノリスのmodule単位で機能を切る場合 Domain ServiceをAPI Serviceに相乗りさせている場合はこちらを推奨
	/api/internal/:client/	BFF的に呼び出し元のサービス単位で機能を切る場合
API Serverとしてドメインをまたぐ操作がある場合はこちらを推奨
Admin API	/api/admin/	（※2026/2 に変更）プロダクトサービスの場合
	/api/internal/	admin専用サービスの場合
host名でadminと分かるようにしておく
Admin Web API	/api/private/	admin専用サービスの場合
host名でadminと分かるようにしておく
	/admin/api/	プロダクトサービスの場合（新規に作成は非推奨）


 
gRPCの場合はProtocolBuffersのpackageをサービスまたはドメインの粒度で切る。Admin APIの場合はgRPCサービス名をXxxAdmin または XxxAdminServiceとすること。
認証
基本的に全てのAPIは何かしらの認証により操作者を特定しないと操作できないようにする。
以下の目的より、外部だけでなく内部のAPIも認証必須にする。
* LeakCheckableによる意図しない操作の防止を確実にする
* セキュリティインシデント時の影響範囲を狭める
* admin等を監査しやすくする
ユーザー認証
誰が操作したかの確認。特定事業所内での操作の場合はcompany_idも認証対象に含める。
 


	認証手段	認証しているID
Public API	OAuth2	user_id
Private API	LoginSession or OAuth2 + cid	user_id + company_id
Internal API	InternalSession	user_id + company_id
Admin API	AdminSession or AdminInternalSession（準備中）	admin_user_id
Admin Web API	AdminSession	admin_user_id


 
* OAuth2 access token
    * 一般ユーザーやパートナーによるPublic APIアクセス用
    * 多くのエンドポイントではcompany_idはパラメーターで指定する形になる
* OAuth2 access token + cid
    * モバイル等freee謹製アプリがPrivate APIにアクセスするとき用
    * 利用可能なOAuth Applicationはホワイトリストで登録されている
    * OAuth2 access tokenに加えて、ヘッダに x-freee-company-id を付ける
        * Userが対象Companyにアクセス可能であるかのチェックを行うので呼び出し元の指定だけで問題ない
* LoginSession
    * Cookieによるログインセッション
    * 基本的にはWebブラウザからのリクエストで用いる
    * freee-accountsでのログイン、もしくはbecomeにより生成
* InternalSession
    * 操作主体のUser/Companyの基本情報を署名したもの
        * JWTの亜種 (protobufベース)
    * session-transformerにてLoginSession or OAuth2+cidから変換する
    * Resqueのような非同期処理にも引き回す
    * AdminSessionからCompanyを指定してAssumedInternalSessionとして生成することもある
* AdminSession
    * 現行のadmin画面のログインセッション相当
    * バッチやadminタスクの実行者もMachine Admin Userとしてadmin_user_idを付与し、セッションを発行する
* AdminInternalSession
    * InternalSessionのadmin版
    * （2026/2現在準備中）
 
Railsにおいては current_user / current_company / current_admin_user で認証済みのリソースにアクセスするのを基本とする。
サービス認証
意図しないサービスからのリクエストを受け付けないように、アプリケーション側でもSecurity Groupに近い粒度でサービス間のアクセスを制御する。
mTLSが使えそうだが、詳細は決まっていない。
メタデータ
リクエストメタデータ
主にログ目的でリクエストのメタデータをサービス間呼び出しで引き回す。
* x-request-id
    * ユーザーからのリクエストのunique ID
* x-requested-time
    * ユーザーからのリクエストを受け取った時間
* x-origin-service-name
    * ユーザーからのリクエストを受け取ったサービス
* x-caller-service-name
    * 呼び出し元のサービス
* x-caller-context-name
    * 呼び出し元のcontroller/action名 or RPC名
* x-tracking-key
    * セッション or AccessTokenに対してuniqueな値
* x-writable
    * 書き込みを許可しているか否か
* x-any-company-accessible
    * current_company以外のCompanyのデータにアクセスを許可しているか否か（LeakCheckable用）
レスポンスメタデータ
* x-request-id
* x-tracking-key
エラー処理
エラーは汎用的なエラー情報を共通のフォーマットで引き回すことで、大域のエラーハンドラーで一律に処理できるようにする。
必要に応じて個別に処理するのは構わない。
汎用エラー情報
* System message
    * freee内部向けの詳細なエラーメッセージ
* User Message
    * end userに出力しても問題ないエラーメッセージ
* Error class
    * エラー処理用の分類
    * Bugsnagのエラークラス
* Error code
    * サービス固有prefix + カテゴリー分けされたコード
        * e.g. ACC-0012-0034
        * サービス固有prefixは 英字3文字
        * 残りはハイフンで連結した任意個の数字のみのコード
        * prefixとハイフン含めて20文字以内
    * web/public APIでのメッセージの出し分けや、サポート問い合わせに使う想定
        * ユーザーに表示することがあるので、User Messageと同じ粒度にするとが望ましい
        * いずれはError codeと表示箇所・言語からUser Messageを生成できるようになるのも見越している
    * エラーコードは一覧のスプレッドシートで管理する
        * Error code list dev FY22
        * ひとまずの対応でコードとまとめて管理しやすい形を模索する
        * 各サービスで機械的な定義をしているならばそちらへのリンクでも構わない
            * 試行錯誤していいやり方があれば標準化する
* Attachments
    * 任意の追加キーバリューペア
    * frontendの表示向けにユーザーに見えるところに出力する
* Metadata
    * 任意の追加キーバリューペア
    * Userに出さないもの
    * Error class/Error codeの追加情報
* Validation
    * フィールド単位のvalidation error
    * [フィールド名, メッセージ, metadata]の列
* 拡張
    * 追加の構造化された情報が追加されることを想定しておく
エラーフォーマット・構造
* Go error
    * omega-go/errorsで実装
    * err.Error()はSystem Message
    * wrap構造で情報を拡張可能
* gRPCレスポンス
    * gRPCの拡張エラーメッセージを使う
    * Status codeはドメイン上のエラーでは当てにしない
        * インフラのエラーや4xx or 5xx系の判別には使う
    * Statusのerror messageはSystem Message
    * omegaにてエラー用protobuf Messageを定義
* Ruby例外
    * omega-ruby/errorsで実装
    * e.messageはSystem Message
    * クラスがError classになると扱いやすいが、レスポンスを引き回す場合などは共通エラークラスのattributeでもいい
* JSON (OpenAPI)
    * Public APIは外部開発者が分かりやすいメッセージに変換
    * Private APIではsystem向けの情報を除去し、クライアントが扱いやすい形に


{
  message: "User Message",
  code: "PAY-001-0001",
  fields: [
    {
      name: "フィールド名",
      message: "項目ごとのエラーメッセージ",
      attachments: { length: 100 } # 追加情報
    }
  ],
  attachments: { employee_id: 1 } # 追加情報
}
Internal APIはエラー情報をそのままJSONに


{
  system_message: "System Message",
  user_message: "User Message",
  error_class: "InvalidError",
  error_code: "PAY-001-0001",
  invalid_fields: [
    {
      name: "フィールド名",
      message: "項目ごとのエラーメッセージ",
      attachments: { length: 100 } # 追加情報
      metadata: { item_id: 88 } # 内部追加情報
    }
  ],
  attachments: { employee_id: 1 } # 追加情報
  metadata: { special_hash: "deadbeaf" } # 内部追加情報
}
* 相互に変換するのはomegaに実装
Severity
エラーをどう処理するかを規定し、大域のエラーハンドラーでSeverityに応じて通知などの処理を行う。
基本的にError classが対応するSeverityを持つが、拡張で上書き可能にしてもいい。
* UserError
    * ユーザーによる入力・操作に起因するエラーで、システム側の修正を必要としないもの
    * 400エラー相当
    * どうしたらいいかユーザーに表示する
    * Bugsnag通知はしない
    * ログには残す
* SystemError
    * システム側の不具合によるエラー、想定外のエラー
    * 500エラー相当
    * ユーザーにはサポートに問い合わせるように案内する
        * エラーの詳細はユーザーに見せない
    * Bugsnag通知する
    * ログに残す
* FatalError
    * 緊急に対応する必要があるシステムの問題
        * 通常のエラーでは必要ないべき
        * エラー記録中のエラーなど
    * ユーザーにはサポートに問い合わせるように案内する
    * Slack等で通知する
    * Bugsnag通知する
    * ログに残す
APIドキュメント
定義ファイルからドキュメントを自動生成し、社内から容易に見れるようにしておく。
置き場所はドキュメント標準化に合わせて決めるので、今はTBD (Github pages or S3 or else)。
OpenAPI
OpenAPI標準実装

 を参照
gRPC
プロトコル定義
* 命名規則
    * 基本Google Cloud APIの命名規則に倣う
        * 例外：リソース指向のメソッドでGet/Create/UpdateなどでもGetHogeResponseの形式を取る
        * バージョニングなどPublicなAPI向けの機能はInternal用途ではわざわざ使わない
    * 設計パターンも参考になる
* エラーは基本的には汎用エラーを拡張エラーに載せる
* packageと言語ごとのオプションを指定し、サービスごとに名前空間を分ける
* 定義ファイルは proto-def/{package}/ ディレクトリに配置
* TODO: スタイルガイドを別ドキュメントで用意する
周辺ツール
* コード生成
    * https://github.com/protocolbuffers/protobuf
    * https://github.com/protocolbuffers/protobuf-go
* プラグイン
    * https://github.com/C-FO/omega-go/tree/master/protoc-plugin/protoc-gen-zap-marshaler
        * セキュリティフィルタ付きのロギング
* formatter
    * clang-format (--style=google)
* Linter (まだTBDで候補のみ)
    * https://github.com/bufbuild/buf
    * ...
* ドキュメント生成
    * TBD
* vendoring
    * https://github.com/stormcat24/protodep があるが機能不足
    * TBD: 多分protodepをomegaでラップする
クライアント
* 各サーバーのrepositoryでクライアントライブラリを提供
    * Ruby用にgem、Go用にmoduleを配布
生成ファイル
* Go用
    * proto/ 以下に配置
        * 複数packageの場合は proto/{package} に分ける
* Ruby用
    * proto-ruby/lib/{package} 以下に配置
    * moduleは Proto::Package::
    * gem名はproto-{package}
その他
omegaによるサポート
* サーバー
    * ロギング
    * 認証
    * メタデータ、認証情報の引き回し
* クライアント
    * メタデータ、認証情報の引き回し
テスト
* 依存サービス
    * APIの実行にさらにバックエンドサービスのAPIが必要になる場合は、APIのテストの際に実際のバックエンドサービスを立ち上げて一気通貫でテストするケースと、バックエンドサービスをモックとして立ち上げるケースがある
    * TODO: 標準的なモックサービスの構築方法を定める
* 認証
    * TODO: 各種認証手段の偽装方法
日付時刻
* OpenAPIはStringのformatとしてdate / date-timeを用い RFC3339 / ISO8601形式で記述する
* gRPCはptypesのgoogle.protobuf.Timestampを用いる
    * 日付のみの型は無いのでomegaで提供する
* 時刻の場合は必ずタイムゾーン情報を付与すること
* 利用者にどのタイムゾーンで見せるかを決めるのはクライアント側の責務とする
ページング
* DB負荷の観点から要件的にpage_tokenによるページングが可能なら推奨
    * Google Cloud  APIのベストプラクティスが参考になる
    * こちらではpage_tokenにprotobuf+Base64を推奨しているが、よほどの高頻度で使われるものでなければJSONでも十分
* 要素数指定による場合は、page/per_page or limit/offsetの形式があるが、limit/offsetの方を推奨する
* 必要ないのであれば総数は返さない
    * こちらもDB負荷が重くなりがちなので
GraphQL
* とりあえず現状は非推奨
    * N+1的にパフォーマンスの課題になりやすい
    * 部分的なエラー処理が根本的に難しい
    * トップページなどの一度にたくさん取得したいケースは、ページ毎にBFF的なendpointを作るほうがよさそう
 
目的
freeeサービスのAPIにおいてページングをどのように設計することを推奨するのかを定めることを目的とします。
特に記述がない限り、Public/Private/Internal/Adminの全てのAPIに対しての推奨事項とします。
フィードバック方法
基本的にはこのドキュメントへのコメントでお願いします。反応がない場合は #team-lego-dev にて @api_engs をメンションしてください。
要求レベルの定義
他のドキュメントと同様に、RFC 2119 に倣い要求レベルのキーワードとして "MUST", "SHOULD", "MAY" を利用します。
* MUST
    * その定義が仕様の絶対条件であることを示します。
* SHOULD
    * 状況次第では適用しない理由が存在することを意味します。しかし、その意味を慎重に検討した上で、別の選択肢を考える必要があります。
* MAY
    * この項目はオプションであり、必須対応ではありません。
方針サマリ
* 汎用的なAPIにおけるページングはカーソル方式を推奨とします。ランダムアクセスが必要なAPIに関してはoffset方式を推奨とします。
* カーソル方式を利用する際には共通ライブラリの利用を推奨とします。未対応の言語やDBでカーソル方式を採用する場合にはまずAPI基盤に相談を投げてください。
方針
[SHOULD] ページングはカーソル方式を優先して利用する
この記事下部の関連知識にもある通り、offset方式では後方の要素を取得しようとするほど性能劣化が懸念されますが、カーソル方式は常に一定のパフォーマンスを期待できます。よってカーソル方式を採用できるユースケースにおいては優先して利用することを推奨とします。
ただしランダムアクセスが必要な場合にはカーソル方式では対応できないため、offset方式を利用してください。
どちらもサポートするケース
単純なページ送りとランダムアクセスの双方のユースケースが考えられるようなAPIでは、ひとつのAPIでどちらもサポートすることが可能です。
どちらもサポートする際の推奨動作は以下とします。
[SHOULD] page_tokenとoffsetが同時に指定された場合にはエラーにする
どちらも指定された場合の動作としては
* page_tokenを優先する
* offsetを優先する
* page_tokenの位置からoffsetを適用する
* エラーにする
が考えられます。
間違えて指定したわけではなくどちらも指定した場合には「page_tokenの位置からoffsetを適用する」を期待していると解釈できます。MySQL系ではこれを実現できますが、Elasticsearch系ではpage_token(search_after)とoffset(from)を同時に指定した際にエラーになります。よって利用する技術によって差が出ないよう一律でエラーを推奨とします。
[SHOULD] 最大取得件数を指定するパラメータは共通してpage_sizeとする
どちらもサポートする際には「page_token/page_size」「page_token/offset」として最大取得件数の指定はpage_sizeに統一することを推奨します。これはクライアントがケースによってpage_tokenとoffsetを使い分ける際に最大取得件数は同一パラメータの方が楽だと考えたためです。
なお既存でlimit/offsetを採用しているAPIにカーソル方式対応を入れる場合にはlimitに対応しないと破壊的変更になってしまうため、alias的にlimitを利用する設計を許容します。
その場合にもpage_sizeのパラメータは用意し、こちらを優先して認識するようにしてください。
例
* page_size=50&page_token=xxx
    * カーソル方式, 最大件数: 50
* page_size=20&offset=10
    * offset方式, 最大件数: 20
* page_size=10&offset=10
    * offset方式, 最大件数: 10
* page_token=xxx&offset=10
    * エラー
[SHOULD] カーソル方式のAPIパラメータはpage_token/page_sizeとnext_page_tokenとする
カーソル方式ではインターフェイスに特定のパラメータが必要になります。
社内各プロダクトでなるべく形式を揃えるため、それぞれの役割を持つパラメータの命名は以下を推奨とします。
リクエストパラメータ
* 利用するトークン : page_token
* 最大取得件数        : page_size
レスポンスパラメータ
* 次ページを取得する際に利用するトークン : next_page_token
[SHOULD] offset方式のAPIパラメータはpage_size/offsetとする
offset方式では主なリクエストパラメータの命名として
* page/per_page
* limit/offset
* page_size/offset
の形式があります。
今後は「最大取得件数を指定するパラメータは共通してpage_sizeとする」の説明にある通り取得件数の命名を統一したいため、page_size/offsetの命名を推奨とします。
[SHOULD] カーソル方式を利用する際には共通ライブラリを利用する
カーソル方式によるページングではパフォーマンスの向上が期待できる一方で、取得の際のクエリの組み立てが複雑になります。そこでAPI基盤からカーソル方式によるページングのためのライブラリを提供しています。ライブラリを利用することでtoken形式の統一化や挙動の安定を目指しています。
現在の提供状況は以下の通りです。未提供の領域で利用したい場合にはライブラリとしての新規実装の検討ができる可能性もあるので、API基盤に相談してください。
Ruby(ActiveRecord) + MySQL
現在CFO-Alphaで試験的に運用中です。将来的にはomega-rubyに移植予定です。
CFO-Alpha以外でカーソル方式の採用をする場合にはAPI基盤に相談してください。
https://github.com/C-FO/CFO-Alpha/tree/develop/lib/cursor_paginationチームが Github のより高度なプレビューを表示しています接続 
Ruby + Elasticsearch系
未対応
Go + MySQL
未対応
Go + Elasticsearch系
未対応
関連知識
カーソル形式によるページネーションとは？
tokenと呼ばれる目印(カーソル)を利用してページングを実現する方法です。tokenには前回どこまで取得したかを判別できる情報が含まれており、これによって次データの取得位置を判別できます。
 
例として経費申請(expense_applications)を金額の大きい順になるべく昔に申請されたものから表示したいとします。
以下のようなレコードが存在しているとします。(分かりやすくソート済みの状態で並べておきます)



id	application_date	amount
5	2025-09-25	5000
1	2025-01-10	4000
3	2025-07-19	3000
4	2025-08-11	3000
6	2025-09-30	2000
7	2025-09-30	2000
2	2025-05-20	1000


まずは1ページ目としてtoken無しでデータを取得します。


SELECT * FROM expense_applications ORDER BY amount DESC, application_date ASC, id ASC LIMIT 5;
取得できるデータは以下になります。



id	application_date	amount
5	2025-09-25	5000
1	2025-01-10	4000
3	2025-07-19	3000
4	2025-08-11	3000
6	2025-09-30	2000


取得したデータのうち最後の要素の値を利用してtokenを作成します。共通ライブラリ(MySQL向け)ではtokenの内容は以下のようになります。


{
  order: [
    {
      "column_name": "amount",
      "direction": "DESC",
      "value": 2000
    },
    {
      "column_name": "application_date",
      "direction": "ASC",
      "value": "2025-09-30"
    },
    {
      "column_name": "id",
      "direction": "ASC",
      "value": 6
    },
  ]
}
次のリクエストでは上記のtokenを利用してデータを取得します。今回のケースでは以下のようなクエリになります。


SELECT *
FROM expense_applications
WHERE amount < 2000
OR (amount=2000 AND application_date>"2025-09-30")
OR (amount=2000 AND application_date="2025-09-30" AND id>6)
ORDER BY amount DESC, application_date ASC, id ASC
LIMIT 5;  
これによって正しく次のページが取得できます。



id	application_date	amount
7	2025-09-30	2000
2	2025-05-20	1000


カーソル方式とoffset方式のメリット/デメリット



	メリット	デメリット
カーソル方式	•	後方の(深い)要素の取得でも一定のパフォーマンスを期待できる
	•	データの重複や欠損が起こりづらい	•	実装が複雑
	◦	共通ライブラリで解決を目指しています
	•	ランダムアクセスができない
offset方式	•	直感的で実装が簡単	•	後方の(深い)要素の取得ほどパフォーマンスが劣化する


パフォーマンス
カーソル方式を推奨する最大の理由です。
カーソル方式ではWHERE句による検索によって、INDEXを利用しながらLIMIT分のレコードを読むだけで済むためどの深さの要素でも一定のパフォーマンスで取得ができます。
一方でoffset方式では深い要素を取得しようとするほど多くのレコードを読む必要が出てしまいます。最初の100件の取得では100行を読めば良いですが、100ページ目を取得するには10000行を読む必要がありパフォーマンスが劣化していきます。
データの重複や欠損の頻度
カーソル形式では特定のレコードを目印として次のデータを取得しにいくため、レコードが追加されたり削除されたりしても重複や欠損が起こりづらいです。
一方でoffset方式では1ページ目として取得したデータに新たなレコードが追加されたり削除されたりすると2ページ目の取得時に重複/欠損が発生してしまします。
ランダムアクセスの可否
ランダムアクセスとは「次ページ」「前ページ」への移動だけではなく、「5ページ目」「7ページ目」と任意のページにジャンプするようなアクセス方法です。
offset方式では任意の件数をずらして取得できるため、1ページあたりの件数からoffsetを計算することでランダムアクセスが可能です。
一方でカーソル方式ではページを移動するためには必ず目印となるtokenが必要となるため、5ページ目に移動するためには1ページ目から順に5回リクエストを行う必要があります。
実装難度
offset方式ではMySQLでもElasticsearch系でも簡単にクエリで表現ができるため、実装が容易です。
一方でカーソル方式ではMySQLの実装において上手くWHERE句を組み立てる必要があるため実装の複雑度がやや上がります。クエリは カーソル形式によるページネーションとは？ にある通りです。
ソート条件が全て一定方向であれば行コンストラクタによって簡略化ができますが、行コンストラクタとAND/OR式を混在させるとINDEXを上手く利用してくれないケースがあるため、company_idによる絞り込みを入れることが多いfreeeのプロダクトでは不向きになります。
カーソル方式でもElasticsearch系ではsearch_afterを利用するだけで良いので実装難度はあまり上がりません。
 
オーナーチーム
#team-service_infra


このドキュメントは？
freee API標準の補助ドキュメントとして、付随するインフラ周りの標準を定める。
ここで定めるインフラ標準のスコープ
* APIサーバーの構成
* ネットワーク構成
* API種別との関係
* （もう少し追加するかも）
APIサーバーの構成
freeeのサービスはk8sクラスタ上にいくつかのDeploymentに分けてサーバーを配置している。これらのDeploymentの粒度はオートスケーリングの粒度であり、大量のリクエストや重いリクエストが来た時に、過負荷による他のリクエストへの影響を遮断したい場合に切り分ける。
標準としては以下の単位でAPIサーバー向けのDeploymentを切る。なお、サーバー以外の用途のDeploymentは別途存在する。



Deployment	用途
web	freee製アプリによる外部からのリクエストを処理する（Private API用）
api	外部アプリによるリクエストを処理する（主にPublic API用）
api-internal	freee内部からのリクエストを処理する（Internal API / Admin API用）
admin	admin画面起点のリクエストを処理する（主にAdmin API / Admin Web API用）


必ずしもAPI種別でどのサーバーで処理するかが決まるわけではない。
ネットワーク構成
外部からのアクセス向けと内部からのアクセス向けてネットワーク経路を変えるため、ドメインを分ける。



ドメイン	用途
.freee.co.jp	外部からのアクセス
.freee-internal.com	内部からのアクセス


API種別との関係
APIコール時に経由するネットワークはAPI種別により定まるが、どのDeploymentで処理するかはある程度は決まるが用途に応じて変わる部分もある。以下に想定されている組み合わせを列挙する。




アクセス元	Deployment	API種別	用途
外部	web	Private API	freee製アプリからのアクセス
外部	api	Public API	3rd-party製アプリからのアクセス
外部	admin	Admin Web API	admin画面からのアクセス
外部	admin	Private API	become中のプロダクト画面からのアクセス
内部	api	Internal API	Public APIの処理中の他サービスへのアクセス
内部	api-internal	Internal API	一般的なプロダクト機能の処理中の他サービスへのアクセス
内部	api-internal	Admin API	PubSubのようなシステム機能からの各サービスへのアクセス
内部	admin	Admin API	admin機能からの各サービスへのアクセス
内部	admin	Internal API	become中 or admin機能でassume中の各サービスへのアクセス


Admin機能の処理が完全にadmin Deploymentに閉じるのが特徴。API接続先のベースURLを環境変数で指定する形にして、admin Deployment用の環境変数ではadmin向けのインターナルドメインを指定する。



OpenAPI標準
Owner: kikanchos + Yutaro Doi
Author: Yui Terashima
Last Update: 2023/08/22
Status: living document
Ticket: 
Translation Ticket: xxx
Original: https://docs.google.com/document/d/1QhUz7X2_IvnpUQBRIUZe4OWs5V4NSEGT2qA2fj0fsyk/edit#heading=h.4hskynqrqq4jチームの 80% が Google Drive のプレビューを表示しています接続 
このドキュメントは？
freeeにおけるOpenAPIの標準実装について提案する。
OpenAPI定義ファイルの構成/記述スタイルや、周辺ツールの使い方など、OpenAPIを利用した実装にまつわる内容がスコープとなる。
対象
このドキュメントで提案する標準の適用対象は以下とする。
* Railsで実装されたサービス
* Public API 以外の API 種別(=Private/Internal/Admin API)
 
「Railsで実装されたサービス」の理由:
freee では Rails か Go でバックエンドサービスが実装されることが多いが、Go の場合 OpenAPI ではなく gRPC によってサービスAPIが提供されるケースがほとんどであるため。
 
「Public API 以外の API 種別」の理由:
Public APIについては既に一定の標準化がPublic APIチーム主導でなされており、性質も他3つとは大きく異なる(Publicは外部向け、他3つは内部向け)ものなので分けて考えたいため。
フィードバック方法
OpenAPI標準実装検討にて議論してまとめたものをLiving Standardとして公開し随時改善中。
詳細を決めきれていない部分もあるが、方向性に対する意見やこの辺どうなっていくのか知りたいなどドキュメントへのコメントやSlackでの議論などでフィードバックをしてもらえるとありがたい。
議論が白熱する場合は同期的に mtg を開き、OpenAPI標準実装検討の方に議事録として議論過程をまとめた上で決定事項を本ドキュメントに反映するものとする。
要求レベルの定義
このドキュメントでは、RFC 2119 に倣い要求レベルのキーワードとして "MUST", "SHOULD", "MAY" を使う。
* MUST
    * その定義が仕様の絶対条件であることを示します。
* SHOULD
    * 状況次第では適用しない理由が存在することを意味します。しかし、その意味を慎重に検討した上で、別の選択肢を考える必要があります。
* MAY
    * この項目はオプションであり、必須対応ではありません。
詳細
OpenAPI定義ファイルの構成
ファイル分割
* Rails Controller の単位で定義ファイルを分割する(MUST)
    * controller に複数の action がある場合、定義ファイルにも複数の action に相当する endpoint を定義する
* ファイル名は、Controller の拡張子を .openapi.yml に変えた形とする(MUST)
* infoなどの共通部分は必要なら base.openapi.yml に切り出す(MAY)
 
良い例
deals_controller.openapi.yml
 
悪い例
* deals_controller_show.openapi.yml
* deals_controller.openapi.yaml
* deals_controller.swagger.yml
 
例外
base.openapi.yml
 
理由
* シンプルで考えることが少ないので分割を機械的に行え、導入ハードルが低いため
* OpenAPI定義が大きくなりすぎるのでちゃんと Controller 分割しようというモチベーションに繋がり、実装の肥大化抑制にもいい影響を与えうるため
 
定義の共有
* Schema Object や Parameter Object などの定義の共有は、同一定義ファイル内でのみ許容する(SHOULD)
    * つまり、同一Controller内に閉じる形でのみ定義を共有できる
 
良い例
deals_controller.openapi.yml と expense_applications_controller.openapi.yml があるとして、それぞれ以下のようになっている
 
deals_controller.openapi.yml 


...
paths:
  "/api/p/deals/{id}":
    get:
      ...
      responses: 
        "200":
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/DealsShowResponse'
    put:
      ...
      responses: 
        "200":
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/DealsUpdateResponse'
components:
  schemas:
    DealsShowResponse:
      type: object
      properties:
        deal:
          $ref: '#/components/schemas/Deal'
    DealsUpdateResponse:
      type: object
      properties:
        deal:
          $ref: '#/components/schemas/Deal'
    Deal:
      type: object
      properties:
        id:
          ...
 
expense_applications_controller.openapi.yml


...
paths:
  "/api/p/expense_applications/{id}":
    get:
      ...
      responses: 
        "200":
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/ExpenseApplicationsShowResponse'
components:
  schemas:
    ExpenseApplicationsShowResponse:
      type: object
      properties:
        expense_application:
          $ref: '#/components/schemas/ExpenseApplication'
    ExpenseApplication:
      type: object
      properties:
        ...
        deal:
          $ref: '#/components/schemas/ExpenseApplicationsShowDeal'
    ExpenseApplicationsShowDeal:
      type: object
      properties:
        id:
          ...
* 同一定義ファイル内では components を用いて Schema Object が共有されている
    * deals_controller.openapi.yml  にて、DealsShowResponse と 、DealsUpdateResponse がどちらも components/schemas/Deal を $ref で参照している
* 似たような構造でも、定義ファイルをまたいだ Schema Object の共有はされていない
    * 以下のように、それぞれの定義ファイルにおいて Deal のデータ構造を示す schema が別々に定義されている
        * components/schemas/Deal
        * components/schemas/ExpenseApplicationsShowDeal
 
悪い例
良い例にある expense_applications_controller.openapi.yml において、components/schemas/ExpenseApplication から deals_controller.openapi.yml の components/schemas/Deal を $ref で参照する
 
例外
* どうしてもファイルをまたいで定義を共有したい場合は、shared/components.yml というファイルを作り、そこに共通のものを配置する
    * ただし、API種別をまたぐ共有は絶対禁止
    * sharedディレクトリはAPI種別を示すディレクトリの直下に配置する
        * 例) app/controllers/api/private/shared/components.yml
    * 基本はリソースとは関係のない真に共通なもの(共通エラーレスポンスやページネーション用parameterの定義など)の定義を置くに留める
* 外部サービスから取得するマスタ系リソース(取引先や部門とか)など、どうしても共通化しないと辛そうなものはそれぞれ専用のディレクトリを更に掘り、その中に共通化したい定義を配置する
    * 例) app/controllers/api/private/shared/partner/components.yml 
    * ここにどんどんディレクトリが増えていくことは過度な共通化に繋がる可能性が高いのでなるべく避けること
 
理由
* APIのスキーマについては、これまで過度な共通化が害を産んできたケースが多いため
    * 共通化された部分に変更を入れようとして思わぬところに不具合が出てしまう問題が、特に private API と mobile API のスキーマやSerializer実装を共有していたところなどで散見されてきた
* freeeのRailsサービスにはリソース指向ではなくBFF的なAPIも多く、それらで変にスキーマを共通化してしまうといくつか問題が生じるため
    * 例えば、その画面固有で必要なプロパティを追加するとか逆に削除するとかのチューニングがやりにくくなったりする
        * 特に一覧画面用APIとかはパフォーマンス的に詳細画面用APIと同じ定義を共有すべきではないことが多い
ディレクトリ構成
* 定義ファイルを対応する Controller の実装ファイルと同じパスに配置する(MUST)
* ファイルをまたいで共有したい定義を配置する shared ディレクトリを、API種別を示すディレクトリの直下に配置する(MUST)
* base.openapi.yml を作成する場合は、API種別を示すディレクトリの直下に配置する(MUST)
 
良い例


/app
  `- /controllers
    `- /api
      |- /private
      | |- /shared
      | |   |- components.yml
      | |   `- /partner
      | |       `- components.yml
      | |
      | |- deals_controller.rb
      | `- deals_controller.openapi.yml
      |
      `- /internal
        |- /shared
        |   |- components.yml
        |   `- /partner
        |       `- components.yml
        |
        |- base.openapi.yml
        |- deals_controller.rb
        `- deals_controller.openapi.yml
 
悪い例


/app
  `- /controllers
    `- /api
      |- /private
      | `- deals_controller.rb
      |
      `- /internal
        `- deals_controller.rb
/docs
  `- /api_schemas
       `- /api
      |- /private
      | `- deals_controller.openapi.yml    
      |
      `- /internal
         |- base.openapi.yml
         `- deals_controller.openapi.yml
実装とは別のdocsディレクトリを作ってそこに定義ファイルを入れている


/app
  `- /controllers
    `- /api
      |-/shared
      | |-components.yml
      | `/partner
      |    `- components.yml
      |
      |- /private
      | |- deals_controller.rb
      | `- deals_controller.openapi.yml
      |
      `- /internal
        |- base.openapi.yml
        |- deals_controller.rb
        `- deals_controller.openapi.yml
app/controllers/api の下に shared を切っている
 
理由
* 実装ファイルのすぐ隣に定義ファイルがあるので、開発者が定義ファイルを見つけやすいため
* まだOpenAPI定義のない Controller がわかりやすいので、後から定義を追加していく時に漏れをチェックしやすいため
 
分割した定義ファイルの連結
※ 連結といっても、$refでまとめるだけの論理的な連結とその$refを展開までする物理的な連結があり、それぞれ「論理連結」,「物理連結」と呼ぶことにする
 
分割した定義ファイルはAPI種別ごとに物理連結ファイルを作り、docs/api/{api_type}/openapi.yml という名前で commit する(MUST)
 
* 一般的に物理連結ファイルを作る前に論理連結ファイルが必要だが、その作成のために freee-bootstrap に用意された便利スクリプトを自身の repository にコピーして使う(MAY)
    * https://github.com/C-FO/freee-bootstrap/blob/master/templates/rails/scripts/merge-openapi-schema-files.rb.tmplチームが Github のより高度なプレビューを表示しています接続 
* 物理連結ファイルを作るために必要になった論理連結ファイルは commit を行わない(SHOULD)
 
* 物理連結は以下のように行う(MUST)
    * ツールは openapi-generator を使い、オプションとして -g openapi-yaml を指定して論理連結ファイルを食わせる
    * openapitools/openapi-generator-cli という docker image を使い、container 経由で実行する
    * コマンドの起点は Makefile とし、{api_type}-api.generate-schema という target 名で起動されるようにする
* 上記のような物理連結を行うために、freee-bootstrapに用意された一連の便利スクリプトを自身の repository にコピーして使う(MAY)
    * https://github.com/C-FO/freee-bootstrap/blob/e68ce03683e4e45709126c6bf2a59a411cc2a2f5/templates/rails/Makefile.tmpl#L59-L69 
 
* 物理連結の実行漏れがないかCIでチェックを行う(MUST)
* CIによるチェックは shared-actions に用意された Reusable workflow を利用して行う(MAY)
    * https://github.com/C-FO/shared-actions/blob/main/docs/check-generating-openapi-merged-schema.md 
 
良い例
docs/api/private/openapi.yml として、以下のようなファイルが repository に commit されている


openapi: 3.0.3
...
paths:
  /api/p/deals:
    get:
      description: |-
        ...
      parameters:
        ...
 
悪い例
docs/api/private/openapi.yml として、以下のようなファイルが repository に commit されている


openapi: 3.0.3
...
paths:
  /api/p/deals:
    $ref: deals_controller.openapi.yml#/paths/~1api~1p~1deals
 ...
 
理由
* OpenAPI周辺ツールの一部に、物理連結されたファイルでないとうまく動かないものがあるため
* API種別ごとにClient生成などのユースケースが違うので、それぞれの定義ファイルは別々にしておきたいため
OpenAPIドキュメント生成
生成方法
* OpenAPI定義ファイルから以下のようにドキュメント生成を行う(MUST)
    * ツールは redoc を使い、redocly build-docs コマンドに物理連結ファイルを食わせる
    * redocly/cli というdocker image を使い、container 経由で実行する
    * redocly build-docs {target_schema_file_path} 一発で静的HTMLを生成可能
* 後述のように別途リモートサーバにホスティングするので、生成したドキュメントを repository に commit するかどうかは自由とする
 
理由
* redoc はコマンド一発で全部入りの静的HTMLとしてドキュメントを生成してくれるので、取り回しがしやすいため
* 生成物が静的HTMLなので、特定のツールを手元に用意せずともブラウザだけで見ることができるため
ホスティング
* ドキュメントを専用の S3 bucket にホスティングし、社内VPN経由でのみアクセスできるように公開する(MUST)
    * [Eng] OpenAPIのドキュメントをS3 + CloudFront でホスティングする 会計バックエンド委員会 FY24Q1 で用意された基盤に乗っかる
    * https://api-doc.freee-internal.com/ 
    * (参考) [Kibela] freee 社内の OpenAPI ドキュメントが閲覧できるポータルサイトをオープンしました！ ~ リポジトリへの導入について ~  
 
* ホスティングしたドキュメントのURLは以下のように命名する(MUST)
    * api-doc.freee-internal.com/{repository_name}/{api_category}
* api-doc.freee-internal.com/index.html には各ドキュメントへのリンク集を表示する(MUST)
    * freee-sso-developer で AWS にログインし、freee-internal-openapi-doc という S3 バケットを開く
    * index.html をダウンロード、追加したいプロダクトとURLを追加してアップロード
 
* shared-actions に用意された generate_and_publish_openapi_doc_usable という Reusable workflow を利用して、ドキュメント生成 & S3 bucket への upload を行う(MUST)
    * https://github.com/C-FO/shared-actions/blob/main/docs/generate-and-publish-openapi-doc.md 
 
良い例
api-doc.freee-internal.com/cfo-alpha/private というURLにアクセスすると、cfo-alpha の private API についてのOpenAPI定義ドキュメントを見ることができる
 
理由
* VPNを通さないと見れないようにすることで、社内の Eng/PdM/PD のみが見れるようなアクセス制限をかけることができるため
    * PdM/PDもAPI仕様見て議論できるようになって欲しいので閲覧を許可したいが、Githubアカウントを個別に発行するのは手間がかかるので GithubPages ではなく S3 を使ってアクセス制御する
* repositoryごと、API種別ごとのAPIドキュメントを探しやすいため
    * URL の path により表現されているので予測しやすい
 
バージョニング
* OpenAPIでは明示的なバージョン管理は行わない(MUST)
    * 従って、定義ファイルの info セクションの version フィールドには、適当な固定値(v1)を入れておく
 
* API定義の変更を行う際には常に前方/後方互換性の維持を意識する(SHOULD)
    * Public APIのポリシーと基本的には同じ
    * どうしても破壊的な変更を行いたい場合は、なるべく安全になるよう段階的にschemaを変更していくこと
 
例.)
* request body の property 名をrenameする場合、以下のように段階的に実施する
    * request body の schema に rename 先の名前で optional な property を追加
    * rename 前の property を受け取った時の挙動は維持しつつ、追加した property でも受け取れるようにサーバー側実装コードを修正
    * サーバー側をデプロイ
    * 追加した property の方を使うようにクライアント側実装コードを修正
    * クライアント側をデプロイ
    * rename 前の property を schema から削除
    * rename 前の property を受け取った時のロジックをサーバー側実装コードから削除
    * サーバー側をデプロイ
 
なお、gRPCを使う方が互換性の維持は楽(フィールド名を変えても、フィールド番号を変えなければクラサバそれぞれでよしなに serialize/deserialize してくれるので互換性に問題がないなど) なので、特に InternalAPI の場合は OpenAPI よりもそちらの使用を検討すること
 
 
理由
* バージョン管理を行うとしたら、常に古いバージョンのAPI実装を本番で動かし続けなければならなくなる
    * 最低でも semantic versioning でいうところの major version については古いものもサポートする必要が出てきてしまう
* しかし、API定義の破壊的変更の頻度が比較的高い(major version が上がる頻度が高い)private/internal/admin API でそれをやるのは厳しすぎるため
 
OpenAPI Client生成
前提
* 以下を主な対象として考える
    * Web Frontend
        * Typescript
            * flowはcfo-alphaすら脱却し始めてるので考えない
    * Web Backend
        * Ruby
        * Go
    * Mobile
        * Swift
        * Kotlin
* BackendとFrontend(Web Frontend/Mobile)が別リポジトリの場合だろうと、Frontend は Private API のみを叩く想定
    * freee API標準 サービス基盤 FY22Q3を参照
        * 別プロダクトのFrontend => api/:module/ を叩く
        * 同プロダクトの別Frontend(マイクロフロントエンド) => api/p/ を叩く
    * Frontendから private API 以外を叩くことはない
        * 逆に言うと Internal/Admin API は Backend からのみ叩かれる
        * 現状そうはなってない(Private API 叩いてる Backend とかいる)けど、あくまでこの理想を前提として考える
* 上記から、Private API と Internal/Admin API はそれぞれ別に方針を考える
    * Private は Web Frontend/Mobile, Internal/Admin は Web Backend がターゲットとなり、それぞれで考慮したいポイントが異なるため
 
Private API
Clientコード生成用ツール
* Web Frontend 向けは、openapitools/openapi-generator-cli という docker image を使った container 経由で、以下のような構成で生成を行う(MAY)
    * Generator: typescript-fetch
    * Template: freee-bootstrap に用意されたfreee推奨カスタムテンプレートを自身の repository にコピーし、それを指定する
        * TODO: 用意できたらリンク貼る
* Mobile 向けは、こちらのドキュメントでは特に標準を規定しない
 
理由
* Web Frontend 向けについて
    * 基本は repository ごとに自由に作ってくれて構わないが、詳しくない人でも導入のハードルが高くならないように推奨構成を用意した
* Mobile 向けについて
    * モバイルチームの事情によるところが大きいので、こちらのドキュメントとしては特に標準も推奨も言及しないことにした
 
Clientコード生成方法
Server側とClient側が同一repositoryにある
* 生成コマンドの起点は Makefile とし、private-api.generate-client という target 名で起動されるようにする(MUST)
* docs/api/{api_type}/openapi.yml を元にClientコードを生成する(MUST)
* (仮)生成したClientコードは front/packages/@api/private/ に配置する(MAY)
    * freee推奨カスタムテンプレートの完成を待って確定する
* 上記のようなコード生成を行うために、freee-bootstrapに用意された一連の便利スクリプトを自身の repository にコピーして使う(MAY)
    * TODO: 用意できたらリンク貼る
 
* コード生成の実行漏れがないかCIでチェックを行う(MUST)
* CIによるチェックは shared-actions に用意された Reusable workflow を利用して行う(MAY)
    * TODO:  用意できたらリンク貼る
 
Server側とClient側が別repositoryにある
Server側の repository にある docs/api/{api_type}/openapi.yml を元に、Client側のrepositoryでClientコードを生成する(MUST)
 
* 生成タイミングや具体的な生成の仕組みはTBD(以下のいずれかになりそう)
    * タイミング
        * 手動
            * 任意のタイミングでトリガできるようにする
                * Client側ですべてを柔軟に制御できるので事故はおきにくそう
        * 定時実行
            *  cron とかで定時実行
        * Server側の openapi.yml が更新されたタイミング
            * Server側の openapi.yml の更新が main ブランチに入ったらトリガ(≒本番デプロイされたら)
    * 生成の仕組み
        * pull型
            * Client側の repository からServer側の repository にある docs/api/{api_type}/openapi.yml を pull し、それを元にClientコードを生成する
                * 例:　https://github.com/C-FO/mob-core-invoice/blob/a86505e5bf56dea62dacbfca22d9b68f26d6257d/fastlane/Fastfile#L428-L438 
        * push型
            * Server側の repository からワールドツアー的に各Client側の repository に、それぞれが保持する openapi.yml を最新化するPRを作って回る
                * https://github.com/C-FO/omega-ruby/blob/master/.github/workflows/world_tour.yml のように、github actions である程度は自動化できる
                *  GitHub Apps で bot を作りそれにやらせる手もある
 
理由
* Clientごとに自由に生成ツールや生成のやり方を選択できる柔軟性を確保したかったため
* Server:Client=1:Nの場合、Server側で全てのClientに関心をもつのが辛いため
    * generator とか template を変えたい、とかなった時に Client を触る開発者が Server 側をいじりにいかないといけなくなって面倒くさい
 
Internal/Admin API
Clientコード生成用ツール
* Ruby 向けは、openapitools/openapi-generator-cli という docker image を使った container 経由で、以下のような構成で生成を行う(MUST)
    * Generator: ruby(--library faraday オプションを付ける)
    * Template: 現時点では特に指定なしでOK
* Go 向けは、openapitools/openapi-generator-cli という docker image を使った container 経由で、以下のような構成で生成を行う(MUST)
    * Generator: go
    * Template: 現時点では特に指定なしでOK
 
理由
Internal/Admin APIの場合、今後 Client に共通で組み込みたい機能(標準ErrorCodeの取り回しとか)が出てくる可能性が高いので生成ツールを標準化しておくメリットが大きいため
Clientコード生成方法
* 具体的な生成方法や生成のタイミングはTBD(以下のいずれかになりそう)
    * Server側で手動でコマンドを打って生成
    * 各repositoryのopenapi.ymlを集約するモノレポを用意し、そちらに最新のopenapi.ymlがpushされたのをトリガーに自動生成
* (仮)生成したClientコードは modules/{repository-name}-client/{internal/admin}/ に配置する
    * 上記の具体的な生成方法が決まったら fix する
 
* Ruby向けに生成した Client コードは gem としてGitHub packagesにて配布する(MUST)
* gem の配布は、shared-actions に用意された Reusable workflow を利用して行う(MUST)
    * https://github.com/C-FO/shared-actions/blob/main/.github/workflows/gem-publish.yaml 
* Go向けに生成した Client コードは Go modules として配布する(MUST)
 
理由
Internal/Admin APIの場合は Client の柔軟性があまり必要なく、むしろ処理を統制したいモチベーションが高いので Server側で生成することにアドバンテージがあるため
 
OpenAPI Ruby Clientの利用
* Open API Ruby Clientを利用するサービスでは config/initializers 以下で omega-rubyの Omega::API.prepare でセットアップする (MUST)
* また上記は Omega::FaradayMiddleware に依存するので、こちらも config/initializers/omega.rb でセットアップする (MUST)


Omega::API.prepare(FreeeInvoiceClient, ENV["INVOICE_INTERNAL_URL"], **opts) do |config|
  ... # 必要ならば追加のFaraday設定
end
# config/initalizers/oemga.rb
Omega::FaradayMiddleware.setup
理由
* Omega::FaradayMiddleware::InternalHandlerにより以下の機能を組み込むため
    * Contextの引き回し
    * Session (InternalSession) の引き回し
    * エラー時(4xx, 5xx)に標準エラーを例外オブジェクトに引き回し & raise
    * 開発環境で簡易ログ
* 設定方法を一元化しておくことで、今後標準的機能を追加する際にoemga-rubyの更新のみで対応できるため
Serverへのcommittee gemの導入
committee gem を使い、OpenAPI定義と実装の乖離がないか自動でチェックを行う(MUST)
 
理由
* OpenAPI定義と実際の実装が乖離していないかを人力レビューによりチェックするのは限界があるため
* committee を選んだのは、他によさそうなツールが特に見つからなかったし社内での利用実績も豊富であるため
 
チェック方法
* committeeが提供するチェック用の Rack Middleware を Rails に差し込み、 run-time check を行う(MUST)
* Rails.env によってチェックの挙動を以下のように変える(SHOULD)
    * RequestValidation
        * 最初から schema 駆動開発を取り入れている場合
            * 全環境で何もしない
        * 途中から schema 駆動開発を取り入れた場合
            * development ではエラーを raise する
            * test では何もしない
            * integration/staging/production ではBugsnag通知する
    * ResponseValidation
        * development/test ではエラーを raiseする
        * integration/staging/production では Bugsnag 通知する
 
理由
* run-time check を行い test 環境ではエラーになるようにすれば、committee-rails を使った request spec でのチェックは不要になるため
* request spec で全ての request/response のパターンを網羅するのは現実的ではないため
    * production環境などでもチェックを行う(ただしユーザー影響出ないようにエラーにはしない)ようにすれば、実際のユーザーからのリクエストを元にある程度網羅性の高いチェックを行える
* RequestValidation の挙動の理由
    * 最初から schema 駆動開発を取り入れている場合
        * 常に schema から生成された client だけが使われていることを期待するので基本的には validation は不要のはず
        * それでもAPI定義に沿わないリクエストが飛んできたとき(例えば postman などによる野良リクエスト)どうするかはアプリケーション側に任せるようにしたい
    * 途中から schema 駆動開発を取り入れた場合
        * schema から client を自動生成するのではなく人力でメンテしている可能性もあり、その場合は client 実装と API 定義がずれることがあるため、それに気づけるようにしたい
        * test環境についてはアプリケーション独自のvalidation(Form層などでのvalidation)と競合するので変わらず committee 側ではチェックしない
 
committee の設定
* committeeの設定ファイルを config/initializers/committee.rb に配置する(SHOULD)
* 設定の中身は以下のようにする(SHOULD)
    * Committee::Middleware::RequestValidation/ResponseValidation どちらも config.middleware.use し、request/response 共にチェックされるようにする
        * development 環境では ReloadableMiddleware で wrap しておくことで定義ファイルを変更したときに自動再読み込みされるようにする
    * 以下のオプションを設定する
        * ignore_error: true
        * error_handler
            * チェック方法 にあるような Rails.env によって挙動の変わるラムダ式を入れる
        * parse_response_by_content_type: true
        * query_hash_key: ‘committee.query_hash'
            * committee v5以降を使う場合はデフォルトでこれなので指定の必要はない
* 上記のような設定を行うために、freee-bootstrapに用意された以下の設定ファイルを自身の repository にコピーして使う(MAY)
    * https://github.com/C-FO/freee-bootstrap/blob/master/templates/rails/config/initializers/committee.rb 
 
lintによるスキーマのバリデーション
redocly/cliによりスキーマの結合時にバリデーションチェックを行う(SHOULD)
 
理由
* Clientコード生成でエラーとなるようなスキーマの不正箇所を事前にチェックするため
* redocly/cliはドキュメント生成で使用しているツールにあわせる
    * 検討経緯: OpenAPIのlint導入検討 会計バックエンド委員会 FY24Q2
 
OpenAPI定義の設計/記述に関する一般的な規約
OpenAPI Specification(OAS)記述ガイドライン eng-kikanchos FY24Q1を参照


Owner	API基盤(lego)
Status	LIVING
Last Update	2026年8月5日
Feedback ch.	#ask-api_first


* 「ドメインAPI」とは
* ドメインAPIが目指すもの
* 全プロダクトのAPIがドメインAPIになったら何が実現できるのか
    * freeeの全操作がAPIから実行できる
    * AIエージェントが自律的に業務を遂行できる
    * 新しいクライアントを既存APIの組み合わせだけで立ち上げられる
    * プロダクト間連携のコストが下がる
    * PublicAPIとして一般公開でき、事業機会が広がる
    * APIが資産として積み上がる
    * ドメイン境界がAPIとして可視化され、アーキテクチャの腐敗を防げる
* 一般公開（PublicAPI）の基準としてのドメインAPI
* 現在地とロードマップ
* Q&A
    * Q1. ドメインAPIを名乗るための基準はどこにありますか？
    * Q2. 一般公開（PublicAPI）したい場合は何を満たせばよいですか？
    * Q3. 基準を一本化したのに、なぜドキュメントは2つのままなのですか？
    * Q4. 再利用可能APIとドメインAPIは、別物として作るのですか？
「ドメインAPI」とは
￼
品質基準とAPI種別の関係性
ドメインAPIとは、特定の業務領域（ドメイン）が管轄する「データ（モデル）」と「ロジック」を、社内外を問わず安全に公開するための唯一のインターフェースである。
具体的な定義は1つだけである。
再利用可能API定義ガイドラインの基準のうち、[MUST] と [SHOULD] をすべて満たしたAPIをドメインAPIと呼ぶ。
基準は再利用可能API側に一本化しており、このドキュメントは基準を持たない。何を満たせばよいかは再利用可能API定義ガイドラインの基準と基準早見チェックリストを参照してほしい。
では、このドキュメントは何のためにあるのか。その水準のAPIが全プロダクトで揃ったときに、freeeが何を実現できるのかを示すためである。個々の基準は一つひとつ見ると細かい作法の話に見えるが、それらが全社で満たされた先には、今とは違う景色がある。
ドメインAPIが目指すもの
基準の背骨になっている設計の4原則（カプセル化 / ビジネス・プリミティブ / 整合性 / 中立性）のうち、[MUST] の段階で担保されるのは整合性と中立性である。ドメインAPIで新たに効いてくるのは、ドメイン境界に踏み込むビジネス・プリミティブと、実装詳細を晒さないカプセル化のほうである。
[MUST] を満たすだけの状態、つまり「どのクライアントからでも叩ける」だけでは、次のような状態がまだ残りうる。
* 1本のAPIが複数ドメインの操作を抱え込んでいる
* レスポンスに自ドメイン外の値を混ぜて返している
* 内部のテーブル構造や歴史的な命名がそのままスキーマに出ている
* 一覧APIのページネーションやレスポンス構造が標準からずれていてハンドリングしづらい
* 破壊的変更が前提の設計で、後方互換を考えずに育ってしまう
これらは1本のAPIだけを見れば「まあ動くから良い」と流せてしまう。しかし数百本のAPIが同じ状態で積み上がると、利用者はAPIごとに作法を学び直すことになり、freee全体としては「APIはあるが使いこなせない」プラットフォームになる。ドメインAPIが正そうとしているのはこの点である。
全プロダクトのAPIがドメインAPIになったら何が実現できるのか
freeeの全操作がAPIから実行できる
「WebからはできるがAPIからはできない」がなくなるのは再利用可能APIの目標だが、ドメインAPIはさらに「業務の単位でできる」ところまで踏み込む。粒度が最小の業務単位に揃っていれば、利用者は自分がやりたい業務をAPIの組み合わせとして素直に表現できる。画面の操作手順を逆算してAPIを探す必要がなくなる。
AIエージェントが自律的に業務を遂行できる
AIエージェントにとって、APIは業務を理解するための語彙そのものである。
* ドメイン境界が正しく、1つのAPIが1つの業務操作に対応していれば、エージェントは「何をすればよいか」を操作の組み合わせとして計画できる
* スキーマに制約や仕様が十分に書かれ、命名が省略されていなければ、ドキュメントを読まずとも意図を推論できる
* バリデーションと整合性がAPI側で完結していれば、エージェントが不整合なデータを作ってしまう事故を防げる
逆に、内部実装が透けた命名や画面都合の粒度が混ざっていると、エージェントは正しい操作を選べない。人間なら「たぶんこれだろう」と補完できるところで、エージェントは間違える。人間向けには許容できていた曖昧さが、そのままコストになる。
新しいクライアントを既存APIの組み合わせだけで立ち上げられる
BFFサーバ、GraphQLのエンドポイント、MCPサーバ。新しいチャネルを作りたくなったとき、裏側がドメインAPIで揃っていれば、既存APIを組み合わせるだけで立ち上げられる。プロダクト側に手を入れる必要がない。
戦略で掲げている「新規UIチャネル立ち上げ期間: 従来比70%削減」（APIファースト戦略のKPI）が現実的になるのは、この状態に到達したときである。
プロダクト間連携のコストが下がる
今までのInternalAPIは、連携したい要件ごとに新しく作られてきた。連携が増えるほどAPIが増え、保守対象が増え、次の連携がさらに重くなる。
ドメインAPIが揃うと、この関係が逆転する。新しい連携要件が来ても、既存のAPIの組み合わせと、必要ならイベントの購読で済むようになる。APIが増えるほど、次の連携が軽くなる。
PublicAPIとして一般公開でき、事業機会が広がる
一般公開の基準はドメインAPIである。つまりドメインAPIが揃うということは、一般公開の候補が揃うということでもある。ビジネス的な戦略で公開するかどうかを制御しているだけで、リリース判断となればapi-hubを使って短期間で実現できる。
パートナー・アライアンス・認定アドバイザー・AI BPOパートナーといった社外のプレイヤーが、freeeを組み込んで自分たちのサービスを作れるようになる。エコシステムとしての広がりは、freee自身が作る機能の数では到達できない範囲まで届く。
APIが資産として積み上がる
破壊的変更を避ける前提で設計されたAPIは、時間が経つほど価値が上がる。利用者が増え、その上に作られたものが増え、それでも壊れないからである。
逆に破壊的変更が前提のAPIは、利用者が増えるほど変更コストが上がり、いずれ変更できなくなるか、変更のたびに全利用者を巻き込む。同じ「APIが増える」でも、積み上がるのか、負債が溜まるのかが分かれる。
ドメイン境界がAPIとして可視化され、アーキテクチャの腐敗を防げる
「単一ドメインに責務が閉じている」を全社で満たすと、ドメイン境界がAPIの形で目に見えるようになる。どこからどこまでが誰の責任範囲なのかが、コードを読まなくても分かる。またドメイン間の依存が整理されることで、マイクロサービス化やDB分離もやりやすくなる。
境界が曖昧なまま育ったシステムは、どこを直せば何が壊れるか分からなくなる。APIとして境界が固定されていれば、内部実装をいくら変えてもインターフェースが守られる。これはfreee Service Architectureが目指してきた姿そのものである。
一般公開（PublicAPI）の基準としてのドメインAPI
PublicAPIとして一般公開するAPIの基準は、このドメインAPIとする。
当初は再利用可能APIとドメインAPIの中間に「PublicAPI公開用の基準」を新設する案もあったが、「再利用可能APIは増やすが、PublicAPIにしていくことは慎重に考える」という方針から、中間基準は設けず、一般公開の裏側にはドメインAPIを必須とすることに決めた。MCP限定やOpenβから一般公開へ昇格させる際の条件にも、これを用いる。（厳しすぎるという声が出たら再考する。）
破壊的変更の観点がここで強く効いてくる。[MUST] のみを満たす段階では「後方互換を壊す変更が頻繁に入る」状態も許容してでも数を揃える側に倒しているが、外向けに公開されたAPIの破壊的変更には、長い準備期間と多くのコミュニケーションコストがかかる。一般公開の基準をドメインAPIと定めている大きな理由がこれである。
実際に公開するスキーマ上ではcompany_idが必須になるなど、PublicAPIとしての運用時には追加の要件もある。この詳細を含むPublicAPI専用の運用ドキュメントは今後整備予定で、本ドキュメントでは「一般公開の裏側はドメインAPIである」という関係のみを示す。
後方互換やバージョニング戦略のドキュメントは作成中。できたら貼る。
PublicAPIのガイドラインは整備中。できたら貼る。
現在地とロードマップ
* 新しく作るAPIは、最初からドメインAPIの水準で設計する。[SHOULD] の多くは設計段階で意識していればほとんど追加コストがかからない一方、後から引き上げようとすると命名やレスポンス構造の変更を伴い、破壊的変更として高くつく。
* 既存のAPIは段階的に移行する。一斉に引き上げることは求めない。機能追加や改修のタイミングで、満たせない [SHOULD] がないか、検討を飛ばしている [MAY] がないかを確認してもらえるとよい。
* 宣言の運用は再利用可能APIと区別しない。現時点では、freee-openapi への掲載による再利用可能APIとしての宣言と別に、ドメインAPIであることを明示する仕組みは存在しない。
* gRPC は現在はスコープ外。再利用可能API／ドメインAPIの区分・計測はHTTP（OpenAPI）を前提としており、gRPCへの適用は検討中である。
Q&A
Q1. ドメインAPIを名乗るための基準はどこにありますか？
再利用可能API定義ガイドラインに一本化しています。そこに記載された基準のうち [MUST] と [SHOULD] をすべて満たせばドメインAPIです。
[MAY] はドメインAPIの要件に含めないため、満たさなくてもドメインAPIを名乗れます。ただし「検討しなくてよい」という意味ではありません。要件次第では「提供しない」ことが正解になりうるため一律には求めませんが、検討そのものは [SHOULD] と同様に必要です。
Q2. 一般公開（PublicAPI）したい場合は何を満たせばよいですか？
ドメインAPIであることが基準です。中間基準は設けず、一般公開の裏側はドメインAPIを必須としています。MCP限定やOpenβから一般公開へ昇格させる条件も同じです。PublicAPI専用の運用ドキュメントは今後整備予定です。
MCP限定やOpenβ、専用公開の場合には必ずしもドメインAPIの基準を満たす必要はなく、再利用可能APIであれば問題ありません。
Q3. 基準を一本化したのに、なぜドキュメントは2つのままなのですか？
役割が違うためです。再利用可能API定義ガイドラインは「何を満たせばよいか」を示す実務のドキュメント、こちらは「その水準が全社で揃うと何が実現できるのか」を示すドキュメントです。
APIを1本作るときに読むのは前者で十分です。こちらは、なぜ [SHOULD] まで満たしてほしいのかに納得できないとき、あるいはチームや組織として投資判断をするときに読んでもらうことを想定しています。
Q4. 再利用可能APIとドメインAPIは、別物として作るのですか？
別物ではありません。同じ基準のどこまでを満たしているか、という到達点の違いです（集合として「再利用可能API ⊃ ドメインAPI」）。詳しくは再利用可能API定義ガイドラインのQ&Aの「Q4. ドメインAPIとは何が違いますか？ 別物として作るのですか？」を参照してください。



Owner	API基盤(lego)
Status	LIVING
Last Update	2026年9月7日
Feedback ch.	#ask-api_first


* 「再利用可能」とは
    * 「社内から汎用的に使える」だけでは足りない
* 今までのInternalAPIと何が違うのか
* 再利用可能APIの立ち位置
    * 品質のグラデーション
    * API種別（Internal / Private / Public / Admin）との関係
    * freee API標準との関係
* 再利用可能APIがInternalAPIである理由
* 設計の4原則
* 運用方針
    * 設計と公開
    * 宣言の方法
    * 利用のルール
* 基準
    * (前提) freee API標準への準拠
    * 要求レベルの定義
    * 基準のカテゴリ
    * 1. 運用体制と品質
    * 2. 技術要件
    * 3. ドメイン設計
    * 4. API設計品質
* 基準早見チェックリスト
    * A. 再利用可能APIとして宣言するために満たすもの（MUST）
    * B. ドメインAPIとして満たすもの（SHOULD）
    * C. ドメインAPIの要件に含めないもの（MAY）
* ケース別実装例
    * 1. API粒度の設計
    * 2. ビジネス・プリミティブ
    * 3. バリデーションの完結性
    * 4. 中立性の保持
    * 5. マスタードメインへの依存
* Q&A
    * Q1. Webの新規機能開発でAPIを追加する場合の基本方針は？
    * Q2. Webから直接InternalAPIを叩けるなら、PrivateAPIとして公開しなくても良いのでは？
    * Q3. MUSTだけ満たして宣言してもよいのですか？ SHOULDはどこまで求められますか？
    * Q4. ドメインAPIとは何が違いますか？ 別物として作るのですか？
    * Q5. 既存の再利用可能APIを、後からドメインAPIに引き上げられますか？
    * Q6. 誰がチェックリストの充足を確認・承認しますか？
    * Q7. 複数ドメインをまたぐ操作を1本のAPIで提供したい要望が強いです。どうすれば？
    * Q8. 他ドメインのマスタ（取引先・部門など）の存在を保証したいときは？
    * Q9. gRPC で提供している InternalAPI も再利用可能API／ドメインAPIにできますか？
    * Q10. 社内の複数プロダクトから汎用的に使われているInternalAPIなら、再利用可能APIとして宣言できますか？
「再利用可能」とは
再利用可能なAPIとは、特定のクライアントにのみ特化せず、どのクライアントから叩かれても使える前提で提供されているAPIを指す。
「APIはそもそもこれが一般的ではないのか」と思うかもしれないが、こうなっていないのが現状である。
* クライアント側のバリデーションやデフォルト値の補完が前提になっている
* 機能を追加するたびに、後方互換性を担保できない変更が頻繁に入る
* 内部のテーブル構造がそのまま露出している
* 特定サービスとの連携用に独自ロジックが分岐しており、要件も狭く、他サービスのニーズには応えられない
APIを再利用可能な形で整備し直せば、やりたい操作をすぐに実現できる状態に近付く。中長期的には、プロダクト運用の負荷を軽減する効果も見込める。
なぜ今これが重要なのか。freeeを操作するのが「人間」だけでなく「AIエージェント」や「AIによって作られたツール」も主流になってきているからである。これまで人間がブラウザを開いて行っていた操作を、AIが迷わず実行できるだけのAPIが網羅されているか。ここがプラットフォームとしての勝負所になる。社内のFDE、AI BPOパートナー、認定アドバイザー、アライアンスパートナー、特定のプランを契約したエンドユーザーなどに対し、高いスピードでAPIを提供できる体制を構築したい。
「社内から汎用的に使える」だけでは足りない
「再利用可能」という言葉から、「社内のどのプロダクトからも汎用的に呼び出せる」状態を思い浮かべるかもしれない。しかしそれだけでは、ここでいう再利用可能APIには足りない。複数のプロダクトから広く使い回せるAPIであっても、
* 呼び出し元が信頼できる社内システムであること
* 特定の業務フローやイベントを起点にしたときだけ呼ばれること
といった暗黙の前提の上で安全性が成り立っているなら、それは再利用可能APIではない。こうした前提は、PrivateAPIやPublicAPIとして公開された瞬間に消える。呼び出し元を選べなくなった結果、本来システムにしか許されない操作をエンドユーザーが直接実行できてしまえば、不正なデータ操作や事業損失に直結する。
再利用可能APIの安全性は呼び出し元への信頼ではなく、API自身の認証・認可・バリデーションで担保する。「このAPIが明日そのままPrivateAPIやPublicAPIとして公開されても、安全性の問題は起きないか」を判断の目安にしてほしい。(ビジネス的な観点からPublicAPIとしても公開しない、という判断は別軸で存在する。再利用可能APIを必ずしも公開する必要はない。)
なお、呼び出し元を信頼する前提のAPIをInternalAPIとして提供すること自体は、正当なサービス間連携の形であり何も問題ない。そうしたAPIは再利用可能APIとして宣言しない、という区別の話である。また、ここで求めているのはあくまで安全性の前提条件であり、PublicAPIとして正式に一般公開できる品質とは別の話である。公開品質の基準はドメインAPI水準とする。
今までのInternalAPIと何が違うのか
今までのInternalAPIはfreeeのプロダクト間連携のために、個別ユースケースに特化して作成されることが多かった。しかも設計・実装を担当するのは連携したい側のプロダクトチームであることが多く、ドメインやアーキテクチャへの理解不足から、内部ロジックも理想とかけ離れてしまいやすかった。
この状態では同じ機能を使いたくても要件ごとにAPIが増え、利用開始までに毎回大きなコストがかかる。保守運用もつらい。今後はこの記事に書いてあるようにニーズが急増するため、都度個別対応をしていてはスピードも工数も足らず破綻してしまう。より早く大きな価値を探索し提供し続けられる体制を作るため、こうした運用を脱却し、プロダクトが責任を持ってAPIを提供しようという変化である。
今あるInternalAPIが全てこうなっているわけではない。既に汎用的な用途で作られているものもあれば、少し手を加えるだけで汎用化できるものもある。こうしたAPIは、このドキュメントの基準を満たせば、新たに作り直さずとも再利用可能APIとして宣言・提供できる。
再利用可能APIの立ち位置

品質基準とAPI種別の関係性
品質のグラデーション
再利用可能APIは、汎用的に利用可能なAPIを最速で揃えるための品質標準である。「WebからはできるがAPIからはできない」操作がなくなるレベルまでの拡充を目指す。
一方で全社として最終的に目指したいのは、品質・設計・運用の理想形に加え、ドメイン境界まで意識したドメインAPIである。この2つは別物ではなく、同じ基準の上のどこまで到達したかの違いでしかない。
* [MUST] を全て満たせば、再利用可能APIとして宣言できる。
* [MUST] と [SHOULD] を全て満たしたものを、ドメインAPIと呼ぶ。
* [MAY] はドメインAPIの要件には含めないが、検討そのものは [SHOULD] と同様に求める。
重要なのは [SHOULD] の扱いである。これは「満たさなくても再利用可能APIとしての宣言はできるが、満たせない理由を検討したうえで見送る」もの。確保できる工数や既存クライアントとの互換性など、諦める判断そのものは各チームに委ねる。ただし検討を飛ばして最初から選択肢に入れない、という進め方は避けてほしい。
新しく作るAPIは、最初から [SHOULD] まで満たす前提で設計してほしい。[SHOULD] の多くは設計段階で意識していれば追加コストがほとんどかからないもので、後から引き上げようとすると命名やレスポンス構造の変更、つまり破壊的変更を伴って高くつく。すでにあるAPIについては、段階的に移行していけばよい。
PublicAPIとして一般公開するAPIの基準は、ドメインAPIとする。したがって、[MUST] だけを満たしていても、PublicAPIとしてそのまま一般公開できるとは限らない。
バージョニング戦略や破壊的変更に関するドキュメントを整備予定。できたら貼る。
PublicAPIを作る際の手順ガイドラインを整備予定。できたら貼る。
API種別（Internal / Private / Public / Admin）との関係
混同しやすいが、「再利用可能API」は InternalAPI・PrivateAPI・PublicAPI・AdminWebAPI・AdminaAPI といったAPI種別と並ぶ区分ではない。これらの種別が「どの経路でアクセスされるか・どの認証が用いられるか」を表すのに対し、再利用可能APIは「どれだけ汎用的で再利用に耐えるか」という、一つ上のレイヤーの品質基準である。
実体としての再利用可能APIは InternalAPI として実装される。その同じ実体を、api-hub を通せば PublicAPI として、p2i proxy を通せば PrivateAPI として公開できる。つまり「再利用可能APIかどうか」と「どの種別で公開するか」は直交する軸であり、1本の再利用可能APIが複数のAPI種別を担えるようになる。裏を返せば、どの種別で公開されても安全性の問題が起きないことが「再利用可能」の前提条件である。
freee API標準との関係
freee API標準は、API種別ごとの通信経路・認証方式・エラー形式などを定める技術標準であり、再利用可能APIとはレイヤーが異なる。API標準が比較的低レイヤーの物理的な層の標準、再利用可能APIは具体的な実装や品質の標準である。
API標準のほかにも、再利用可能APIと連携する個別領域の標準・ガイドラインがある。
* ページネーションの仕様はページング標準
* http で使う OpenAPI の管理方法はOpenAPI標準
* gRPC 連携で使う proto の管理方式はproto管理標準
* サービス間の API／イベント連携の原則はサービス間連携ガイドライン
再利用可能APIがInternalAPIである理由
再利用可能APIをInternalAPIとして整備するのは、InternalAPIがアーキテクチャ的に最も内側に位置するからである。これはfreee Service Architectureの思想を引き継いでいる。プロダクトは汎用的なAPIをInternalAPIとして提供し、これがドメインを操作する唯一のインターフェースになる。プロダクト間の連携は、このAPIを用いるか、イベント駆動でデータをやり取りすることで実現する。クライアントからプロダクトを利用する際には、間にGateway Serverを置くことでInternalAPIを柔軟に活用できる。
freeeでのAPI設計の歴史と変化についてアーキテクチャ視点でまとめられている記事もあるので、ぜひ読んでみてほしい。
APIファースト戦略に至るまでの歴史（アーキテクチャ視点）

現時点では以下のような仕組みを用意している。
* api-hub という基盤を利用することで、指定したInternalAPIをPublicAPIとして公開できる
* p2i proxy という基盤機能を利用することで、指定したInternalAPIをPrivateAPIとして公開できる
InternalAPIが充実することで、柔軟であり続けるシステムアーキテクチャが実現できる。将来的に実現できるかもしれない理想像はドメインAPIのガイドラインに記載している。
設計の4原則
個々の基準は細かく分かれているが、その背骨になっているのは次の4つの考え方である。判断に迷ったときは、この4原則に立ち返ってほしい。
* カプセル化
    * 内部のDB構造やビジネスロジックの実装詳細を晒さない。ドメインの操作はAPI経由のみとし、内部状態の直接書き換えは許さない。
* ビジネス・プリミティブ
    * 操作の粒度は、そのドメインにおける「最小の業務単位」とする。マスタ系はCRUDや一括処理、プロセス系は「承認する」「差し戻す」といった業務上の振る舞いに基づく。
* 整合性
    * そのドメイン知識に基づくバリデーションはAPI内部で完結させ、APIが成功を返した時点でドメインのビジネスルールが満たされていることを保証する。
* 中立性
    * 特定UIや特定の呼び出し元（BFFなど）の都合に特化した加工・制御をしない。複数ドメインをまたぐオーケストレーションは上位層(呼び出し側)の責務とし、自ドメインの本質的な機能に集中する。
運用方針
設計と公開
APIファースト戦略の方針に沿って、今後追加するAPIは再利用可能APIとして設計・作成することを推奨する。前述のとおり、新規に作るものは最初から [SHOULD] まで満たす（＝ドメインAPIの水準で作る）前提で設計してほしい。
Webプロダクトから利用する場合は、再利用可能API（InternalAPI）を p2i proxy で PrivateAPI として公開して利用するのが基本形である。これにより、InternalAPI と PrivateAPI を別々に実装するダブルメンテを避け、再利用可能API 1本のメンテナンスで運用が回る状態を目指す。ただし、これはWebから利用するPrivateAPIを必ずこの方法で提供しなければいけないという強い制約ではない。画面に表示するための情報を1回のリクエストで取得したい、画面表示のためにレスポンス構造を特化させたい、など特有の要件や仕様で汎用化できない場合は、PrivateAPIとして実装しBFFとしての役割を担わせるべきである。この場合には再利用可能APIも作成し、PrivateAPIと並行してメンテナンスを行うこと。
宣言の方法
このドキュメントに記載した基準のうち [MUST] をすべて満たすものを、再利用可能APIと定義する。再利用可能APIとして宣言するには、freee-openapi または freee-protobuf にスキーマを掲載する必要がある。
基準の充足は、すべてを中央集権的にレビューするのではなく自己申告制とする。チェックリストを満たしているかは、再利用可能APIとして宣言する人が確認する。[SHOULD] を見送る判断も、オーナーであるドメインチームが行う。見送った項目については「なぜ満たせないのか」を、DDや設計ドキュメントに1行残しておくとよい。後から引き上げる際の手がかりになる。
HTTPの場合
リポジトリに設定を入れたうえで、スキーマに x-freee-openapi: true というタグを付けるだけで掲載できる（参考: freee-openapiの使い方）。掲載すると、以下が自動で達成される。
* 社内に再利用可能なAPIとして周知できる
* 該当APIを叩くためのSDK（Go / Ruby）が自動生成される
* 専用サイトでGUIとしてスキーマ情報を閲覧できる
* モニタリングの「再利用可能API数」として計測され、APIファースト戦略の推進度に反映される
gRPCの場合
ProtobufのRPC定義にコメントを追加することで宣言できる。


// これらは有効
service Foo {
  // @reusable-api
  rpc Bar(BarRequest) returns (BarResponse) {}
  // あいうえお
  // @reusable-api
  // かきくけこ
  rpc Baz(BazRequest) returns (BazResponse) {}
}
// これらは無効
service Foo {
  // 前後に余計な  @reusable-api  文字がある
  rpc Bar(BarRequest) returns (BarResponse) {}
  // @reusable-api
  // 宣言とRPC定義の間に空白行がある
  rpc Baz(BazRequest) returns (BazResponse) {}
}
厳密には、以下を満たすものが再利用可能APIとしての宣言とみなされる。
* RPC定義の直前にあるコメント（複数行でもよい）の一部であること。宣言とRPC定義の間に空白行がある場合は無効となる
* 前後の空白を除き、@reusable-api というキーワードのみからなる行であること。
gRPCについてもGUIカタログを準備中であり、2026年9月中を目処に提供される予定である。
利用のルール
宣言済みのAPIは、検証段階であればドメインチームに相談せず利用してよい。ただし、リクエスト数の増減によってインフラリソースの調整や監視体制の整備が必要になることもある。また破壊的な変更を加える際の連携も必要である。正式な利用の開始・停止にあたっては、ドメインチームと連携を取ること。
基準
(前提) freee API標準への準拠
freee API標準は、通信経路・認証方式・エラー形式などを定める技術標準であり、社内のベストプラクティスの積み重ねでできている。以下の基準は、すべてこれに準拠していることを前提とする。特に守ってほしいものは個別の基準として抜き出しているが、抜き出していないものを守らなくてよいという意味ではない。社内の標準的なプロダクト構築手順に則り、omegaの仕組みを使っていれば遵守できているものも多いはずである。
要求レベルの定義
RFC 2119 に倣い、"MUST" / "SHOULD" / "MAY" を要求レベルのキーワードとして利用する。ただし MAY は、RFC 2119 の「純粋に任意」よりも強い意味で用いる。
* MUST
    * その定義が仕様の絶対条件であることを示す。すべて満たすことで再利用可能APIとして宣言できる。
* SHOULD
    * 状況次第では適用しない理由が存在することを意味する。しかし、その意味を慎重に検討した上で、別の選択肢を考える必要がある。MUSTと合わせてすべて満たしたものをドメインAPIと呼ぶ。
* MAY
    * ドメインAPIの基準としても必須としないものを定義している。考え方や重要度はSHOULDと同様である。優先度が低い、満たさなくてよいというわけではない。
基準のカテゴリ
基準は、判断の性質によって4つに分けている。



カテゴリ	性質
1	運用体制と品質	誰がオーナーで、どんな品質保証のプロセスを通すか。実装そのものではなく体制・合意の話
2	技術要件	規約どおりかを機械的に、あるいはレビューで一目で判定できる項目
3	ドメイン設計	ドメインごとの個別判断が必要で、正解が一様には決まらない項目
4	API設計品質	ドメイン知識に依らず、一般的なAPI設計として求められる品質


「技術要件」は曖昧さがなく、明確に守れているかどうか分かるものである。一方で「ドメイン設計」や「API設計品質」は追い求めたいものだが絶対的な正解が存在しないものもあり、設計者の練度に依存するものになる。各組織でレビューをできる体制を構築することが理想だが、相談できる相手がおらず設計の壁打ち相手が欲しい場合にはAPI基盤でも対応できるので #ask-api_first で相談してほしい。
いつかはAIレビュー(壁打ち)ができる体制を整えて全社に提供したい
1. 運用体制と品質
[MUST] 1-1. ドメインチームがオーナーシップを持ち保守運用を行う体制になっている
リソースの都合で、APIの設計・実装を他チームが担当するケースもある。それ自体に問題はないが、レビュー・保守・運用・改善は、オーナーであるドメインチームが担うべきものである。
[MUST] 1-2. QA等を行いプロダクトとしての品質を担保している
再利用可能APIもプロダクトを操作する立派な機能である。APIに不具合があれば、クライアント側の挙動が乱れたり、不整合なデータが作られたりしてしまう。
特に既に存在しているInternalAPIを再利用可能APIとして運用としている場合には、過去に品質担保が行われているか確認が必要。
[MUST] 1-3. 脆弱性診断を行いセキュリティが担保されている
再利用可能APIはInternalAPIとはいえ、様々なクライアントから利用される。そのままPrivateAPIやPublicAPIとして公開される可能性もある。したがって宣言にあたっては、外部に露出するAPIと同等のセキュリティ要件を満たす必要がある。
特に既に存在しているInternalAPIを再利用可能APIとして運用としている場合には、過去に脆弱性診断が行われているか確認が必要。
[MUST] 1-4. 他プロダクトやAIエージェントがドメインチームへの問い合わせなしに自律的に利用を開始しても問題ない
正式利用の前には相談をすることをルールとしているので現時点では認識や覚悟の話でしかないが、この前提に立った運用をするべきである。
「特定のプロダクトやシステムからしか呼ばれない」「特定の業務フローの中でしか呼ばれない」という呼び出し文脈を安全性の前提にしているAPIは、この基準を満たせない。
[SHOULD] 1-5. 破壊的変更を容易に行わない前提で設計・議論している
そのドメインにとって柔軟性・拡張性が高く、将来の開発予定にも耐えられる設計を、できる限り考え抜いてほしい。難易度が高く正解のない領域だが、特に PublicAPI として公開されると、破壊的変更には長い準備期間と多くのコミュニケーションコストがかかる。社内のみの利用でも、変更に伴う周知や移行の段取りは提供側が責任を持つためコストがゼロではない。
2. 技術要件
[MUST] 2-1. InternalAPIである
再利用可能APIがInternalAPIである理由のセクションで述べたとおり、InternalAPIとして作成することで柔軟な展開が可能になる。
[MUST] 2-2. InternalSessionによる認証を導入している
InternalSession による認証を導入する。安全にスケールさせながらAPIを増やしていくために欠かせない。InternalAPIだから認証しなくてよい、という時代ではない。
[MUST] 2-3. プロダクトとしての必要な認可制御(権限・プラン制御)がかけられている
そのエンドポイントでどのようなプラン・権限制御が必要なのかドメインチームとして検討してほしい。その結果、どんなユーザー・プランでも呼び出せるべきと判断するのであれば認可制御をかけなくても問題ない。
様々なクライアントから利用されるため、認可制御が不十分だとセキュリティインシデントや事業損失に直結してしまう。InternalAPIだから制御をかけなくてよい、というわけではない。
* 会計の取引を作成する権限のない従業員が、別プロダクト経由なら再利用可能APIを使って取引を作成できてしまう
* 取引先の閲覧権限のない従業員が、あるプロダクトのリソースAに紐付いた取引先を、再利用可能API経由なら見られてしまう
という状態はまずい。
また、「特定のシステムやバッチ、イベント経由でしか呼ばれない」という呼び出し元への信頼を、認可制御の代わりにしてはならない。あらゆるクライアントから直接呼ばれても、権限・プランに加えて「その操作を実行してよい状態か」というドメイン上の前提条件が破られないことを、API側で保証すること。
認可基盤の sekisyo を用いることを推奨する。既に存在する認可の仕組みや簡単な新規実装で実現することも可能。迷った場合にはsekisyoチームに相談すること。
[MUST] 2-4. 利用者が自律的に利用方法を理解できるスキーマ情報が整備されている
HTTPなら OpenAPI、gRPCなら protobuf でAPIのスキーマ情報を定義する必要がある。
* HTTPの場合
    *  TypeSpec の利用を推奨する
        * ただし、最終的に OpenAPI を生成できることは必須である
* gRPCの場合
    * freee-protobuf でproto定義を管理することを必須とする
        * freee-protobuf をSingle Source of Truthとする方式と、それぞれのリポジトリにproto定義を配置して都度syncする方式があるが、どちらを採用してもよい
[MUST] 2-5. エラーレスポンスの形式がfreee API標準に沿っている
エラーの形式が揃っていれば、ハンドリングがやりやすくなる。omegaが提供しているエラー処理機構を使うと簡単に実現できる。
[MUST] 2-6. (一覧取得APIの場合) ページネーションのためのパラメータ名を標準の命名で受け入れられる
リクエストパラメータはpage_token/page_size/offsetの3点セット。カーソルページネーションのみならpage_size/page_token、OFFSETページネーションのみならpage_size/offsetを使う。カーソルページネーションのレスポンスパラメータはnext_page_tokenとする。後方互換のためにpage/per_pageやlimitを残しても良いが、標準の命名によるパラメータも受付可能にすること。
ページネーションの仕様はページング標準にあり、ここはその要点の抜粋である。カーソル方式の優先や方式比較など詳細はそちらを参照してほしい。
[SHOULD] 2-7. (一覧取得APIの場合) 非標準のページネーションパラメータ名を新規に採用していない
2-6 では後方互換のために page / per_page や limit を残すことを許容しているが、これはあくまで既存クライアントのための措置である。新しく作るAPIでこれらの非標準命名を採用してはいけない。既存APIに残っているものも、新しく使い始めるクライアントからは標準の命名で指定してもらい、古い形式のパラメータを使うクライアントがいなくなったら消すのが理想である。
[SHOULD] 2-8. レスポンス構造が標準に沿っている
2-8-1. トップレベルの階層構造
（HTTPの場合）単体のドメインモデルを返却する場合には階層を掘らず、トップレベルにフィールドを展開すること。複数のドメインモデル・ドメインモデルと同時にメタデータを返す場合は階層を掘ってもよいが、その場合はAPIの粒度が最小でない可能性も疑い、設計の見直しも検討する。 （gRPCの場合）API標準が定めるように階層を掘る。階層の名前はドメインモデル名のsnake_case表記を推奨する。これは、ドメインモデルの型定義がproto定義ファイル上で散らばることを防ぐための方針である。ただし、api-hubを通じて一般公開する際は、api-hubで階層を展開しユーザーが受け取るレスポンスがHTTPの場合と同様の形式になるようにする。なお、api-hubで階層を展開する設定はまだ整備できていないので、この形式を使用する場合はAPI基盤に必ず相談すること。複数のドメインモデル・ドメインモデルと同時にメタデータを返す場合はapi-hubで階層を展開しなくてもよいが、その場合はAPIの粒度が最小でない可能性も疑い、設計の見直しも検討する。


GET /api/internal/users/{user_id}
# OK
{
  "id": 1,
  "name": "hoge"
}
# NG
{
  "user": {
    "id": 1,
    "name": "hoge"
  }
}


// gRPCの場合
serice Hoge {
  rpc GetHogeFuga(GetHogeFugaRequest) returns (GetHogeFugaResponse) {}
}
// OK
message GetHogeFugaResponse {
  optional HogeFuga hoge_fuga = 1;
}
// NG: Response型にドメインモデルを展開している
message GetHogeFugaResponse {
  optional int64 id = 1;
  optional string title = 2;
  ...
}
2-8-2. トップレベル配列の禁止（HTTP）・非推奨（gRPC）
（HTTPの場合）トップレベルを配列にすることは禁止。オブジェクトとすること。 （gRPCの場合）server streaming responseはAPI基盤側の対応方針が未確定のため、どうしても必要な場合のみ使用する。使用する場合はAPI基盤に必ず相談すること。


GET /api/internal/users
# OK
{
  "data": [
    { "id": 1, "name": "hoge"},
    { "id": 2, "name": "fuga"}
  ]
}
# NG
[
  { "id": 1, "name": "hoge"},
  { "id": 2, "name": "fuga"}
]


// OK
rpc Example(FooRequest) returns (FooResponse);
// 非推奨: streaming response となっている
rpc Example(FooRequest) returns (stream FooResponse);
2-8-3. （主に検索系APIで値の配列を返す場合の）配列のフィールド名
（HTTPの場合）配列で返却するドメインモデルのフィールド名は data とする。複数の異なるドメインモデルを返す場合（リソースではなくusecaseベースなAPIの場合）はそれぞれ適切な名称を付けてよいが、その場合はAPIの粒度が最小でない可能性も疑い、設計の見直しも検討する。 （gRPCの場合）配列で返却するドメインモデルのフィールド名は、ドメインモデル名の複数形のsnake_caseを推奨するが、 data とすることも認める。これはgRPCが必ずしもRESTの思想に従うものではなく、単一リソースを取得するgRPC APIとの整合性を優先した方がよいと考えるためである。ただし、api-hubを通じて一般公開する場合はapi-hub上でフィールド名を data に書き換える設定を行い、HTTP APIとフォーマットを統一する。これは、内部的なAPI設計の都合と外部向けのスキーマ設計の都合のずれは、社内と社外の境界に位置するapi-hubで吸収すべきという思想に基づくものである。複数の異なるドメインモデルを返すAPIも認めるが、APIの粒度が最小でない可能性も疑い、設計の見直しを検討する。


GET /api/internal/users
# OK
{
  "data": [
    { "id": 1, "name": "hoge"},
    { "id": 2, "name": "fuga"}
  ]
}
# NG
{
  "users": [
    { "id": 1, "name": "hoge"},
    { "id": 2, "name": "fuga"}
  ]
}


message User {
  uint64 id = 1;
  string name = 2;
}
// OK
// 一般公開する際はapi-hubでusersをdataに変換する
message SearchUsersResponse {
  repeated User users = 1;
}
// OK
// フィールド名をdataとすることは、推奨はしないが許容する
message SearchUsersResponse {
  repeated User data = 1;
}
2-8-4. （主に検索系APIにおける）メタデータの位置
(主に検索系APIにおいて)メタデータは meta などで階層を掘らず、total_count や next_page_token をトップレベルに配置する。リクエストされたpage_sizeやoffsetを載せない。


GET /api/internal/users?page_size=2&offset=0
# OK
# api-hubがユーザーに返すレスポンスでは users が data に変換される
{
  "users": [
    { "id": 1, "name": "hoge"},
    { "id": 2, "name": "fuga"}
  ],
  "total_count": 10,
  "next_page_token": "dummy_token"
}
# NG
{
  "users": [
    { "id": 1, "name": "hoge"},
    { "id": 2, "name": "fuga"}
  ],
  "meta": {
    "total_count": 10,
    "next_page_token": "dummy_token"
  }
}
# NG
{
  "users": [
    { "id": 1, "name": "hoge"},
    { "id": 2, "name": "fuga"}
  ],
  "total_count": 10,
  "next_page_token": "dummy_token",
  "page_size": 2,
  "offset": 0
}


// OK
message SearchUsersResponse {
  repeated User users = 1;
  int64 total_count = 2;
  string next_page_token = 3;
}
// NG: meta で階層を掘っている
message SearchUsersResponse {
  repeated User users = 1;
  SearchUsersResponseMeta meta = 2;
}
message SearchUsersResponseMeta {
  int64 total_count = 1;
  string next_page_token = 2;
}
// NG: リクエストのパラメータ（page_sizeやoffset）を載せている
message SearchUsersResponse {
  repeated User users = 1;
  int64 total_count = 2;
  string next_page_token = 3;
  int64 page_size = 4;
  int64 offset = 5;
}
[SHOULD] 2-9. (差分取得を提供する場合) 更新日時の検索パラメータが標準通りである
更新日時で絞り込む検索パラメータは start_updated_time / end_updated_time、単位は秒まで。システム的な updated_at ではなく別途管理した更新日時で絞り込むのが理想（updated_at しかなければそれでよい）。RDBで提供する際はパフォーマンス要件をよく考慮する。
差分取得を提供するかどうか自体は任意である。ここで求めているのは、提供する場合に命名を標準に揃えることである。
gRPCの場合、日時は google.protobuf.Timestamp 型を用いて表す。
3. ドメイン設計
[MUST] 3-1. 特定の画面やクライアントのためではなく汎用的である
BFFのように特定のクライアントのために存在するのではなく、ユースケース単位で提供すべきものである。ドメインによって、リソースベースのCRUD操作が向くこともあれば業務操作が向くこともあり、ベストな選択は一様ではない。
4原則の「中立性」がここに対応する。特定UIや特定の呼び出し元の都合に特化した加工・制御をしない、ということである。複数ドメインをまたぐオーケストレーションは上位層（呼び出し側）の責務とし、自ドメインの本質的な機能に集中する。
[MUST] 3-2. バリデーション・整合性の担保とデフォルト値の設定がAPI側で完結している
クライアント側でバリデーションをする前提や、特定のデフォルト値を指定してくれる前提では、APIとして期待通りの動作を担保できない。必要なドメイン制約はAPI内部で確認し、適切なデフォルト値もAPI側で用意する必要がある。
4原則の「整合性」がここに対応する。APIが成功を返した時点で、そのドメインのビジネスルールが満たされていることを保証できる状態にする。
[SHOULD] 3-3. 粒度が最小の業務単位になっている
操作の粒度は、そのドメインにおける「最小の業務単位」とする。4原則の「ビジネス・プリミティブ」がここに対応する。
* マスタ系: データの整合性を保つためのCRUD操作や一括処理。一括操作系は、その単体処理が最小の業務単位であれば問題ない。
* プロセス／ワークフロー系: 「承認する」「差し戻す」といった、業務上の振る舞いに基づいた操作。
業務を遂行するために必要な操作を過不足なく網羅しつつ、不要な副作用を発生させない最小限の単位で提供する。1つの操作に副作用を詰め込みすぎると、柔軟性が失われ、テストや変更の影響範囲が広がる。
[SHOULD] 3-4. 単一ドメインに責務が閉じている
そのドメインが集中すべきことのみに責務を閉じる。「中立性」を、ドメイン境界の観点まで厳密にしたものである。
推奨しない例:
* 経費精算を作成するAPIで部門名を受け取り、内部で部門の find_or_create を動かす → 前段で部門のAPIを叩いてから経費精算のAPIを叩く流れにする。
* 経費精算を取得するAPIで部門名まで取得してレスポンスに含める → 部門IDを返却し、部門のAPIを叩いてもらう流れにする。
許容する例:
* 事業所の設定を変更するAPIで、その設定値に合わせて再計算が必要な帳票情報を更新する → ドメイン制約として不整合を防ぐために必ず行う必要があるものは、それ自体が「最小の業務単位」に含まれるのでOK。
複数ドメインをまたぐ複雑な制御は上位層（BFFやオーケストレーター）の責務とし、APIはあくまで自ドメインの本質的な機能に集中する。なお、上記の「前段で別ドメインのAPIを叩く」流れに対する需要が非常に多い場合、複数ドメインを束ねる粒度のAPI（いわゆる「ユースケースAPI」）として定義することはありうるが、この概念はまだ正式に整備されていないため、本ドキュメントでは推奨パターンとしては扱わない。
4. API設計品質
[MUST] 4-1. スキーマに制約や仕様が十分に書かれている
文字数制限や一意制約、予想がしづらい挙動などはスキーマの制約や説明から読み取れると自律的に利用をしやすくなる。
gRPCの場合、custom optionsまたはインラインコメントで表現する。最低限何かしらの形で表現されていれば基準を満たしたとして良いが、 googleapis/google/api/field_behavior.proto at master · googleapis/googleapis や GitHub - bufbuild/protovalidate: Protocol Buffer Validation - Go, Java, Python, C++ and JS/TS を用いて構造的に表現することを推奨する。
[SHOULD] 4-2. 内部のDB設計や歴史的経緯に引きずられない構造・命名になっている
4原則の「カプセル化」がここに対応する。結果的にDBのテーブル構造と同じになること自体は問題ないが、テーブル構造から切り離して設計することで拡張性や利便性を確保できることが多い。歴史的にそのままになっている命名を、実態に沿ったものへ変える好機でもある。
実際に、API定義が曖昧なまま作られた結果、データベースのプロパティ名がそのまま外部に公開されてしまっているケースがある。一度公開してしまうと直すのに破壊的変更が必要になるため、設計の段階で気付きたい。
[SHOULD] 4-3. 過度に省略した命名が採用されていない
description を読まなくても、命名だけで何を表すか想像できる状態が理想である。フィールド名は短いコメントである意識を持つ。一般的で一意に解釈できるもの以外は省略しないことを推奨する。
[SHOULD] 4-4. 似通った性質のパラメータの命名が揃っている
特にフラグ系の命名は歴史的経緯でずれていることが多い（use_xxx / yyy_enable / is_zzz が混在するなど）。性質の近いパラメータは命名規則を揃える。
[SHOULD] 4-5. (一覧取得APIの場合) total_count の提供有無を要件から判断している
提供するかどうかは任意だが、なんとなく付ける・なんとなく付けないという判断は避けてほしい。要件として必須でなければ、パフォーマンス観点から提供しないことを推奨する。
[MAY] 4-6. (一覧取得APIの場合) カーソルページネーションを導入している
パフォーマンス観点から推奨する。導入自体を絶対条件とはしないが、「導入できないのか／必要ないのか」を十分に検討したうえで判断してほしい。ランダムアクセスが必要な場合などOFFSET方式を選ぶべきケースもあるが、カーソル方式とOFFSET方式はどちらも1つのAPIで提供できる。どちらか片方のみを採用するのではなく、両方を採用することも検討してほしい。
[MAY] 4-7. (一覧取得APIの場合) 更新日時による差分取得を提供している
提供するかどうかは任意だが、特にPublicAPIとして公開する際に要望が多い。変更をトリガーに何かを行うシステムを作るためにはWebhook等の方が便利だが、システム構成によっては更新日時による差分取得が必要なケースも求められる。
[MAY] 4-8. ( 更新APIの場合) 部分更新を採用している
（HTTPの場合）PATCHを採用することを推奨する。単にHTTPメソッドの話ではなく、部分更新として提供できているかどうかである。HTTPであれば、完成形の情報をすべて渡す必要があるPUTよりも、更新したい箇所だけ更新できるPATCHのほうが好ましい。採用しない場合も、採用できない理由を検討したうえで判断してほしい。
（gRPCの場合）gRPCの場合もPATCHに相当する部分更新が可能なようにRPCを設計することが好ましい。なお、Protobufスキーマの書き方によっては値の省略とデフォルト値を区別できない場合があることに十分注意し、意図しないフィールドの更新が発生しないようにすること。
基準早見チェックリスト
各項目の詳細は基準のセクションにまとめている。
A. 再利用可能APIとして宣言するために満たすもの（MUST）
運用体制と品質


1-1. ドメインチームがオーナーシップを持ち保守運用を行う体制になっている


1-2. QA（テストによる品質保証）を行いプロダクトとしての品質を担保している


1-3. 脆弱性診断を行いセキュリティが担保されている


1-4. 他プロダクトやAIエージェントがドメインチームへの問い合わせなしに自律的に利用を開始しても問題ない
技術要件


2-1. InternalAPIである


2-2. InternalSessionによる認証を導入している


2-3. プロダクトとしての必要な認可制御(権限・プラン制御)がかけられている


2-4. 利用者が自律的に利用方法を理解できるスキーマ情報が整備されている


2-5. エラーレスポンスの形式がfreee API標準に沿っている


2-6. (一覧取得APIの場合) ページネーションのためのパラメータ名を標準の命名で受け入れられる
ドメイン設計


3-1. 特定の画面やクライアントのためではなく汎用的である


3-2. バリデーション・整合性の担保とデフォルト値の設定がAPI側で完結している
API設計品質


4-1. スキーマに制約や仕様が十分に書かれている
B. ドメインAPIとして満たすもの（SHOULD）
Aと合わせてすべて満たしたものをドメインAPIと呼ぶ。満たせない項目があっても再利用可能APIとしての宣言はできるが、見送る場合は「なぜ満たせないのか」を検討したうえで判断すること。検討の結果を DD か設計ドキュメントに1行残しておくと、後から引き上げる際の手がかりになる。
運用体制と品質


1-5. 破壊的変更を容易に行わない前提で設計・議論している
技術要件


2-7. (一覧取得APIの場合) 非標準のページネーションパラメータ名を新規に採用していない（page / per_page / limit）


2-8. レスポンス構造が標準に沿っている（トップレベル配列の禁止／単一メインオブジェクトは data／メタデータは階層を掘らずトップレベル配置）


2-9. (差分取得を提供する場合) 更新日時の検索パラメータが標準通りである（start_updated_time/ end_updated_time、秒単位）
ドメイン設計


3-3. 粒度が最小の業務単位になっている


3-4. 単一ドメインに責務が閉じている（他ドメインの find_or_create を内包しない／他ドメインの値をレスポンスに混ぜない）
API設計品質


4-2. 内部のDB設計や歴史的経緯に引きずられない構造・命名になっている


4-3. 過度に省略した命名が採用されていない


4-4. 似通った性質のパラメータの命名が揃っている


4-5. (一覧取得APIの場合) total_count の提供有無を要件から判断している
C. ドメインAPIの要件に含めないもの（MAY）
満たさなくてもドメインAPIを名乗れる。考慮しなくてよい、満たさなくてよい、というわけではないがドメインAPIの基準としても必須としないものを定義している。要件に応じて判断すればよい。
API設計品質


4-6. (一覧取得APIの場合) カーソルページネーションを導入している


4-7. (一覧取得APIの場合) 更新日時による差分取得を提供している


4-8. (更新APIの場合) 部分更新を採用している
ケース別実装例
ガイドラインに沿ってどう実装するか迷ったときのためのケース集。例のパスやドメインはfreeeのサービスに寄せているが架空のものであり、実際は要件によって最適な実装が異なる。迷ったら #ask-api_first または #eng-kikanchos で相談すること。
サービス間連携の実装方針はサービス間連携ガイドラインに委ね、ここではインターフェースの設計方針に絞る。
ここで例示しているものは全て架空のシステムであり、実際のプロダクトとは設計や要件は異なります
1. API粒度の設計
要件: 請求書を作成した際に、会計の取引も自動で作成したい。
❌ アンチパターン: 複合API（オーケストレーション混入）


POST /invoices/create_with_deal
{
  "invoice": { "issue_date": "2026-01-14", "amount": 10000 },
  "deal": { "account_item_id": 456, "tax_code": 1 }
}
複数ドメインをまたぐ操作を1つのAPIに詰め込んでおり、請求書ドメインが会計ドメインの知識を持ってしまう（中立性違反）。片方が失敗したときのロールバックが複雑で、会計側の仕様変更が請求書APIに波及する。
✅ 推奨: 単一責任API + 上位層でオーケストレーション


# 請求書ドメインのAPI
POST /invoices
{ "issue_date": "2026-01-14", "amount": 10000, "partner_id": 123 }
# 会計取引ドメインのAPI
POST /deals
{
  "issue_date": "2026-01-14",
  "partner_id": 123,
  "invoice_id": "INV-001",
  "lines": [ { "account_item_id": 456, "amount": 10000, "tax_code": 1 } ]
}
各APIは自ドメインの操作のみに集中し、2つのAPIの順次呼び出しはBFF／オーケストレーター層で行う。または請求書作成イベントをPubSubで発行し、会計側がsubscribeして取引を作成する。ただし「請求書が存在する際には必ず1対1で取引が存在する」というような仕様がある場合、整合性担保のための最小粒度のAPIとなるため許容する。
2. ビジネス・プリミティブ
要件: 経費申請の「承認〜コメント〜取引作成〜給与連携」を提供したい。
❌ アンチパターン: 副作用を詰め込みすぎ


POST /expense_applications/{id}/approve_and_process
{ "comment": "承認します", "issue_date": "2026-07-07", "attach_to_payroll_month": 8 }
承認という業務操作に、コメント・取引作成・給与連携という副作用が密結合している。「承認だけしたい」に対応できず、承認のテストにコメント基盤・会計連携・人事労務連携のモックが全て必要になり、ある仕様の変更の影響規模が大きくなってしまう。
✅ 推奨: 最小の業務単位で分割し、副作用は疎結合に


POST /expense_applications/{id}/approve            {}
POST /expense_applications/{id}/comment            { "message": "承認します" }
POST /expense_applications/{id}/create_deal        { "issue_date": "2026-07-07" }
POST /expense_applications/{id}/attach_to_payroll  { "month": 8 }
副作用は、承認イベントをPubSubで発行して各サービスがsubscribeするか、別エンドポイントで明示的に実行し、順序は上位層で制御する。
3. バリデーションの完結性
要件: 請求書から会計取引を作成する際、取引先・発行日・勘定科目などのバリデーションを行いたい。
クライアント側にバリデーションを依存させてサーバー側で行わないと、複数クライアントで同じ実装が必要になり、「会計期間が締められていたら登録不可」のようなビジネスルールがクライアントに散在し、データ不整合や保守性低下を招く。API内でバリデーションを完結させること。
4. 中立性の保持
要件: 会計の取引一覧を取得するAPIを提供したい。
❌ アンチパターン: 特定画面専用API


GET /deals/mobile_dashboard  # モバイル専用。固定フォーマット・固定ソートで最新10件
GET /deals/web_home          # Webホーム画面専用。全フィールド+関連データ・絞り込みあり
UI要件の変更のたびにAPIが増え、同じデータを返すAPIが乱立し、新しいクライアント（AI・外部連携）が使いづらい。
✅ 推奨: 汎用的なフィルタリング可能API


GET /deals          # 絞り込み、ソート条件、ページングあり
GET /deals/web_home # Webホーム画面専用。全フィールド+関連データ・絞り込みあり
日付範囲・ステータス・取引先などで絞り込み、任意フィールドでソート、カーソルページネーションで制御する。要件がカバーできるAPIは個別に作らず、汎用的なAPIを使う。画面の要件やパフォーマンスのために汎用的なAPIでは足りない場合には、BFFのAPIとして別途作成する。
5. マスタードメインへの依存
要件: 販売登録時に、会計の取引先が存在することを保証したい。
❌ アンチパターン: ドメイン間の直接的な書き込み依存（販売ドメインのAPI内部で取引先を find_or_create する）
販売ドメインが取引先の作成ロジックを持つことになり（責務の混在）、トランザクション境界が曖昧になる。他ドメインも同様に取引先を作り始めると、取引先ドメインの変更影響が広範囲に及ぶ。
✅ 推奨: 参照整合性チェック + 事前登録の強制


class SalesDomain::Usecases::CreateSales
  include Master::Concerns::Usecases::EnsureMasterExists
  def call(params)
    # 1. 取引先の存在確認（読み取りのみ）
    ensure_partner_exists!(partner_id: params[:customer_id], partner_type: :customer)
    # 2. 存在する場合のみ販売データを登録
    sales_repository.create(
      customer_id: params[:customer_id], amount: params[:amount], issue_date: params[:issue_date]
    )
  end
end
* マスタードメインの読み取りは許容: 参照整合性チェックのための読み取りAPI呼び出しは問題ない。
* 書き込みは上位層の責務: マスターデータの作成・更新はBFFやオーケストレーター層で明示的に制御する。
* 存在しない場合は具体的なエラーを返し、クライアント側で適切に処理できるようにする。
Q&A
Q1. Webの新規機能開発でAPIを追加する場合の基本方針は？
再利用可能API(InternalAPI)を作成し、それをp2i proxyの仕組みを利用してPrivateAPIとしても公開してWebから使うことを推奨とします。無理に共通化する必要はないため、運用上楽な方法を選んで問題ありません。Write系は共通化しやすそうだと考えています。
p2i proxyによる共通化を進める場合でも、画面の要件やパフォーマンスのために汎用化できない場合は、再利用可能APIを諦めて専用のAPIをPrivateAPIとして実装します。この場合にはPrivateAPIと同じ機能を実現できる再利用可能APIも併せて整備をお願いします。
Q2. Webから直接InternalAPIを叩けるなら、PrivateAPIとして公開しなくても良いのでは？
InternalAPIは内部通信を前提としているため、認証をはじめいくつかの点でそのままWebから叩くことはできません。
* 認証方式が異なる。InternalAPIはInternalSessionTokenによる認証、PrivateAPIはCookieSession(もしくはTrustedApplicationのOAuthアクセストークン)による認証
* InternalAPIではCSRF保護やIPアドレス制限、事業所ロックの確認がスキップされている
これまでは内部のドメインロジックを共有しつつ、PrivateAPI/InternalAPIで別々にAPIを作成する必要がありました。PrivateAPIが基本でInternalAPIは必要に応じて作る、という時代にはこれで成り立っていました。しかし今後は両方が必要になるため、ダブルメンテが当たり前に求められてしまいます。そこで、InternalAPIとPrivateAPIの根本的な差分を吸収し、そのままPrivateAPIとして公開できるp2i proxyの仕組みを用意しました。これにより再利用可能APIの1本のみをメンテナンスするだけで運用が回る状態を構築しています。
Q3. MUSTだけ満たして宣言してもよいのですか？ SHOULDはどこまで求められますか？
宣言できます。[MUST] をすべて満たしていれば再利用可能APIです。
ただし [SHOULD] については、「満たすかどうか」の前に「満たせない理由があるか」を必ず検討してください。工数が確保できない、既存クライアントとの互換性が壊れる、といった理由で見送る判断はチームに委ねます。避けたいのは、検討そのものをせずに選択肢から外してしまうことです。
なお新しく作るAPIについては、最初から [SHOULD] まで満たす前提で設計してほしいと考えています。[SHOULD] の多くは設計段階で意識していればほとんど追加コストがかからない一方、後から引き上げようとすると命名やレスポンス構造の変更を伴い、破壊的変更として高くつくためです。
Q4. ドメインAPIとは何が違いますか？ 別物として作るのですか？
別物ではありません。同じ基準のどこまでを満たしているか、という到達点の違いです。[MUST] をすべて満たしたものが再利用可能API、[MUST] と [SHOULD] をすべて満たしたものがドメインAPIです。
以前は基準が2つのドキュメントに分かれていましたが、差分が読み取りづらく、また再利用可能APIを作りに来た人が理想状態を知る機会を持てないという問題がありました。そのため基準はこのドキュメントに一本化しています。ドメインAPI側のドキュメントには、その水準のAPIが全プロダクトで揃ったときに何が実現できるのかを記載しています。
Q5. 既存の再利用可能APIを、後からドメインAPIに引き上げられますか？
引き上げられますが、命名やレスポンス構造の変更が破壊的変更になりやすい点に注意してください。最初からPublicAPIでの一般公開や長期安定提供を見込むなら、[SHOULD] まで満たす前提で設計するのが安全です。
Q6. 誰がチェックリストの充足を確認・承認しますか？
中央集権的なレビューは行わず、自己申告制です。freee-openapi に掲載する（宣言する）人が、チェックリストを満たしているかを確認します。[SHOULD] を見送る判断も、オーナーであるドメインチームが行います。
Q7. 複数ドメインをまたぐ操作を1本のAPIで提供したい要望が強いです。どうすれば？
原則は、各ドメインのAPIに分割し、順次呼び出しを上位層（BFF／オーケストレーター層、現状は再利用可能APIではない純粋なPrivateAPI）で行うか、イベント駆動で疎結合にします。需要が非常に多い場合は複数ドメインを束ねる粒度のAPIとして定義することもありえますが（いわゆる「ユースケースAPI」）、この概念はまだ正式に整備されていないため、現時点では推奨パターンとして示しません。
Q8. 他ドメインのマスタ（取引先・部門など）の存在を保証したいときは？
読み取りは許容、書き込みは上位層です。自ドメインのAPI内で他ドメインのマスタを find_or_create するのではなく、参照整合性チェックのための読み取りにとどめ、作成・更新はBFF／オーケストレーター層で明示的に制御してください。
Q9. gRPC で提供している InternalAPI も再利用可能API／ドメインAPIにできますか？
はい、可能です。基準としては記載されている通りです。
ただし、gRPC を再利用可能API／ドメインAPIとして宣言・計測する仕組みは整備中です。2026年9月中を目処にこれらの仕組みを用意する予定です。
Q10. 社内の複数プロダクトから汎用的に使われているInternalAPIなら、再利用可能APIとして宣言できますか？
汎用的に使われていることと、再利用可能であることは別です。「呼び出し元が信頼できる社内システムである」「特定の業務フローを起点にしたときだけ呼ばれる」といった暗黙の前提の上で安全性が成り立っているAPIは、たとえ複数プロダクトから使われていても宣言できません。認可制御などをAPI側で完結させ、どのクライアントから直接叩かれても安全な状態にしてから宣言してください。なお、呼び出し元を信頼する前提のままInternalAPIとして提供し続けること自体は問題ありません。



オーナーチーム
#go


Goノウハウ整備会でまとめた内容のうち、特にfreeeでGoのプロジェクトを進める上で知っておいて欲しい情報をまとめました。
https://drive.google.com/file/d/1NCseHvOnS4g4HA7rHjSv5q7498E5XH9F/viewチームの 80% が Google Drive のプレビューを表示しています接続 
https://drive.google.com/file/d/1WQ65zef0WfYVDdKNAoXlfp_lYCo_PZdM/viewチームの 80% が Google Drive のプレビューを表示しています接続 
https://docs.google.com/document/d/1CZVy5M_mrrM7IEebL0OINDtSy9cep0qnWgmlsXQH5w8/edit?tab=t.0チームの 80% が Google Drive のプレビューを表示しています接続 
* エラーハンドリングではエラータイプを指定し、標準ライブラリや外部ライブラリのエラーをハンドリングする時はomega-go/errorsでWrapする
    * 課題
    * 対応策
* SQLの組み立てはプレースホルダーやQueryBuilderを用い、ORMは単純なCRUDに対して用いる。また、DBアクセス時はLeakCheck機構を必ず利用する。
    * 課題
    * 対応策
    * LeakCheck機構活用の注意点（これだけは守って！）
* ormのTimestampをアプリ側で使うのは推奨しない
    * 課題
    * 対応策
* metadataをoverrideしたいケースのベストプラクティス
    * 課題
    * 対応策
* domain層のデータについて、基本的にデータ型を定義する
    * 課題
    * 対応策
* 外部ライブラリを使う時はinterfaceを使う側で定義する
* wireのベストプラクティス
* package構成のどこにどういうコードを置けばいいか
* テストを書く時のベストプラクティス
エラーハンドリングではエラータイプを指定し、標準ライブラリや外部ライブラリのエラーをハンドリングする時はomega-go/errorsでWrapする
課題
Goの標準ライブラリや外部ライブラリのエラーはスタックトレースがついておらず、そのままだとトラブル時にログからの調査が難しくなります。また、エラーをカテゴライズしてシステム起因のエラーならリクエストのレスポンス時にBugsnagにエラーを通知したいが、不正なリクエストのようにクライアント起因の場合は通知したくないといったケースもあります。こういったケースがあるため、Goでの開発ではエラーはそのまま使わず加工した方が便利に扱えます。
対応策
omega-go/errors のerrors.NewTypeを用いてSeverityを指定してエラータイプを定義し、標準ライブラリや外部ライブラリのエラーをそのエラータイプでラップする事でログの通知のハンドリングやスタックトレースの追加がされます。
errors.NewTypeでは以下のようにエラータイプをvariableとして定義を行います。ここでSeverityUserがクライアントエラー、SeviritySystemがシステムエラーを示すものになります。


var (
    // BadRequest indicates that it is caused by unexpected user input.
    BadRequest = NewType("BadRequest", SeverityUser)
    // SystemError indicates that it is caused by unexpected system behavior
    SystemError = NewType("SystemError", SeveritySystem)
)
標準ライブラリや外部ライブラリのエラー時には以下のようにエラーをラップする事でエラータイプやスタックトレースが付与されます。


	bytes, err := json.Marshal(input)
	if err != nil {
		return "", SystemError.Wrap(err, json.Marshal failed)
	}
SQLの組み立てはプレースホルダーやQueryBuilderを用い、ORMは単純なCRUDに対して用いる。また、DBアクセス時はLeakCheck機構を必ず利用する。
課題
ActiveRecordのようなORMの活用は便利な一方で生成されるクエリが想定外に複雑になり、大規模なシステムの開発においては問題となるリスクもあります。そのため、freeeではGoのプロジェクトにおいて基本的には生っぽくクエリを書く事を推奨しています。しかしながら、Sprintfなどで単純に文字列結合でSQLを組み立てるとSQLインジェクションなどの脆弱性が生まれます。また、検索機能などでは微妙に条件が異なるクエリが多くのパターン生まれる事があり、全てのパターンを網羅するようにクエリを用意するのもそれはそれで不便です。
合わせて、freeeでは事業所を跨いだデータアクセスはデータ漏洩のリスクにつながるためLeakCheckと呼ばれる事業所IDがコンテキストと異なる場合は処理を失敗としたいケースもあり、このような仕組みを自前で用意するのは面倒です。
ここではfreeeでの対処方針を示します。
対応策
omega-go/database/sql を用いる事で以下のように?の部分にパラメーターが入るようにセキュアにクエリを生っぽく書く事ができます。


	obj := entity.Membership{}
	query := "SELECT * FROM " + orm.MembershipTable.Name() + " WHERE user_id = ? AND company_id = ? AND class = ? ORDER BY id"
	found, err := sql.QueryRows(repo.db, query, uid, cid, pb.Membership_MEMBER).FindAndScanStruct(ctx, orm.Membership(&obj))
また、検索機能のようなクエリのパターンが爆発するようなケースではQueryBuilderによって条件を分岐させながらクエリを組み立てる事が可能です。


qb := sql.NewQueryBuilder("users AS u")
qb.AppendSelect("u.*")
if someCond {
  qb.AppendSelect("NULL", "c.*")
  qb.AppendJoin("JOIN companies AS c ON c.id = u.cid")
}
if status != 0 {
  qb.AppendAndWhere("status = ?", status)
}
単純なCRUDであればomega-goでORMが提供されています。


err := sql.InsertStruct(ctx, db, um)
また、これらを利用する場合標準でLeakCheckの機構が提供されています。
LeakCheck機構活用の注意点（これだけは守って！）
LeakCheck機構で事業所跨ぎのチェックを行うためには、CompanyIDの型はmetadata.CompanyIDである必要があります。（参考スレ）
uint64のような型にキャストした状態でQueryに含めた場合はLeakCheckが機能しなくなるため、こうしたキャストはせず、metadata から直接参照するかアプリ側で型定義を行ってください（example）。
LeakCheckが機能しない例: 


func (p FooRepository) FindBar(ctx context.Context, companyId uint64) ([]entity.Bar, error) {
	query := sql.NewQueryBuilder(FooTable)
    // uint64 なので事業所外のデータにアクセスできてしまう
	query.AppendAndWhere("company_id = ?", companyId)
	query.AppendSelect("*")
    var bar []*schema.BarSchema
上記の例では、omega-goのbeforeQuery / beforeExecによるチェックが動作しません。
また、omega-goのLeakCheckはScanのタイミングでも実行されます。この際も取得した結果をstructに格納する際の型にmetadata.CompanyIDが含まれる場合のみ、AssertAccessibleによるチェックが行われます。
ormのTimestampをアプリ側で使うのは推奨しない
課題
ORMのTimestamp (created_at, updated_at) は調査等には大変便利だが、この日時をエンドユーザーに表示したりソート順に使ったりしていると、データ移行・スキーマ拡張等によるスクリプトでの変更時に更新されて意図せぬユーザー影響を出してしまうことがあります。
対応策
omega-goのdatabase/ormではcreated_at, updated_atをGo structのフィールドには持たずDB上のみで扱うカラムにして（TimestampingMode = Implicit）、エンドユーザー向けに作成・更新日時を使いたい場合はドメインモデルの一部として別カラムに持つようにするのを推奨しています。
各種事情に応じてシステム上のcreated_at, updated_atを扱いたい場合はGo structに含めたり（TimestampingMode = Explicit）、Timestampが不要で全て自前で管理する（TimestampingMode = None）こともできますが、標準から外れるのでコメントに理由を書いておくようにしましょう。
metadataをoverrideしたいケースのベストプラクティス
課題
metadataはログの共通フィールドやリークチェック用に使うコンテキスト情報を保持していて、基本的にミドルウェアにて設定されますが、時折内容を上書きしたくなることがあります。
対応策
metadata.OverrideContextを使うと上書きできます。


ctx = metadata.OverrideContext(ctx, metadata.WithCompanyID(cid), metadata.WithUserID(uid))
歴史的経緯によりmetadata.OverrideContextV2は古く、そのうちdeprecateするので注意してください。
参考記事：omega-go/metadata v2

 
domain層のデータについて、基本的にデータ型を定義する
課題
例えば同じ文字列データ型であっても、電話番号やメールアドレスのように取り扱うデータのドメイン知識によってGoのプリミティブ型の文字列よりもデータ構造についてより詳細なルールがある場合があると思います。このような場合はただの文字列型としてプログラム上で取り扱うよりもEmail型などを別で定義する方がコードの可読性が上がる他、コンストラクタを用意して想定外のデータが入るのを防ぐ事ができるなどのメリットがあります。
対応策
プリミティブ型のstringと区別して扱いたい場合は delibirdの例 のようにDefined typeを利用する事ができます。


type FunctionID int64
type FunctionKey string
type DisplayName string
type Function struct {
	ID           FunctionID
	FunctionKey  FunctionKey
	DisplayName  DisplayName
...
}
例のようにコンストラクタを用意してデータを作成するように習慣づけする事でバリデーションを行う事ができるので想定外のデータが入ってきてバグが発生するリスクも減らす事ができます。
また、例のようにテストデータの作成に型スイッチを利用する事でテストデータの生成を簡単にする事ができます。


func Function(opts ...interface{}) domain.Function {
	f := domain.Function{
		ID:           domain.FunctionID(ID()),
		FunctionKey:  FunctionKey(),
		DisplayName:  domain.DisplayName(RandomString()),
        ...
	}
	for _, o := range opts {
		switch v := o.(type) {
		case domain.FunctionID:
			f.ID = v
		...
}
// 利用時は以下のようにするとFunctionのうちFunctionKeyのフィールドを引数で渡した値に置き換えてくれる。
function := fake.Function(fake.FunctionKey())
外部ライブラリを使う時はinterfaceを使う側で定義する
Goでは一般的に利用するパッケージのinterfaceを定義するのはパッケージ側ではなくパッケージを利用する側にして、パッケージ自身はconcrete typeを返す事を推奨しています。(Doc)
外部ライブラリによっては外部ライブラリ自身がモックを提供する事もありますが、基本的には外部ライブラリを利用する側でinterfaceを定義して必要な部分のみモックを作成する事でコード中で利用するAPIがはっきりとするなどのメリットがあります。
具体的には filebox-coreの例 のようにinterfaceを定義すれば良いです。
wireのベストプラクティス
wire を利用した DI（Dependency Injection 依存性注入）

 を参照してください
package構成のどこにどういうコードを置けばいいか
Go package構成標準

 を参照してください
TODO: yuichi+terashi
テストを書く時のベストプラクティス
[WIP]Goのテストを書く時の特に重要なノウハウ集

 を参照してください




C-FO 組織内 dbt プロジェクトの「統一的な作り方」を定めるガイドです。 既存 22 リポジトリの横断調査から、すでに揃っている点は除き、割れている点はこれからも割れると考えて標準を1つ選ぶ方針で構成しています。（あらゆる標準を作るのは非効率なのでこの方針にしています）
凡例
* ✅ 標準 … 本ガイドで推奨する書き方
* ❌ 避ける … 現状見られるが標準から外れる書き方
「現状例」について: 各リポジトリの調査で確認した スタイル傾向を代表する再構成例 です。逐語のコピーではなく、そのリポジトリの書き癖を示すためのものです。
参照実装: advisor-score-mart / crossing-report-datamart / benefit-coupon-recommendation（本ガイドに最も近い3リポジトリ。迷ったらこの3つを見る）

全体方針
標準化を 3つの階層 に分けて扱います。



階層	内容	強制方法
構造	層トポロジー・命名・材質・依存方向	本ガイド＋レビューで合議
表層	大小文字・カンマ・別名・インデント	.sqlfluff ＋ CI で機械強制（付録A）
運用	テスト・ドキュメント・マクロ・環境切替	本ガイド＋テンプレリポジトリ


表層は sqlfluff fix でほぼ自動整形できます。人が議論すべきは 構造 です。

ディレクトリ構成とレイヤー
Data Modeling Layer Policy

  を参照してください。

命名規約
✅ 標準



対象	規則	例
staging モデル	stg_<source>__<entity>（ダブルアンダースコア区切り）	stg_accounting__companies
intermediate モデル	int_<domain>__<entity>	int_accounting_advisors__advisor_client_links
ディメンションモデル	dim_<entity>	dim_companies
ファクトモデル	fct_<subject>	fct_monthly_charges
モデル用 yml	_<table>__models.yml（先頭アンダースコア）	_accounting__models.yml
ソース定義 yml	_<source>__sources.yml	_accounting__sources.yml


* モデル名は簡潔に。過度に長い記述的命名は避ける。
* 特にsourceやdomainに当たる部分が長い場合、一般的に使用される略称は用いて良い。しかし国際的な基準があるわけではなく個人の認識によって揺れるので、レビューアと認識が一致するならOKとする。
    * ✅ standard → std、identifier → id、database → db など
    * ❌ user → usr、customer → cust

ソースと鮮度（source / freshness）
✅ 標準
* すべての source テーブルに description と freshness（error_after 目安 24h） を定義する。
* source 定義は models/staging/<source>/_<source>__sources.yml に置く。
* 鮮度を必要としない場合は除外。
📍 参考
* freshness 運用の手本: advisor-score-mart / crossing-report-datamart / benefit-coupon-recommendation（共通マクロ freshness_query で loaded_at を注入）。

SQL コーディングスタイル（表層・.sqlfluff で機械強制）
本章は個人グローバル設定 ~/.claude/rules/sql.md のルールを組織標準として採用したものです。 すべて .sqlfluff（付録A）で自動整形・CI強制できます。
大文字・小文字：すべて小文字


-- ✅ 標準
select
    company_id
    , company_name
from stg_accounting__companies
-- ❌ 避ける
SELECT
    company_id
    , company_name
FROM stg_accounting__companies
📍 corporate-master は .sqlfluff に小文字を指定しているのにコードは大文字＝lint が強制されていない。CI 必須化（付録A）で防ぐ。
インデント：4スペース
* 4文字の空白文字をインデントとして扱う。
select 句とカンマ：先頭カンマ


-- ✅ 標準：select の後に改行してインデント、2つ目以降は先頭カンマ、別名は as
select
    id as company_id
    , name as company_name
    , date(created_at, 'Asia/Tokyo') as created_date
-- ❌ 避ける
select
    id as company_id,
    name as company_name,
    date(created_at, 'Asia/Tokyo') as created_date
* カラム別名には as を付ける。
from 句とテーブル別名：別名に as を付けない


-- ✅ 標準：from は同一行にテーブル名、テーブル別名に as を書かない
from stg_accounting__companies c
-- ❌ 避ける
from stg_accounting__companies as c
join：可能な限り using、不可なら on


-- ✅ 標準：結合キー名が一致するなら using
from deals d
    inner join companies c using (company_id)
    left join advisors a on d.advisor_id = a.id and d.dt = a.dt
* 結合キー名が左右で異なる場合のみ on を使う。
* sqlfluffには無いルールのため、カスタムルールを作成して対応が必要。
where 句


-- ✅ 標準：where の後に条件、2つ目以降はインデントして and/or を先頭
where dt = date_sub(current_date('Asia/Tokyo'), interval 1 day)
    and status = 'active'
-- ❌ 避ける
where dt = date_sub(current_date('Asia/Tokyo'), interval 1 day) and
    status = 'active'
CTE と import CTE パターン


-- ✅ 標準：最初の CTE は with と同一行、2つ目以降は新しい行で先頭カンマ
with source as (
    〜
)
, renamed as (
    〜
)
select * from renamed
* インラインサブクエリよりも CTE で段階分割する。
* sqlfluffには無いルールのため、カスタムルールを作成して対応が必要。

テスト
✅ 標準（最低ライン）
* marts の主キーに unique ＋ not_null を必須。
* テスト記法は data_tests:（新記法）に統一（旧 tests: は使わない）。
* 複雑な変換ロジックには unit test を推奨。


# _marts__models.yml
models:
  - name: dim_companies
    columns:
      - name: company_id
        data_tests:
          - unique
          - not_null

ドキュメント
✅ 標準
* 全モデル・全カラムに日本語 description を付ける。
* yml のファイル方式は 1つに統一する（下記いずれか。テンプレリポジトリで固定）。
    * 案A: _<table>__models.yml（テーブル単位。参照実装が採用）
    * 案B: _<model>.yml サイドカー（dbt-osmosis 運用。bpaas 系 / ocr-…-dwh-a が採用）

共通マクロ・パッケージ
✅ 標準
* 各リポジトリにほぼ同一のマクロが散在する場合 社内 dbt package に集約して共有する。
    * freshness_query（source 鮮度）
    * create_shared_views（on-run-end で共有ビュー生成）
    * dev_source（本番/dev のソース切替）
    * snapshot_dt（スナップショット断面のピン留め）
* 外部パッケージは dbt_utils を標準採用。品質担保に dbt_project_evaluator の導入を推奨。

環境切替
✅ 標準
* 本番/dev やプロジェクト切替は env_var() に方式を統一する。
* target.name の文字列分岐や、リポジトリ固有の環境名分岐（例: harbor）を各様に増やさない。

スナップショット / 断面の扱い
✅ 標準
* 日次パーティション断面は JST 前日 を共通イディオムにする。
* ただし、一部のデータソース(Salesforce)において特殊な理由がある場合は除外。
* martsで作成したテーブルが対象


where dt = date_sub(current_date('Asia/Tokyo'), interval 1 day)
* 履歴保持（SCD Type2）は dbt の snapshot 機能を使う。

付録A：.sqlfluff スターター設定
各リポジトリのルートに配置し、CI（sqlfluff lint）で強制する。第6章のルールに対応。


[sqlfluff]
dialect = bigquery
templater = dbt
max_line_length = 120
[sqlfluff:indentation]
indent_unit = space
tab_space_size = 4
[sqlfluff:layout:type:comma]
line_position = leading            ; 先頭カンマ（6.3）
[sqlfluff:rules:capitalisation.keywords]
capitalisation_policy = lower      ; キーワード小文字（6.1）
[sqlfluff:rules:capitalisation.identifiers]
capitalisation_policy = lower
[sqlfluff:rules:capitalisation.functions]
capitalisation_policy = lower
[sqlfluff:rules:aliasing.table]
aliasing = implicit                ; テーブル別名に as を付けない（6.4）
[sqlfluff:rules:aliasing.column]
aliasing = explicit                ; カラム別名に as を付ける（6.3）
[sqlfluff:rules:references.consistent]
force_enable = True
導入時は sqlfluff fix で一括整形 → 差分レビュー → CI 必須化、の順で。大文字リポジトリ（Y-a）は差分が大きいので単独 PR で。







freeeポイントDB設計 (UC最小構成)

￼
作成者: ryosuke-matsushita

聴く

8

リアクションを追加
￼
Content Report





サンクスレター機能 DB設計 (UC最小構成)
本ドキュメントは、提示されたユースケース（UC-1 / UC-3従業員 / UC-5 / UC-3管理者）のみを満たすことを目的に、元ER図を最小構成へ絞り込んだDB設計です。
employees はfreee人事労務マスタの外部参照とし、本サービス内には実テーブルを作成しません。各テーブルの *_employee_id は人事労務マスタ上の従業員IDへの論理参照です (物理外部キー制約は張りません)。
1. 対象ユースケース



UC	区分	概要
UC-1	従業員	ポイント付きレターを送る (1 to 1, ポイント設定, メッセージ)
UC-3	従業員	送付可能ポイント残数 (今週) を確認する
UC-5	従業員	フィード閲覧 / もらった・送ったフィルタ / 今月もらった / 累積もらった
UC-3	管理者	サンクスレターサービスのメンバー (利用者) と権限を管理する


UC-管理者-3 の解釈について
従業員マスタそのものは人事労務側で管理されるため、本サービスで実体を CRUD しません。本サービスにおける「マスタ管理」は「人事労務に在籍する従業員のうち、誰をこのサービスのメンバーとするか」の管理 (招待・停止・除外) + 権限割当 と解釈します。
2. 設計方針
* 従業員マスタは外部参照: 人事労務側の employee_id を論理参照のみで保持 (本サービスに実テーブルなし)
* サービス利用状態: service_memberships で、人事労務在籍者のうちサンクスレターを使う人を管理 (招待/active/停止/削除)
* 1 to 1限定: letter_recipients 子テーブルは作らず、letters に recipient_employee_id/points を埋め込む
* 残数・集計: サマリテーブルは作らず、point_transactions からの集計で実現 (YAGNI)
* マルチテナント: 全テーブルに company_id を保持しインデックス付与
* 論理削除: service_memberships は deleted_at センチネル運用
* 権限管理: 履歴管理可能な member_roles テーブルで admin/member を区別
* 不変ログ: point_transactions は追記専用、updated_at を持たない
3. ER図


erDiagram
    %% ===== freee人事労務マスタ（外部参照・本PJで管理しない） =====
    companies ||--o{ employees : has
    %% ===== サンクスレターサービス内テーブル =====
    companies ||--o{ service_memberships : scopes
    companies ||--o{ letters : scopes
    companies ||--o{ member_roles : scopes
    companies ||--o{ point_grants : scopes
    companies ||--o{ point_transactions : scopes
    employees ||--o{ service_memberships : "registered (外部参照)"
    employees ||--o{ letters : "sends (sender)"
    employees ||--o{ letters : "receives (recipient)"
    employees ||--o{ member_roles : assigned
    employees ||--o{ point_grants : granted
    employees ||--o{ point_transactions : involves
    letters ||--o{ point_transactions : "origin"
    employees {
        bigint id PK "freee人事労務マスタ・外部参照"
        bigint company_id
    }
    service_memberships {
        bigint id PK
        bigint company_id "テナント"
        bigint employee_id "人事労務マスタへの論理参照"
        smallint status "1:invited/2:active/3:suspended"
        datetime invited_at
        datetime joined_at
        datetime suspended_at
        datetime created_at
        datetime updated_at
    }
    member_roles {
        bigint id PK
        bigint company_id
        bigint employee_id "人事労務マスタへの論理参照"
        smallint role_kind "1:admin/2:member"
        datetime effective_from
        datetime effective_to "null=現在有効"
    }
    letters {
        bigint id PK
        bigint company_id
        bigint sender_employee_id "送り手 (人事労務マスタ参照)"
        bigint recipient_employee_id "受け手 (人事労務マスタ参照, 1to1)"
        int points "送付ポイント(0以上)"
        text message "メッセージ本文"
        datetime created_at
    }
    point_grants {
        bigint id PK
        bigint company_id
        date start_on "週の起算日"
        date end_on "週の終了日(失効日)"
        int granted_points "週内に送れる原資"
        datetime created_at
        datetime updated_at
    }
    point_transactions {
        bigint id PK
        bigint company_id
        bigint subject_employee_id "取引対象 (人事労務マスタ参照)"
        bigint counterpart_employee_id "相手 (人事労務マスタ参照)"
        bigint letter_id FK "起点レター"
        smallint kind "1:送付/2:受領"
        int amount "送付/受領ポイント"
        datetime occurred_at
        datetime created_at
    }
￼
4. テーブル定義
4.1 employees (外部参照・本サービスでは管理しない)
人事労務マスタの employees を参照します。本サービス内には実テーブルを作成せず、各テーブルが *_employee_id として論理参照を保持します。
* 名前・メールアドレス等の従業員属性は表示時に人事労務マスタAPIから取得する想定
* 物理外部キー制約は張らない (テナント越境・サービス越境のため)
* 削除や名前変更は人事労務側で行われ、本サービスは追従するのみ
4.2 service_memberships (UC-管理者-3 サービス利用管理)



カラム	型	備考
id	bigint PK	
company_id	bigint	テナント
employee_id	bigint	人事労務マスタへの論理参照
status	smallint	1:invited / 2:active / 3:suspended
invited_at	datetime	招待日時
joined_at	datetime	初回ログイン (active化) 日時
suspended_at	datetime	停止日時
created_at / updated_at	datetime	


Index: (company_id, employee_id, deleted_at) ユニーク
4.3 member_roles (UC-管理者-3 権限管理)



カラム	型	備考
id	bigint PK	
company_id	bigint	
employee_id	bigint	人事労務マスタへの論理参照
role_kind	smallint	1:admin / 2:member
effective_from / effective_to	datetime	履歴で残す (effective_to IS NULL が現有効)


Index: (company_id, employee_id, effective_to)
4.4 letters (UC-1)



カラム	型	備考
id	bigint PK	
company_id	bigint	
sender_employee_id	bigint	送り手 (人事労務マスタ参照)
recipient_employee_id	bigint	受け手 (人事労務マスタ参照, 1 to 1 のため埋め込み)
points	int	0以上
message	text	
created_at	datetime	


Index:
* (company_id, recipient_employee_id, sent_at) — UC-5「もらった」フィード
* (company_id, sender_employee_id, sent_at) — UC-5「送った」フィード
4.5 point_grants (UC-3 残数算出の原資側)



カラム	型	備考
id	bigint PK	
company_id	bigint	
employee_id	bigint	人事労務マスタへの論理参照
start_on / end_on	date	週単位
granted_points	int	その週に送れる原資
created_at / updated_at	datetime	


Index: (company_id, employee_id, start_on) ユニーク
4.6 point_transactions (UC-3 残数 / UC-5 集計)



カラム	型	備考
id	bigint PK	
company_id	bigint	
subject_employee_id	bigint	自分視点の主体 (人事労務マスタ参照)
counterpart_employee_id	bigint	相手 (人事労務マスタ参照)
letter_id	bigint	
kind	smallint	1:送付 / 2:受領
amount	int	送付/受領ポイント (正値)
occurred_at	datetime	
created_at	datetime	不変ログのため updated_at なし


Index: (company_id, subject_employee_id, kind, occurred_at) — 残数・今月集計・累積集計に使用
5. UC ↔ クエリの対応



UC	解決方法
UC-1 送付	letters 1件 + point_transactions 2件 (送付:sender, 受領:recipient) をトランザクションでINSERT
UC-3 今週残数	point_grants.granted_points - SUM(point_transactions.amount WHERE kind=送付 AND 当週)
UC-5 フィード	letters を sender/recipient で WHERE + sent_at DESC。表示時の氏名は人事労務マスタAPIから取得
UC-5 今月もらった	SUM(point_transactions.amount WHERE subject=自分 AND kind=受領 AND 今月)
UC-5 累積もらった	SUM(point_transactions.amount WHERE subject=自分 AND kind=受領)
UC-管理者-3 メンバー追加	人事労務マスタから検索した employee_id を service_memberships に status=invited でINSERT
UC-管理者-3 編集	service_memberships.status 更新、member_roles に新規行追加 (旧行は effective_to セット)
UC-管理者-3 削除	service_memberships.deleted_at をセット (論理削除)。送付済みレターや取引履歴は残る


6. 元ER図から削除したもの (理由)



削除対象	理由
letter_recipients	1 to 1 のため不要、letters に埋め込み
labels / letter_labels	UCにラベル要件なし
reactions / applauses / comments	UC-5はフィード閲覧のみでインタラクション要件なし
company_point_settings / role_point_allowances / role_post_permissions	上限・付与設定はUC範囲外 (point_grantsに付与結果のみ保持)
各種 notification / reminder 系	通知・リマインドはUC範囲外
member_point_summaries / weekly_digests / ranking_snapshots	集計は transactions から直接算出 (必要に応じて後から導入)
letter_exports	エクスポート機能はUC範囲外


7. 人事労務マスタ参照に関する留意点
* 従業員削除への追従: 人事労務側で従業員が削除/退職しても、本サービスの letters・point_transactions は履歴として残る。表示時は人事労務マスタAPIからの取得結果に応じて「(退職済み)」等の表示を行う想定
* 氏名・属性のキャッシュ: パフォーマンス要件によっては表示用に氏名スナップショットを letters へ非正規化する判断もあり得るが、本UC範囲ではAPI直参照とし非正規化を保留
* 不正な参照の検出: 本サービスで service_memberships に存在しない employee_id でレター送信を試みた場合の妥当性検証はアプリケーション層で実施
8. 拡張余地
本設計は UC最小構成 です。以下の機能を追加する際は元ER図のテーブルを参照する想定です。
* 1 to N 送信 → letter_recipients 子テーブルへ分離
* ラベル / リアクション / 拍手 / コメント → 各サブテーブル追加
* ランキング・週次ダイジェスト → 集計用テーブル新設 (パフォーマンス劣化が顕在化したタイミングで)
* 通知 → Delibird連携 + プリファレンステーブル
* 氏名表示のパフォーマンス対策 → 人事労務マスタ参照のキャッシュ層または非正規化



設計原則
* マルチテナント（company_id で完全隔離）
* FK制約禁止ポリシー（論理参照のみ）
* 新規テーブルのPKは bigint unsigned
* 他テーブルのIDを参照するカラム（company_id, employee_id など）は bigint で統一
* 主要クエリが単一テーブル＋インデックスで完結すること
* 「送るポイント」と「もらったポイント」は別概念。スキーマレベルで分離する
テーブル構成（6テーブル）



テーブル名	説明
point_company_settings	テナント（事業所）ごとの設定
point_memberships	サービス利用登録＋ロール
point_weekly_grants	「送るポイント」の残高スナップショット（メンバー・週単位）
point_letters	レター本体（不変）
point_sendable_logs	「送るポイント」の変化ログ（付与・送付・失効）
point_received_logs	「もらったポイント」の変化ログ（受領）


2種類のポイントの性質



	送るポイント	もらったポイント
発生源	週次バッチで付与	レター受領
失効	週末に失効	しない
送付に使える	✅	❌
残高スナップショット	point_weekly_grants	ログ集計で導出
ログテーブル	point_sendable_logs	point_received_logs


詳細定義
1. point_company_settings
テナントごとのサービス設定を1行で保持する。サービス開始時にINSERT、変更時はUPDATE。


create_table :point_company_settings,
             id: { type: :bigint, unsigned: true },
             comment: 'freee-point テナント設定' do |t|
  t.bigint  :company_id, null: false,
            comment: '事業所ID（companies.id への論理参照）'
  t.integer :weekly_grant_points, null: false, default: 100,
            comment: '1週間に各メンバーへ付与する送るポイント数'
  t.integer :grant_day_of_week, limit: 2, null: false, default: 1,
            comment: '付与曜日（0:日 1:月 2:火 3:水 4:木 5:金 6:土）'
  t.timestamps
  t.index :company_id, unique: true,
          name: 'idx_point_company_settings_company_id'
end
2. point_memberships
誰がこのサービスのメンバーか、そのロールを1テーブルで管理する。ロールの変更頻度は極めて低いため temporal pattern は不要。変更履歴が必要になれば v2 以降で history テーブルを追加する。


create_table :point_memberships,
             id: { type: :bigint, unsigned: true },
             comment: 'freee-point サービス利用登録（ロール含む）' do |t|
  t.bigint  :company_id,  null: false,
            comment: '事業所ID（companies.id への論理参照）'
  t.bigint  :employee_id, null: false,
            comment: '従業員ID（employees.id への論理参照）'
  t.integer :role,   limit: 2, null: false, default: 2,
            comment: 'ロール（1:admin / 2:general）'
  t.integer :status, limit: 2, null: false, default: 1,
            comment: '利用状態（1:invited / 2:active / 3:suspended）'
  t.datetime :invited_at,   null: true, comment: '招待日時'
  t.datetime :joined_at,    null: true, comment: '利用開始日時'
  t.datetime :suspended_at, null: true, comment: '利用停止日時'
  t.timestamps
  t.index %i[company_id employee_id], unique: true,
          name: 'idx_point_memberships_company_employee'
  t.index %i[company_id status],
          name: 'idx_point_memberships_company_status'
  t.index %i[company_id role],
          name: 'idx_point_memberships_company_role'
  t.index :employee_id,
          name: 'idx_point_memberships_employee_id'
end
3. point_weekly_grants
「送るポイント」の残高スナップショット（メンバー・週単位）。「今週あと何ポイント送れるか」を O(1) で返す唯一の正。
* レター送信のたびに remaining_points をデクリメント（楽観的ロックで競合防止）
* 失効バッチが remaining_points = 0, expired_at = now に更新


create_table :point_weekly_grants,
             id: { type: :bigint, unsigned: true },
             comment: 'freee-point 送るポイント残高スナップショット（メンバー・週単位）' do |t|
  t.bigint   :company_id,  null: false, comment: '事業所ID'
  t.bigint   :employee_id, null: false, comment: '従業員ID'
  t.date     :week_start_on, null: false, comment: '週の起算日'
  t.date     :week_end_on,   null: false, comment: '週の終了日＝失効日'
  t.integer  :granted_points,   null: false, comment: '付与された送るポイント（変更不可）'
  t.integer  :remaining_points, null: false, comment: '現在の送付可能残高（送信のたびに減算）'
  t.datetime :expired_at, null: true, comment: '失効処理実行日時（null=まだ失効していない）'
  t.timestamps
  t.index %i[company_id employee_id week_start_on], unique: true,
          name: 'idx_point_weekly_grants_member_week'
  t.index %i[company_id week_start_on],
          name: 'idx_point_weekly_grants_company_week'
  t.index %i[week_end_on expired_at],
          name: 'idx_point_weekly_grants_expiry'
  t.index :employee_id,
          name: 'idx_point_weekly_grants_employee_id'
end
4. point_letters
1対1のサンクスレター本体。送信後は変更不可（updated_at を持たない）。


create_table :point_letters,
             id: { type: :bigint, unsigned: true },
             comment: 'freee-point サンクスレター（不変）' do |t|
  t.bigint  :company_id,            null: false, comment: '事業所ID'
  t.bigint  :sender_employee_id,    null: false, comment: '送り手従業員ID'
  t.bigint  :recipient_employee_id, null: false, comment: '受け手従業員ID'
  t.integer :points, null: false, comment: '送付ポイント（正の整数）'
  t.text    :message, null: false,  comment: 'メッセージ本文'
  t.datetime :created_at, null: false  # updated_at は持たない（不変）
  t.index %i[company_id created_at],
          name: 'idx_point_letters_company_created_at'
  t.index %i[company_id recipient_employee_id created_at],
          name: 'idx_point_letters_company_recipient_created_at'
  t.index %i[company_id sender_employee_id created_at],
          name: 'idx_point_letters_company_sender_created_at'
  t.index :sender_employee_id,
          name: 'idx_point_letters_sender_employee_id'
  t.index :recipient_employee_id,
          name: 'idx_point_letters_recipient_employee_id'
end
5. point_sendable_logs
「送るポイント」の変化ログ。付与・送付・失効の3種類のみ。すべてのイベントが weekly_grant_id に紐づく（送るポイントの操作は必ず週次付与に起因する）。


create_table :point_sendable_logs,
             id: { type: :bigint, unsigned: true },
             comment: 'freee-point 送るポイント変化ログ（付与・送付・失効）（不変）' do |t|
  t.bigint   :company_id,      null: false, comment: '事業所ID'
  t.bigint   :employee_id,     null: false, comment: '従業員ID'
  t.integer  :kind, limit: 2,  null: false,
             comment: 'イベント種別（1:grant 週次付与 / 2:spend 送付 / 3:expire 失効）'
  t.integer  :points,          null: false, comment: 'ポイント数（常に正の整数）'
  t.bigint   :weekly_grant_id, unsigned: true, null: false,
             comment: 'point_weekly_grants.id への論理参照（全 kind で必須）'
  t.bigint   :letter_id, unsigned: true, null: true,
             comment: 'point_letters.id への論理参照（kind=spend のときのみ）'
  t.datetime :occurred_at, null: false, comment: 'イベント発生日時'
  t.datetime :created_at,  null: false  # updated_at は持たない（不変）
  t.index %i[company_id employee_id occurred_at],
          name: 'idx_point_sendable_logs_employee_occurred'
  t.index :weekly_grant_id, name: 'idx_point_sendable_logs_weekly_grant_id'
  t.index :letter_id,       name: 'idx_point_sendable_logs_letter_id'
  t.index :employee_id,     name: 'idx_point_sendable_logs_employee_id'
end
6. point_received_logs
「もらったポイント」の変化ログ。v1ではレター受領のみ。kind カラムを持たない（このテーブルに入るイベントは受領のみであり、失効は構造上存在しない）。
v2 以降でレター以外の受領イベント（管理者付与・拍手・外部連携）が発生する際は、letter_id を nullable にして source_type / source_id カラムを追加するマイグレーションで対応する。


create_table :point_received_logs,
             id: { type: :bigint, unsigned: true },
             comment: 'freee-point もらったポイント変化ログ（レター受領）（不変）' do |t|
  t.bigint   :company_id,  null: false, comment: '事業所ID'
  t.bigint   :employee_id, null: false, comment: '受領した従業員ID'
  t.integer  :points,      null: false, comment: 'ポイント数（常に正の整数）'
  t.bigint   :letter_id, unsigned: true, null: false,
             comment: 'point_letters.id への論理参照'
  t.datetime :occurred_at, null: false, comment: 'イベント発生日時'
  t.datetime :created_at,  null: false  # updated_at は持たない（不変）
  t.index %i[company_id employee_id occurred_at],
          name: 'idx_point_received_logs_employee_occurred'
  t.index :letter_id, unique: true, name: 'idx_point_received_logs_letter_id'
  t.index :employee_id, name: 'idx_point_received_logs_employee_id'
end
主要クエリとインデックスの対応



#	ユースケース	テーブル	使用インデックス
Q1	今週の送付可能残高	point_weekly_grants	UNIQUE(company_id, employee_id, week_start_on) → O(1)
Q2	全社フィード（降順ページング）	point_letters	(company_id, created_at)
Q3	もらったフィード	point_letters	(company_id, recipient_employee_id, created_at)
Q4	送ったフィード	point_letters	(company_id, sender_employee_id, created_at)
Q5	今月もらったポイント	point_received_logs	(company_id, employee_id, occurred_at) で月範囲
Q6	累積もらったポイント	point_received_logs	(company_id, employee_id, occurred_at) で全件 SUM
Q7	メンバー一覧	point_memberships	(company_id, status)
Q8	権限チェック	point_memberships	UNIQUE(company_id, employee_id) → O(1)
Q9	失効バッチ対象取得	point_weekly_grants	(week_end_on, expired_at)


ユースケース別の操作フロー
レター送信時（DBトランザクション内）
1. point_weekly_grants : remaining_points -= letter.points（楽観的ロックで競合防止）
2. point_letters INSERT
3. point_sendable_logs INSERT（kind=2:spend, weekly_grant_id=現在の付与レコード）
4. point_received_logs INSERT（letter_id=新レター）
週次付与バッチ（week_start_on の朝）
1. 全 active メンバーに対して point_weekly_grants INSERT
2. point_sendable_logs INSERT（kind=1:grant）
失効バッチ（week_end_on の夜）
1. 対象 point_weekly_grants（week_end_on < today AND expired_at IS NULL）を取得
2. remaining_points > 0 のレコードに対して point_sendable_logs INSERT（kind=3:expire, points=remaining_points）
3. point_weekly_grants UPDATE：remaining_points = 0, expired_at = now
設計判断まとめ

判断事項	採用方針	理由
判断事項	採用方針	理由
2種類のポイントの分離	テーブルを分割（sendable_logs / received_logs）	v1から存在するドメインルール。1テーブルにすると全クエリに kind フィルタが必要になり、スキーマが概念を正しく表現できていないサインとなる
ロール管理	point_memberships.role に統合	変更頻度が極めて低い。履歴が必要なら将来 history テーブルを追加
送るポイントの残高管理	point_weekly_grants.remaining_points を保持	レター送信前に「残高 ≥ 送るポイント数」をリアルタイムで確認し、原子的にデクリメントする必要があるためスナップショットが必要
もらったポイントに残高スナップショットを持たない	point_received_logs の集計で導出	v1ではもらったポイントを消費する操作が存在しないため、送信前の残高チェックが発生しない。「今月もらった」「累積もらった」はSUMで十分。v2でギフト交換が加わった時点で残高スナップショットテーブルを追加する
received_logs の source_type	v1 では追加しない	v1→v2 移行は速く DB の変化は容認。実際の要件が確定してから追加する（YAGNI）
point_transactions を使わない	sendable_logs / received_logs に分割	1テーブルで2種類のポイントを混在させると、weekly_grant_id が receive レコードで常に null になるなどカラムの意味が不整合になる
tinyint(1) を boolean 以外で使わない	role / status / kind / grant_day_of_week は limit: 2（smallint）を使用	DBガイドライン準拠。データ基盤側で tinyint(1) は boolean として取り込まれるため、列挙値に使うと集計時に問題が起きる






freeeポイント(仮)概念設計v0.4



作成者: ryosuke-matsushita

聴く

4

リアクションを追加
アプリ アイコンContent Report
freeeポイント(仮) 概念設計 v0.4
本ドキュメントの位置づけ

感謝メッセージ＋ピアボーナス型サービス「freeeポイント(仮)」の概念設計です。テーブル設計、設計思想、freee DBガイドラインへの準拠ポイントをまとめています。

v0.4 では、週次ダイジェストで「誰に送ったか・誰からもらったか」を表示する要件に応えるため、子テーブル weekly_digest_partners を新設しました。

1. 全体像
本サービスは 8つのドメイン で構成されます。コアは「レター」と「ポイント取引」で、それ以外は周辺機能です。

#

ドメイン

主要テーブル

一言で

①

外部マスタ

employees / companies / departments / department_memberships

freee人事労務から参照（本PJでは作らない）

②

コア：レター

letters / letter_recipients

感謝メッセージ本体。下書きは sent_at IS NULL

③

ラベル（バリュー）

labels / letter_labels

会社の行動指針をタグ付け

④

ポイント取引

point_transactions

不変ログ。kindで送付/受領/付与/取消を表現

⑤

インタラクション

reactions / applauses / comments

スタンプ/拍手(ポイント可)/コメント

⑥

ポイント経済の制御

company_point_settings / role_point_allowances / weekly_point_grants

付与・上限・原資の3層管理

⑦

権限・組織モニタリング

service_memberships / member_roles / role_post_permissions

利用可否、ロール、投稿権限

⑧

通知・リマインド・集計

member_notification_settings / company_notification_settings / member_point_summaries / weekly_digests / weekly_digest_partners / ranking_snapshots / letter_exports

Delibird に委譲しつつ、freeeポイント固有のプリファレンス・集計を保持

2. ER図


erDiagram
    %% ===== freee人事労務マスタ（外部参照・本PJで管理しない） =====
    %% マネージャー配下関係は departments / department_memberships を直接参照する
    companies ||--o{ employees : has
    companies ||--o{ departments : has
    departments ||--o{ department_memberships : has
    employees ||--o{ department_memberships : belongs_to
    %% ===== コア：レター =====
    employees ||--o{ letters : "sends (sender)"
    companies ||--o{ letters : scopes
    letters ||--o{ letter_recipients : "addressed_to"
    employees ||--o{ letter_recipients : "receives"
    letters {
        bigint id PK
        bigint company_id "テナント"
        bigint sender_employee_id "送り手"
        text message "感謝メッセージ本文"
        datetime sent_at "送信日時"
        datetime created_at
        datetime updated_at
    }
    letter_recipients {
        bigint id PK
        bigint company_id
        bigint letter_id FK
        bigint recipient_employee_id "受け手"
        int points "この受け手に渡るポイント(0以上)"
        datetime created_at
        datetime updated_at
    }
    %% ===== ラベル（バリュー） =====
    companies ||--o{ labels : defines
    labels {
        bigint id PK
        bigint company_id
        string name "ラベル名"
        datetime created_at
        datetime updated_at
    }
    letters ||--o{ letter_labels : tagged_with
    labels ||--o{ letter_labels : applied_to
    letter_labels {
        bigint id PK
        bigint company_id
        bigint letter_id FK
        bigint label_id FK
        datetime created_at
    }
    %% ===== ポイント取引（不変ログ） =====
    employees ||--o{ point_transactions : involves
    letters ||--o{ point_transactions : "origin (nullable)"
    letter_recipients ||--o{ point_transactions : "origin (nullable)"
    point_transactions {
        bigint id PK
        bigint company_id
        bigint subject_employee_id "取引対象(送り手or受け手)"
        bigint counterpart_employee_id "相手(null可)"
        bigint letter_id FK "起点レター(null可)"
        bigint letter_recipient_id FK "起点受信者レコード(null可)"
        smallint kind "1:送付/2:受領/3:週次付与/4:管理者個別付与/5:拍手贈与/6:拍手受領/7:送付取消/8:受領取消/9:拍手贈与取消/10:拍手受領取消"
        int amount "正:加算,負:減算"
        datetime occurred_at
        datetime created_at
    }
    %% ===== インタラクション =====
    letters ||--o{ reactions : has
    employees ||--o{ reactions : posts
    reactions {
        bigint id PK
        bigint company_id
        bigint letter_id FK
        bigint employee_id "リアクション主"
        string emoji_code "スタンプ種別"
        datetime created_at
    }
    letters ||--o{ applauses : has
    employees ||--o{ applauses : posts
    applauses {
        bigint id PK
        bigint company_id
        bigint letter_id FK
        bigint employee_id "拍手者"
        int points "贈与ポイント(0可)"
        datetime created_at
        datetime updated_at
    }
    letters ||--o{ comments : has
    employees ||--o{ comments : posts
    comments {
        bigint id PK
        bigint company_id
        bigint letter_id FK
        bigint employee_id "コメント主"
        text body
        datetime created_at
        datetime updated_at
    }
    %% ===== ポイント原資・上限 =====
    companies ||--|| company_point_settings : configures
    company_point_settings {
        bigint id PK
        bigint company_id
        smallint reset_monthday "1〜31: 月リセットの日付"
        smallint reset_weekday "0:日〜6:土 週リセット曜日"
        string timezone "Asia/Tokyo等"
        int monthly_grant_amount "月次付与ポイント数(一律)"
        int weekly_grant_amount "週次付与ポイント数(一律)"
        datetime effective_from
        datetime effective_to
    }
    companies ||--o{ role_point_allowances : configures
    role_point_allowances {
        bigint id PK
        bigint company_id
        bigint position_type_id "Ohmu PositionType の外部参照ID"
        int monthly_max_points "月次の送付上限"
        int weekly_max_points "週次の送付上限"
        datetime effective_from
        datetime effective_to
    }
    employees ||--o{ weekly_point_grants : granted
    point_grants {
        bigint id PK
        bigint company_id
        date start_on "起算日(事業所設定の曜日)"
        date end_on "終了日(失効日)"
        int granted_points "期間中送れる原資"
        datetime created_at
    }
    %% ===== 権限・組織モニタリング =====
    %% マネージャー配下関係は freee人事労務の departments / department_memberships を直接参照
    employees ||--o{ service_memberships : registered
    companies ||--o{ service_memberships : scopes
    service_memberships {
        bigint id PK
        bigint company_id
        bigint employee_id "Ohmu PositionType の外部参照ID"
        smallint status "1:invited/2:active/3:suspended"
        datetime invited_at
        datetime joined_at
        datetime suspended_at
        datetime created_at
        datetime updated_at
    }
    service_memberships ||--o{ member_roles : has
    employees ||--o{ member_roles : assigned
    companies ||--o{ member_roles : scopes
    member_roles {
        bigint id PK
        bigint company_id
        bigint employee_id
        smallint role_kind "1:admin/2:manager/3:member"
        datetime effective_from
        datetime effective_to
        datetime created_at
        datetime updated_at
    }
    companies ||--o{ role_post_permissions : configures
    role_post_permissions {
        bigint id PK
        bigint company_id
        smallint role_kind "1:admin/2:manager/3:member"
        boolean can_send_letter
        boolean can_send_with_points
        int per_letter_max_points "1レターあたりの上限"
        int per_recipient_weekly_max_points "同一人物への週次上限(不正利用防止)"
        datetime updated_at
    }
    %% ===== 通知 =====
    %% 通知配信は Delibird マイクロサービスに一元化。
    %%   - Slack/Teams/LINE WORKS/モバイル/メール への配信・トークン管理は Delibird 側
    %%   - freee-payroll は DelibirdClient 経由で gRPC リクエストするのみ
    %% 本サービスが持つのは
    %%   - イベント単位の通知プリファレンス（受信/リアクション/拍手/リマインド）
    %%   - リマインドの発火スケジュール設定
    %% に限定する。
    employees ||--o{ member_notification_settings : has
    member_notification_settings {
        bigint id PK
        bigint company_id
        bigint employee_id
        smallint event_kind "1:受信/2:リアクション/3:拍手/4:リマインド"
        boolean enabled
        datetime updated_at
    }
    companies ||--o{ company_notification_settings : configures
    company_notification_settings {
        bigint id PK
        bigint company_id
        smallint event_kind "1:受信/2:リアクション/3:拍手/4:リマインド"
        boolean enabled
        datetime updated_at
    }
    %% ===== リマインド =====
    companies ||--o{ auto_reminder_settings : configures
    auto_reminder_settings {
        bigint id PK
        bigint company_id
        string schedule_cron "発火スケジュール"
        smallint target_role_kind
        boolean enabled
        datetime created_at
        datetime updated_at
    }
    employees ||--o{ manual_reminders : "sends/receives"
    manual_reminders {
        bigint id PK
        bigint company_id
        bigint sender_employee_id "管理者orマネージャー"
        bigint recipient_employee_id
        text message
        datetime sent_at
    }
    %% ===== 集計・閲覧支援 =====
    employees ||--o{ member_point_summaries : "summarized_in"
    member_point_summaries {
        bigint id PK
        bigint company_id
        bigint employee_id
        date current_week_start_on "現在週の起算日"
        int weekly_sent_points "今週送付済み合計"
        int weekly_granted_points "今週付与原資"
        int monthly_received_points "今月受領合計"
        bigint cumulative_received_points "累積受領(サービス開始〜)"
        datetime last_synced_at "最終同期時刻"
        datetime created_at
        datetime updated_at
    }
    %% ===== 週次ダイジェスト =====
    %% v0.4 で weekly_digest_partners 子テーブルを新設。
    %% 「誰に送ったか・誰からもらったか」をレター単位の明細として持つ
    %% （同一相手が複数回登場し得る = 重複あり）。
    employees ||--o{ weekly_digests : "owned_by"
    weekly_digests {
        bigint id PK
        bigint company_id
        bigint employee_id
        date week_start_on
        int sent_points "週内に送ったポイント合計"
        int received_points "週内にもらったポイント合計"
        int sent_letter_count "週内に送ったレター件数"
        int received_letter_count "週内にもらったレター件数"
        datetime created_at
    }
    companies ||--o{ ranking_snapshots : computed
    ranking_snapshots {
        bigint id PK
        bigint company_id
        smallint period_kind "1:weekly/2:monthly/3:all_time"
        smallint ranking_kind "1:point/2:letter"
        date period_start_on
        date period_end_on
        bigint employee_id
        int rank
        int value
        datetime computed_at
    }
    companies ||--o{ letter_exports : requests
    letter_exports {
      bigint id PK
      bigint company_id
      bigint requested_by_employee_id
      smallint format_kind "1:CSV/2:PDF"
      date period_start_on
      date period_end_on
      smallint status "1:enqueued/2:working/3:complete/4:failed"
      string s3_filename "S3オブジェクトキー（ダウンロード時に署名URLを動的生成）"
      string filename "ダウンロード時の表示ファイル名"
      text error_message "failed 時のエラー詳細（nullable）"
      datetime created_at
      datetime updated_at
    }
3. 主要ドメインの解説
3.1 コア：レター（letters / letter_recipients）
感謝メッセージのドメイン中心です。1対N構造により、1通のレターを複数の受け手に同時送付できます。

letters: 1通のレター。sender_employee_id（送り手）と company_id（テナント）で送信元を特定

letter_recipients: 1レターに対する受け手をN件保持。各受け手に個別の points 設定可能

下書き対応: sent_at IS NULL で下書き状態を表現

3.2 ポイント取引（point_transactions）— 不変ログ
本設計の最大の特徴です。会計の複式簿記をイベントソーシングで実装しています。

複式記録: 1回のレター送信で最低2レコード生成（送り手 kind=1 amount=-N、受け手 kind=2 amount=+N）

不変性: UPDATE/DELETE 禁止。取消は逆仕訳レコードを追加（kind=7〜10）

残高計算: SUM(amount) WHERE subject_employee_id = ? AND company_id = ?

レター起点以外も表現可能: letter_id / letter_recipient_id は nullable

運用上の注意

テーブルが青天井に成長します。(company_id, subject_employee_id, occurred_at) の複合インデックスとパーティショニング戦略が必須です。残高参照は member_point_summaries をキャッシュとして使い、生クエリは read replica に向けます。

3.3 インタラクション（reactions / applauses / comments）
性質が異なる3種のインタラクションを別テーブルで分離。

reactions: 軽量・大量発生、UPDATE不要。UNIQUE (letter_id, employee_id, emoji_code) を推奨

applauses: ポイント贈与を伴う「重い」アクション。1人1レター1回制限が MVP では無難

comments: 編集可能（updated_at あり）。ハードデリート前提

3.4 ポイント原資・上限（3層構造）
ポイント経済を3層で制御します。

company_point_settings: 事業所単位の基本設定（週リセット曜日、TZ、付与額）

role_point_allowances: ロール別週次送付上限（admin/manager/member）。effective_from で履歴管理

weekly_point_grants: 個人別週次原資のスナップショット。失効日（week_end_on）で繰り越し不可を明示

3.5 権限・組織モニタリング
v0.2での変更点

当初 manager_subordinates テーブルでマネージャー配下関係をキャッシュする設計でしたが、freee人事労務の departments / department_memberships を直接参照する方針に変更しました。

service_memberships: サービス利用ステータス（invited / active / suspended）

member_roles: ロール割当（effective_from / effective_to で履歴管理）

role_post_permissions: ロール別投稿権限。per_recipient_weekly_max_points で自作自演・馴れ合いを防止

変更理由:

同期コスト不要（組織変更が即時反映）

Single Source of Truth の維持

freee社内の他サービスとの整合性

性能要件で問題が出た時点でキャッシュテーブル導入を再検討可能

3.6 通知 — Delibird への委譲（v0.3で整理）
freee人事労務における通知配信は Delibird マイクロサービスに一元化されています。Slack DM／Slack チャンネル／Teams／LINE WORKS／モバイルプッシュ（FCM）／メールへの配信、各プロバイダのトークン管理、配信ステータス管理はすべて Delibird 側で完結します。

このため本サービスでは、配信に関するインフラ機能は一切自前で持ちません。

観点

Delibird 側

freeeポイント側

Slack/Teams のトークン保持

○

×

既定チャンネル設定

○

×

配信ステータス（queued/sent/failed）

○

×

イベントごとの ON/OFF プリファレンス

×

○ (member_notification_settings / company_notification_settings)

リマインドの発火スケジュール

×

○ (auto_reminder_settings)

v0.2 からの整理

external_integrations テーブルを削除：Slack/Teams のトークン・既定チャンネル管理は Delibird に完全委譲。freee-payroll 側は list_slack_settings / check_teams_setting_exists で参照するのみ。

notification_deliveries テーブルを削除：配信ステータス管理は Delibird 側で完結する。ローカルに持つと真実の二重化になる。

各 *_notification_settings / auto_reminder_settings から channel_kind カラムを除去：チャンネル別の通知可否は Delibird のユーザー設定に委ね、本サービスは「どのイベントを通知するか」のプリファレンスに責務を絞る。

参考実装:

/lib/delibird_client.rb（gRPC Create 通知、ULID 生成、バッチ送信）

/app/controllers/api/internal/delibird_controller.rb

/app/models/delibird_user_time_clock_message_setting.rb

3.7 週次ダイジェスト（v0.4で拡張）
表示要件

週次ダイジェスト画面では以下を従業員に提示します。

今週送ったポイント合計 ／ もらったポイント合計

今週送ったレター件数 ／ もらったレター件数

今週誰に送ったか ／ 今週誰からもらったか（時系列）

テーブル分割の考え方

集計値（合計ポイント・件数）と明細（誰宛・誰から）は性質が異なるため、設計思想 4.3「読み取りはキャッシュテーブルへ分離」に揃えて 2 テーブルで管理します。

weekly_digests: 1ユーザー × 1週で1行。サマリー指標（合計値）のみ

weekly_digest_partners: 1ユーザー × 1週内のレター単位で 1行ずつ。重複あり（同一相手が複数回登場し得る）

weekly_digest_partners の方針

direction で sent_to ／ received_from を区別。送付方向と受領方向を 1 テーブルに集約することで、UI 側で「タイムライン表示」が単純な ORDER BY で実現できる。

letter_id / letter_recipient_id を保持し、明細クリックでレター本体に遷移可能にする。

points は当該レターでこの相手に動いた個別ポイント値（letter_recipients.points または applauses.points を写経）。

上位N件に絞らず全件を保持。N件絞り込みは UI 側でのページネーション／フィルタリングで吸収する。

occurred_at は letters.sent_at または applauses.created_at をコピー。並び替え専用カラム。

生成タイミング

週次バッチが週末締めで両テーブルを同時に生成。weekly_digests 1 行 INSERT → weekly_digest_partners 複数行 INSERT を 1 トランザクションで実施。

「重複あり」設計の意味

ユーザー A がユーザー B に同一週で 3 通レターを送った場合、weekly_digest_partners には direction=sent_to の B に紐づく行が 3 行できます。相手別の合計を出したい場合は SQL の GROUP BY で集約します。事前に相手別集計を持たないのは、

拍手・コメントなど後付け要件が出たときに「行を追加するだけ」で済む

レター単位の明細クリック導線が自然に作れる

同一週で送・受両方向が発生したユーザーも 2 種の direction で素直に表現できる

ため。

4. 設計思想
4.1 ポイントは複式簿記 × イベントログで扱う
残高カラムを持たず、point_transactions の SUM を真実とする。残高がズレるバグが構造的に発生しない。

4.2 設定変更は履歴型で持つ
effective_from / effective_to 付きのテーブル群（role_point_allowances、member_roles）で、過去時点の運用ルールで判定可能にする。

4.3 読み取りはキャッシュテーブルへ分離
point_transactions は書き込み専用のログ、読み取りは member_point_summaries / weekly_digests (+ weekly_digest_partners) / ranking_snapshots に事前集計を保存。CQRS 的な役割分担。サマリーと明細は別テーブルに分け、性質に合わせた粒度で持つ。

4.4 通知インフラは freee 標準（Delibird）に寄せる
通知配信は freee 標準の Delibird マイクロサービスに完全に委譲し、本サービスは「いつ、誰に対して、どのイベントを送るか」というドメイン知識のみを持つ。Slack/Teams トークンや配信ステータスといったインフラ寄りの情報は持たない。

5. freee DBガイドライン準拠ポイント
項目

対応

外部キー制約

使用しない（pt-osc 非対応のため）。ER図の関係線は論理表現のみ

tinyint(1) はboolean専用

kind / status / role_kind / direction 等は smallint（limit: 2）で実装

ENUM型

使用しない。smallint + コメントで列挙値を表現

論理削除

使用しない。物理削除またはアーカイブテーブル方式

company_id

全テーブルに付与（LeakCheckable によるテナント漏洩防止）

unsigned bigint

user_id / company_id は unsigned bigint で nest-auth と整合

charset / collation

migration で明示指定（utf8mb4, utf8mb4_general_ci）

read replica活用

残高計算・ランキング集計・週次ダイジェスト表示は Aurora reader へ向ける

データ削除戦略

point_transactions / ranking_snapshots / weekly_digest_partners は時系列で増えるためパーティショニング前提

INSERT ... ON DUPLICATE KEY

gap lock 回避のため SELECT → 分岐 INSERT/UPDATE で実装

通知の自前実装

行わない。配信は Delibird に委譲し、本サービスはプリファレンスのみ保持

6. インデックス設計の指針
テーブル

推奨インデックス

用途

point_transactions

(company_id, subject_employee_id, occurred_at)

個人別残高計算・期間集計

point_transactions

(company_id, subject_employee_id, kind, occurred_at)

kind別集計（週次付与のみ等）

letter_recipients

(company_id, recipient_employee_id, created_at)

受信履歴の時系列取得

letters

(company_id, sender_employee_id, sent_at)

送信履歴の時系列取得

reactions

UNIQUE (letter_id, employee_id, emoji_code)

重複防止・トグル動作

weekly_point_grants

(company_id, employee_id, week_start_on)

週次原資の検索

member_roles

(company_id, employee_id, effective_from, effective_to)

有効ロールの判定

weekly_digests

UNIQUE (company_id, employee_id, week_start_on)

ユーザー×週で一意

weekly_digest_partners

(company_id, weekly_digest_id, direction, occurred_at)

ダイジェスト内の方向別時系列表示

weekly_digest_partners

(company_id, weekly_digest_id, partner_employee_id)

相手別 GROUP BY 集計

インデックス設計の原則（DBガイドラインより）

範囲検索・order by するカラムは複合 index の最後に配置

cardinality の低いカラム単独の index は避ける（kind / direction など）

redundant index を作らない

covering index を狙えるなら積極的に活用

7. 残課題・今後の判断ポイント
項目

論点

判断時期

applauses の重複可否

1人1レター1回 vs 何度でも可（クラップ型）

仕様確定時

point_transactions.applause_id 追加

拍手起点取引の逆引き経路

applauses 仕様確定時

effective_to の有無統一

履歴管理テーブル間で運用ルールがばらつく

実装前

company_point_settings の履歴型化

付与額の遡及変更を許すか

仕様確定時

weekly_grant_amount のロール別化

「正社員100pt、契約社員50pt」要件への対応

仕様確定時

manager_subordinates の再導入

EmployeeMaster 直参照のクエリ性能次第

MVP リリース後計測

point_transactions パーティショニング戦略

パーティションキー設計（occurred_at / company_id）

実装前

Delibird 配信結果の表示要否

「Slackに送れた／失敗した」をUIで見せる場合は Delibird API 連携 or イベント購読を検討

UI要件確定時

weekly_digest_partners に拍手・コメントも含めるか

「誰に送ったか／誰からもらったか」を拡張し「誰に拍手したか」等まで含めるか

UI要件確定時

weekly_digest_partners の取消反映

取消後のダイジェストに行を残すか／フラグ立てか／物理削除か

取消仕様確定時

8. Revision
version

date

主な変更点

v0.1

—

初版（コアスキーマ策定）

v0.2

2026-05-19

manager_subordinates 削除（EmployeeMaster 直接参照へ）／ tinyint → smallint（DBガイドライン準拠）／ 設計思想・ガイドライン準拠ポイントを明文化

v0.3

2026-05-19

通知系を Delibird に委譲：external_integrations / notification_deliveries を削除、各 *_notification_settings / auto_reminder_settings から channel_kind を除去、3.6 節と設計思想 4.4 を新設

v0.4

2026-05-19

週次ダイジェストに weekly_digest_partners 子テーブルを新設（誰に送ったか・誰からもらったかをレター単位で重複あり全件保持）。3.7 節を新設、インデックス指針・残課題を追記




スキーマ設計 ver0



作成者: ryosuke-matsushita

聴く

3

リアクションを追加
アプリ アイコンContent Report
概念設計 — freeeポイント(仮)
物理テーブル設計の前段として、4本のPRD（ポイント付きレター送信 / タイムライン / ポイント確認画面 / メンバーのポイント状況確認）と「管理者・マネージャーは従業員マスタを管理できる」要件から、業務上のエンティティを論理名で整理する。テーブル名・カラム型は出さず業務用語で記述する。物理設計は別ページ「freeeポイント(仮) DB設計 - rio」を参照。

1. エンティティ一覧
1-1. 中核エンティティ（業務の主体・客体）
エンティティ

役割

PRD出典

事業所

サービス利用組織。すべての操作のスコープ

全PRD

メンバー

事業所に所属する人。送り手・受け手・閲覧者・管理対象はすべてメンバーの一面

全PRD

部門

事業所内の組織単位。フィルタ・配下定義の基礎

PRD2, PRD4

ロール

管理者 / マネージャー / 一般メンバー の3種

PRD4, 追加要求

1-2. 行為・イベントエンティティ
エンティティ

役割

PRD出典

レター

1人の送り手 → 1人の受け手 への感謝の単位（メッセージ + ポイント数 + 送信日時）

PRD1, PRD2

ポイント付与

事業所からメンバーへポイントが配布されるイベント

PRD3, PRD4

「ポイント送付」「ポイント受領」はレターの一側面なので独立エンティティ化しない（レターを介してポイントが動くため）。

1-3. 状態・残高エンティティ
エンティティ

役割

PRD出典

送付ポイント残高

「今週送れるポイント」の現在値。期間とともに失効する

PRD1, PRD3, PRD4

ポイント有効期限ポリシー

残高の失効周期（事業所単位の設定）

PRD4

1-4. 関係性エンティティ
エンティティ

関連するモノ

PRD出典

補足

部門所属

メンバー × 部門

PRD2, PRD4

多対多

部門管理関係

マネージャー（メンバー）× 管理対象部門

PRD4

「配下メンバー」の根拠

ロール付与

メンバー × ロール

PRD4, 追加要求

「誰が管理者か」の根拠

1-5. 派生・集計エンティティ（履歴から算出可能）
エンティティ

算出元

PRD出典

送付サマリ

レター（送り手側を期間で集計）

PRD4

受領サマリ

レター（受け手側を期間で集計）

PRD3, PRD4

派生エンティティはレター履歴から都度SUMで計算可能。性能要件次第でキャッシュ化する。

1-6. 値オブジェクト（独立したエンティティではない属性）
感謝メッセージ（レターの属性）

送付ポイント数（レターの属性）

送信日時（レターの属性）

集計期間（週次 / 月次 / 累積 / 特定期間 / 有効期限前）

2. エンティティ関連図


erDiagram
    事業所 ||--o{ メンバー : "所属する"
    事業所 ||--o{ 部門 : "保有する"
    事業所 ||--|| ポイント有効期限ポリシー : "定める"
    メンバー }o--o{ 部門 : "部門所属"
    メンバー ||--o{ ロール付与 : "持つ"
    ロール付与 }o--|| ロール : "種別"
    メンバー ||--o{ レター : "送る（送り手）"
    メンバー ||--o{ レター : "受け取る（受け手）"
    ポイント有効期限ポリシー ||--o{ ポイント付与 : "周期で発生"
    メンバー ||--o{ ポイント付与 : "受け取る"
    ポイント付与 ||--|{ 送付ポイント残高 : "形成する"
    レター }o--|| 送付ポイント残高 : "消費する"
    メンバー }o--o{ 部門 : "部門管理関係（マネージャー→管理対象）"
    レター }o..o{ 送付サマリ : "集計対象（派生）"
    レター }o..o{ 受領サマリ : "集計対象（派生）"
凡例: 実線 = 業務上必須の関係、点線 = 派生（履歴から計算可能で物理化はオプション）

3. ロールと「できること」
ロール

レター送信

自分のサマリ閲覧

全メンバーのサマリ閲覧

配下メンバーのサマリ閲覧

従業員マスタ管理

一般メンバー

✓

✓

–

–

–

マネージャー

✓

✓

–

✓

✓

管理者

✓

✓

✓

✓

✓

「マネージャー」を独立エンティティではなく「部門管理関係を1件以上持つメンバー」として導出すれば、ロールは2種（管理者・一般）に簡略化できる。freee-payroll 既存方式（bumon_managers_managing_relations）と整合する。

4. 主要な業務不変条件
不変条件

根拠PRD

レターの送り手と受け手は別人である

PRD1 除外項目

レターの送り手と受け手は同一事業所に所属する

PRD1, PRD2 暗黙

送付ポイント残高は0以上である（マイナスにならない）

PRD3 データ整合性

レター送信時、送付ポイント数 ≤ 送付ポイント残高

PRD1 バリエーション

送付ポイント残高は有効期限到来で0にリセットされる

PRD4

マネージャーが見える配下メンバー = マネージャー自身の管理対象部門に所属するメンバー

PRD4

管理者は事業所内の全メンバーが見える

PRD4

5. 概念モデルと物理設計のマッピング
概念エンティティ

物理テーブル

備考

概念エンティティ

物理テーブル

備考

事業所

companies

既存

メンバー

employees

既存

部門

bumons 系

既存

部門所属

emp_bumons 系

既存

部門管理関係

bumon_managers_managing_relations

既存

ロール / ロール付与

NestAuth + permission.yaml

既存（DBには持たない）

レター

freee_point_letters

新規

送付ポイント残高

employee_freee_point_pools

新規

ポイント付与

freee_point_grants

新規

ポイント有効期限ポリシー

（MVP は固定値、freee_point_pools.period_* に埋め込み）

物理化は次フェーズ

送付サマリ / 受領サマリ

（MVPは未物理化、レター集計クエリで対応）

必要なら employee_freee_point_received_summaries を追加

