# 設計書: sample_account (Go 移植版)

## 1. 目的とスコープ

C++17 で書かれた `sample_account` CLI（合成日本語アカウントレコードの CSV ジェネレータ）を Go へ移植する。同等の機能を維持したうえで、以下の追加要件を満たす：

1. **並行化**: 行間で独立な処理を `runtime.NumCPU()` 規模で並列実行し、CPU を最大限に使う
2. **Nix 完結環境**: ツールチェーン・ビルド・テスト・リントを `flake.nix` だけで再現
3. **C++ 比で圧倒的なスピード**: アルゴリズム・I/O・並列の三方向で詰める

オリジナル C++ 実装は `/Volumes/dev/src/cpp/work/sample_account/` を参照。

---

## 2. 機能要件 (C++ 版踏襲)

| 項目 | 仕様 |
|---|---|
| 列定義 | 17 種類（id, lastname, firstname, mail, telephone, prefecture, ward, city, gender, blood, age, agegroup, birthyear, reward, date, random, quotient） |
| 出力順 | CLI フラグの**出現順**で列順が決まる（getopt 互換） |
| 短形式クラスタ | `-ilfm` のように 1 引数で複数列を指定可能 |
| 長形式 | `--lastname` 等。`--telehpne` は `--telephone` のレガシー別名（保持） |
| COUNT | 末尾の正の整数として渡す。既定 100 |
| デフォルト出力 | フラグ無しなら id のみ |
| 入力データ | CSV 4 ファイル（person 10K 行、prefectures 47 行、address 約 117K 行、ages 19 行） |
| 再現性 | `SAMPLE_ACCOUNT_SEED` で RNG シード固定、`SAMPLE_ACCOUNT_NOW` で時刻固定 |
| 終了コード | 正常 0 / 例外 1 / 引数エラー 2 |

---

## 3. 非機能要件

| 項目 | 目標 |
|---|---|
| ビルド | `nix build` 一発、`nix develop` 内で `make build` |
| 単体テスト | `go test ./... -race -cover` で **80%+ カバレッジ** |
| スナップショットテスト | `SAMPLE_ACCOUNT_SEED=42 SAMPLE_ACCOUNT_NOW=1700000000 TZ=Asia/Tokyo` で固定し、Go 版基準のゴールデンファイルと diff |
| 速度 | 単スレで C++ `-O2` 比 5〜10x、並列で更に 6〜8x、合計 **30〜100x** を目指す。`count=1,000,000`・全列で 100ms 未満 |
| バイナリサイズ | < 15 MB（データ埋め込み込み） |
| CWD 依存 | なし（`go:embed` でデータ同梱） |

---

## 4. アーキテクチャ

C++ 版と同じ 3 層構造を踏襲しつつ、Go の慣用に合わせて **multi-package** へ分割する。

```
┌─────────────────────────────────────────────────┐
│ cmd/sample_account/main.go                      │ ← エントリポイント
└─────────────────────────────────────────────────┘
            │ uses
            ▼
┌─────────────────────────────────────────────────┐
│ internal/cli   parser, help, argv 順保持        │
│ internal/runner  並列ランナー                   │
└─────────────────────────────────────────────────┘
            │ uses
            ▼
┌─────────────────────────────────────────────────┐
│ internal/field   17 列の Field 実装 + Registry  │
└─────────────────────────────────────────────────┘
            │ uses
            ▼
┌─────────────────────────────────────────────────┐
│ internal/gen     PersonGen / AddressGen /       │
│                  AgeAndDateGen / Rng (PCG)      │
└─────────────────────────────────────────────────┘
            │ uses
            ▼
┌─────────────────────────────────────────────────┐
│ internal/repo    go:embed CSV → 構造体          │
│                  累積分布・オフセット事前計算   │
└─────────────────────────────────────────────────┘
```

依存は下方向のみ。テストは各層で完結する。

### 4.1 ディレクトリ構成

```
sample_account/
├── flake.nix                     # Nix 単一ソース (Go 1.26 最新, devShell, buildGoModule)
├── flake.lock
├── .envrc                        # direnv: use flake
├── go.mod                        # module sample_account
├── Makefile                      # build / test / bench / snapshot
├── README.md
├── CLAUDE.md                     # Go 版アーキテクチャ概要
├── docs/
│   ├── design.md                 # 本書
│   └── plan.md                   # 計画書 (tasks/todo.md と同期)
├── tasks/
│   └── todo.md                   # 実装タスクリスト (チェック可能)
├── data/                         # //go:embed で同梱
│   ├── address.csv
│   ├── ages.csv
│   ├── prefectures.csv
│   └── sample_account.csv
├── cmd/sample_account/main.go
├── internal/
│   ├── cli/      parser.go, help.go, *_test.go
│   ├── repo/     embed.go, person.go, prefecture.go, age.go, *_test.go
│   ├── gen/      person.go, address.go, age.go, rng.go, *_test.go
│   ├── field/    field.go, registry.go, fields.go, *_test.go
│   ├── runner/   runner.go, *_test.go
│   └── version/  version.go
└── tests/
    ├── snapshot_test.go          # ゴールデンファイル比較
    └── expected/                 # Go 版基準スナップショット
        ├── all-flags-seed-42.csv
        ├── default-seed-42.csv
        ├── ilfm-seed-42.csv
        └── long-aliases-seed-42.csv
```

### 4.2 Field 抽象化

C++ の `IField` インタフェースを Go の `Field` インタフェースに置換する：

```go
type Field interface {
    ShortFlag() byte
    LongName() string
    Description() string
    Emit(buf []byte, ctx RowContext, deps Deps) []byte
}
```

**重要**: `Emit` は `[]byte` を返すパターン（`strconv.AppendInt` と同じシグネチャ）にして、文字列アロケーションを完全に避ける。

### 4.3 RowContext と並列化戦略

C++ 版では単一の `Rng` を行毎に逐次消費していたため、行を並列化すると bytewise 再現性が壊れる。Go 版では：

```
masterSeed   = ENV["SAMPLE_ACCOUNT_SEED"] (なければ time.Now())
                       │
                       ▼
        per-row seed = splitmix64(masterSeed XOR rowIndex)
                       │
                       ▼
        rowRng = pcg.NewPCG(seed1=perRowSeed, seed2=perRowSeed * 0x9E3779B97F4A7C15)
                       │
                       ▼
        RowContext.{First, Last, Pref, Ward, City, Age} を rowRng から取得
        + 行内での追加 next() (blood, telephone, reward, random, quotient, date) も rowRng
```

これにより：
- **行は完全独立** → 並列実行しても結果不変
- **再現性は維持** （masterSeed が同じなら同じ出力）
- C++ 版とは byte 一致しない（RNG が違う）→ Go 版独自スナップショットを採用

### 4.4 並列ランナーのデータフロー

```
count, fields, masterSeed
        │
        ▼  count <= 1000 ?
        ├──[Yes]── 単一スレッド経路 (goroutine 起動コスト回避)
        │              │
        │              ▼ bufio.Writer (1MiB) → stdout
        │
        └──[No]──→ chunk 分割 (NumCPU 個)
                        │
        ┌───────────────┼───────────────┐
        ▼               ▼               ▼
    worker 0        worker 1   ...  worker N-1
    [start, end)    [start, end)
    bytes.Buffer    bytes.Buffer    bytes.Buffer
        │               │               │
        └───────────────┼───────────────┘
                        ▼  WaitGroup
        bufio.Writer に順番 (worker 0, 1, ..., N-1) で flush
                        │
                        ▼
                       stdout
```

各 worker は自分の `bytes.Buffer` だけに書き込み、完了後にメインが順番に stdout へコピー。**チャネル不要**、**ロック不要**、**マージ不要**。

### 4.5 Repo 層の最適化

| 操作 | C++ | Go |
|---|---|---|
| CSV パース | `std::getline` + `istringstream` | `bufio.Scanner` + 自作スプリッタ（Index ベース、アロケ無し） |
| `weightedPrefectureIndex` | 線形 O(P) | 累積分布配列 + `sort.Search` で O(log P) |
| `addressIndex` (offset) | 毎回 O(P) で総和 | 起動時に prefix sum を計算、O(1) ルックアップ |
| データ取得 | 実行時に `data/*.csv` を相対パスで開く | `//go:embed data/*.csv` でバイナリ同梱 |

### 4.6 出力の高速化

- `bufio.NewWriterSize(os.Stdout, 1<<20)` で 1 MiB バッファ
- `[]byte` ベースで `strconv.AppendInt`, `strconv.AppendFloat`, `append(buf, str...)` を直接使用
- `fmt.Fprintf` の format パース・リフレクションを完全回避

---

## 5. 設計判断と根拠

| 判断 | 採用 | 根拠 |
|---|---|---|
| Go バージョン | **1.26** (最新) | `math/rand/v2` PCG が安定、ツールチェーン最新 |
| RNG | `math/rand/v2` PCG (`NewPCG(seed1, seed2 uint64) *PCG`) | Context7 で API 確認済み。128bit 状態、`Uint64()` 高速、`Seed()` 再シード可 |
| データ埋め込み | `//go:embed data/*.csv` | CWD 依存解消、配布が単一バイナリで完結 |
| CLI パース | 自作スキャナ (stdlib `flag` 不可) | 出現順保持が必須、getopt クラスタ構文 (`-ilfm`) 対応必須 |
| Field ディスパッチ | interface (Field) | 17 個程度なら関数テーブル化との性能差は微小、可読性優先 |
| 並列分割 | 行レンジを NumCPU 個にチャンク化 | チャネル不要・順序自然・キャッシュ局所性良好 |
| 並列出力 | per-worker `bytes.Buffer` + 順次 flush | mutex/PQ 不要、ストリーミング不要なら最速 |
| Nix flake | `flake-utils.eachDefaultSystem` + `buildGoModule` | 標準パターン、`vendorHash = null` (依存ゼロ) |
| 環境変数 | `env = { CGO_ENABLED = "0"; }` | Nix 25.05+ の推奨スタイル、リーン静的バイナリ |

---

## 6. リスク

| ID | リスク | 影響 | 対応 |
|---|---|---|---|
| R1 | C++ と byte 一致しない | HIGH | RNG 仕様差は不可避。Go 版独自スナップショットへ切替え、README に明記 |
| R2 | 行ごとサブシードの統計的偏り | MEDIUM | splitmix64 → PCG の組合せは標準的。テストで χ² 検定の簡易版を含める |
| R3 | `address.csv` 117K 行の埋め込みでバイナリ肥大 | MEDIUM | 計測して 10 MB 超えなら gzip+embed に切替検討。当面は plain で進める |
| R4 | getopt クラスタ構文の自作再現の抜け | MEDIUM | テストで網羅: `-ilfm`, `-i -l -f -m`, `-imlf 5`, `--id --lastname` |
| R5 | `time.Local` の TZ 依存で `birthYear`/`date` が変わる | LOW | スナップショットは `TZ=Asia/Tokyo` 強制 |
| R6 | nixpkgs unstable の Go 削除（go_1_23 が EoL で削除済の前例） | LOW | flake.lock コミット、Go 最新を追従 |

---

## 7. 期待性能

C++ `-O2` 単スレッド比のおおまかな見積（理論値）：

| 要素 | 倍率 | 備考 |
|---|---|---|
| `fprintf` → `strconv.Append*` + 大バッファ | 3〜5x | format パース除去、syscall 削減 |
| `std::string` アロケ → `[]byte` 再利用 | 1.5〜2x | GC 圧減 |
| 線形→二分探索 (prefecture) | 2〜3x | 該当列使用時 |
| アドレスオフセット O(P) → O(1) | 1.5x | 同上 |
| 並列化 (8 コア想定) | 6〜8x | I/O より計算がボトルネック |

**総合: 30〜100x**（`count=1,000,000` 全列で 100ms 未満が目標）。

実測は `make bench-compare` で C++ `-O2` バイナリと wall-clock 比較し、README に掲載する。

---

## 8. テスト戦略

| 種別 | 範囲 | ツール |
|---|---|---|
| ユニット | repo, gen, field, cli, runner | `go test ./... -race -cover` |
| スナップショット | end-to-end CSV diff (4 シナリオ) | `go test -tags=snapshot ./tests/` |
| ベンチマーク | runner, gen | `go test -bench=. -benchmem` |
| Lint | 全パッケージ | `golangci-lint run` |
| Format | 全ファイル | `gofumpt -d -e .` (CI) |

カバレッジ目標 **80%+**。

---

## 9. 移行手順 (フェーズ)

1. **Phase 1**: Nix flake 構築 — `nix build` / `nix develop` 動作確認
2. **Phase 2**: data layer — embed + 4 リポジトリ + 5 ユニットテスト
3. **Phase 3**: gen layer — Person/Address/AgeAndDate/Rng + ロジック移植 + 二分探索化
4. **Phase 4**: field layer — 17 Field + Registry + alias
5. **Phase 5**: CLI parser — 自作スキャナ + ヘルプ + alias 解決
6. **Phase 6**: parallel runner — chunk 分割 + per-row 副シード + 順次 flush
7. **Phase 7**: main — エントリポイント結合
8. **Phase 8**: snapshots + bench — ゴールデンファイル + C++ 比較
9. **Phase 9**: docs — README + CLAUDE.md + ベンチ結果

詳細は `tasks/todo.md` 参照。
