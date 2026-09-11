
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

