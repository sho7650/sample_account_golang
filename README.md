# sample_account (Go port)

合成日本語アカウントレコードの CSV ジェネレータ。
[元の C++17 実装](../cpp/work/sample_account/) を Go へ移植し、Nix で再現可能な開発環境を提供する。

## 主な特徴

- **17 列**を任意の順序で出力（CLI フラグの出現順が列順）
- **行ごとに独立な並列生成** で `runtime.NumCPU()` まで使い切る
- **`go:embed` でデータを内蔵** — 実行時にカレントディレクトリ依存なし
- **再現可能** — `SAMPLE_ACCOUNT_SEED` / `SAMPLE_ACCOUNT_NOW` 環境変数
- **依存パッケージゼロ** — Go 標準ライブラリのみ

## ダウンロード

GitHub Releases から各 OS / アーキテクチャの実行ファイルが取得できます:

- Linux (`x86_64` / `arm64`): `sample_account_X.Y.Z_linux_amd64.tar.gz` / `sample_account_X.Y.Z_linux_arm64.tar.gz`
- macOS (Apple Silicon): `sample_account_X.Y.Z_darwin_arm64.tar.gz`
- Windows (`x86_64`): `sample_account_X.Y.Z_windows_amd64.zip`

`checksums.txt` に SHA-256 が同梱されているので、解凍前に検証可能です。

リリースは `main` への conventional commit が release-please によってまとめられ、
Release PR を merge することでタグ + バイナリが自動生成されます (詳細は
[CLAUDE.md](CLAUDE.md) のリリースフロー参照)。

## ビルド & 実行

### Nix (推奨)

```sh
nix develop                         # 開発シェル (go 1.26, gopls, gofumpt, golangci-lint, delve)
nix build                           # 静的バイナリを result/bin/ に生成
nix run                             # 直接実行
```

### Make ターゲット

```sh
make build         # go build → ./sample_account
make test          # go test ./... -race
make test-cover    # カバレッジレポート (>80%)
make snapshot      # tests/expected/ とのゴールデンファイル diff
make bench         # ベンチマーク
make bench-compare # C++ -O2 と直接比較
make lint          # golangci-lint
```

### 使用例

```sh
./sample_account --help
./sample_account                                 # COUNT=100, id 列のみ
./sample_account -ilfm 10                        # id, lastname, firstname, mail を 10 行
./sample_account --age --prefecture 5            # 年齢と都道府県を 5 行
./sample_account -ilfmpwc 100 > out.csv          # 住所込みの 100 行を CSV 保存

# 並列度を制御
./sample_account -j 1 -ilfm 1000                 # シングルスレッド
./sample_account -j 8 -ilfm 1000000              # 8 worker 並列
./sample_account --jobs=4 -ilfm 1000000          # 4 worker (--jobs= 形式)
# -j 0 は auto (NumCPU)。デフォルトと同じ。

# 再現可能な実行 — worker 数を変えても出力は同一
SAMPLE_ACCOUNT_SEED=42 SAMPLE_ACCOUNT_NOW=1700000000 TZ=Asia/Tokyo \
  ./sample_account -ilfmatpwcgbdorynq 5
```

### 並列スケーリング (count=1M, 17 列)

| `-j` | wall (s) | speedup vs `-j 1` |
|---|---|---|
| 1 | 0.39 | 1.00x |
| 2 | 0.20 | 1.95x |
| 4 | 0.11 | 3.55x |
| 8 | 0.07 | 5.57x |
| 16 | 0.06 | 6.50x |

per-row sub-RNG (`splitmix64(masterSeed XOR row)`) で行を独立化しているので
`-j` を変えても出力は **byte-identical**。

### 大規模生成のメモリ・スループット (count=1B / 13 列)

| 指標 | 値 |
|---|---|
| wall time | 253s |
| peak RSS | 2.7 GB |
| user CPU | 254s (≈ 1 core 相当) |

10 億行 (約 134 GB の出力) を OOM なく生成できる。CPU 利用率が 1 core 相当に
なるのは stdout (`> /dev/null` 含む) への write syscall が ~500 MB/s で律速して
いるため。実ファイルへの書き出し (バッファキャッシュ経由) なら更に短縮される。

メモリ上限の設計: `subChunkRows × outboxDepth` 個のバッファをパイプラインに
持つだけで、行数 N に依存しない `O(workers × subChunkCap)` (デフォルト 1.25 GiB
ceiling)。10B でも 100B でも RSS は同等。

## 列リファレンス

| 短形 | 長形 | 内容 |
|---|---|---|
| `-i` | `--id` | 1-based 行番号 |
| `-l` | `--lastname` | 苗字 (kanji,kana — 2 列出力) |
| `-f` | `--firstname` | 名前 (kanji,kana — 2 列出力) |
| `-m` | `--mail` | メールアドレス (`first_last@example.com`) |
| `-t` | `--telephone` | 電話番号 (`090-XXXX-XXXX`) |
| `-p` | `--prefecture` | 都道府県名 (人口加重) |
| `-w` | `--ward` | 市区町村 |
| `-c` | `--city` | 町字 |
| `-g` | `--gender` | 性別 (男 / 女) |
| `-b` | `--blood` | ABO 血液型 |
| `-a` | `--age` | 年齢 (人口加重) |
| `-o` | `--agegroup` | 年代 (10 年単位) |
| `-y` | `--birthyear` | 出生年 |
| `-r` | `--reward` | 年収風の数値 |
| `-d` | `--date` | ランダム日付 (`YYYY/M/D`) |
| `-n` | `--random` | ±10,000,000 の整数 |
| `-q` | `--quotient` | 0.00〜0.99 の小数 |

`--telehpne` は `--telephone` のレガシー別名 (旧ツールチェーン互換のため保持)。

## 性能

C++ `-O2` バイナリと同条件で同 CSV を生成した実測値 (Apple M4 Max, 16 cores):

| count | C++ -O2 | Go | speedup |
|---|---|---|---|
| 100 | 0.040s | 0.030s | 1.31x |
| 1,000 | 0.056s | 0.052s | 1.08x |
| 10,000 | 0.068s | 0.044s | 1.53x |
| 100,000 | 0.244s | 0.053s | 4.58x |
| **1,000,000** | **1.900s** | **0.087s** | **21.9x** |

小さい `count` ではプロセス起動コストが支配的なので差が小さくなる一方、
大量生成では並列化が効いて **22 倍** まで開く。生成だけのカーネル時間で
比較すると C++ 1.9s に対し Go 42 ms (約 45x) まで詰まる。

## C++ 版との差異

| 項目 | C++ | Go |
|---|---|---|
| 出力 | バイト一致 | **異なる** (RNG が違う) |
| RNG | C `rand()` (LCG) | `math/rand/v2` PCG |
| データ | `data/*.csv` を相対パスで読む | `go:embed` で内蔵 (CWD 不要) |
| 並列化 | なし | NumCPU 分のチャンク並列 |
| 文字列 | `std::string` ヒープ | `[]byte` 直書き (`strconv.Append*`) |

C++ の `tests/expected/` とは互換ではない。Go 版は `tests/expected/` を独自に
生成・コミットしている。

## アーキテクチャ概要

```
cmd/sample_account/main.go    エントリポイント
↓
internal/cli                  argv パース (出現順保持・getopt 互換)
internal/runner               チャンク並列ランナー (per-row sub-RNG)
↓
internal/field                17 Field + Registry + alias
↓
internal/gen                  PersonGen / AddressGen / AgeGen / Rng (PCG)
↓
internal/repo                 //go:embed CSV → 構造体 (累積分布事前計算)
```

詳細設計は [docs/design.md](docs/design.md) 参照。

## 再現性

```sh
# シードと "現在時刻" を固定すれば常に同じ出力
SAMPLE_ACCOUNT_SEED=42 SAMPLE_ACCOUNT_NOW=1700000000 TZ=Asia/Tokyo \
  ./sample_account -ilfmatpwcgbdorynq 5
```

並列化しても per-row sub-RNG (`splitmix64(masterSeed XOR row)`) で行を独立
させているので、worker 数や CPU 数が変わっても出力は不変。

## 開発

```sh
nix develop                    # devShell に入る
go test ./... -race            # 全テスト + race detector
go test -tags=snapshot ./tests # スナップショットテスト
golangci-lint run              # lint
gofumpt -w .                   # 整形
```

カバレッジ: 85.4% (2026-05-08 時点)。

### Conventional Commits

リリースは `main` への [Conventional Commits](https://www.conventionalcommits.org/) を
release-please が解析して自動化します。コミットメッセージ規約:

| 接頭辞 | 意味 | バージョン影響 |
|---|---|---|
| `feat:` | 新機能 | minor bump (`0.5.0 → 0.6.0`) |
| `fix:` | バグ修正 | patch bump (`0.5.0 → 0.5.1`) |
| `feat!:` / footer に `BREAKING CHANGE:` | 互換性破壊 | major bump (`0.5.0 → 1.0.0`) |
| `perf:` / `refactor:` | 性能・整理 | バンプなし、CHANGELOG に記載 |
| `docs:` / `test:` / `chore:` / `ci:` / `build:` | 周辺作業 | バンプなし、CHANGELOG 非表示 |

PR 作成時に上記 prefix を必ず使用してください。

## ライセンス

MIT — 詳細は [LICENSE](LICENSE) を参照。
