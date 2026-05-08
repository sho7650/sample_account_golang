# 計画書: sample_account Go 移植

設計詳細は [docs/design.md](../docs/design.md) を参照。

## 確認済み事項

| 項目 | 決定 |
|---|---|
| Module path | `sample_account` (ローカル module、後で `github.com/<owner>/sample_account_golang` に書き換え予定) |
| データ戦略 | `//go:embed data/*.csv` でバイナリ同梱 |
| RNG | `math/rand/v2` の PCG (`NewPCG`) |
| C++ との出力一致 | byte 一致は諦める。Go 独自スナップショットを採用 |
| Lint | `golangci-lint` を flake に含める |
| Go バージョン | **1.26** (nixpkgs 最新) |
| Nixpkgs | `nixos-unstable` (flake.lock で固定) |

---

## 実装タスク

### Phase 1: Nix flake で Go ツールチェーン構築

- [x] `flake.nix` 作成 (go_1_26, gopls, gofumpt, golangci-lint, delve, gnumake)
- [x] `.envrc` (`use flake`)
- [x] `.gitignore`
- [x] `go.mod` (module sample_account, go 1.26)
- [x] `nix flake check --no-build` で構文 OK 確認
- [ ] `nix develop -c go version` で 1.26 起動確認
- [ ] `nix build` で空ビルドが通るか確認 (Phase 7 完了後)

### Phase 2: データ層 (repo パッケージ + go:embed)

- [ ] `data/*.csv` を C++ 版からコピー
- [ ] `internal/repo/embed.go`: `//go:embed data/*` で `embed.FS`
- [ ] `internal/repo/person.go`: `PersonRecord`, `LoadPersons(fs)` (8 列)
- [ ] `internal/repo/prefecture.go`: `PrefectureRecord`, `AddressRecord`, prefix sum で zip オフセット O(1) 化、累積 population で重み付き探索用配列
- [ ] `internal/repo/age.go`: `AgeBucket`, `parseDigits` で桁区切り処理、`start` 累積
- [ ] `internal/repo/repo_test.go`: C++ test_repos.cpp 5 件移植
  - `person_repo_loads_records`
  - `prefecture_repo_loads_47_prefectures`
  - `prefecture_repo_assigns_zips_to_each_prefecture`
  - `age_repo_strips_thousand_separators`
  - `person_repo_throws_on_missing_file` → embed 版では別の負例 (空ファイル等) に置換

### Phase 3: 生成層 (gen パッケージ)

- [ ] `internal/gen/rng.go`: `Rng` 型 (PCG ラップ), `NewMasterRng(seed uint64)`, `NewRowRng(masterSeed, row uint64)`, `Next() uint32`, `RollDate() (year, month, day int)`
- [ ] `currentTime()`: `SAMPLE_ACCOUNT_NOW` 読み取り、なければ `time.Now().Unix()`
- [ ] `internal/gen/person.go`: `lastName`, `firstName`, `mailAddress`, `gender`, `bloodType`
- [ ] `internal/gen/address.go`: `weightedPrefectureIndex` (`sort.Search`), `prefectureName`, `ward`, `city`, `addressIndex` (precomputed offset)
- [ ] `internal/gen/age.go`: `age`, `ageGroup`, `birthYear`, `reward`
- [ ] `internal/gen/gen_test.go`: 各メソッドのプロパティテスト + 境界条件

### Phase 4: フィールド層 (field パッケージ, 17 列)

- [ ] `internal/field/field.go`: `Field` interface, `RowContext`, `Deps`
- [ ] `internal/field/registry.go`: `Registry`, `Add`, `FindShort`, `FindLong`, `All`, `ShortOptString`
- [ ] `internal/field/fields.go`: 17 実装
  - id, lastname, firstname, mail, telephone
  - prefecture, ward, city
  - gender, blood
  - age, agegroup, birthyear, reward
  - date, random, quotient
- [ ] alias テーブル: `--telehpne` → `t`
- [ ] `internal/field/field_test.go`: 各列の `Emit` 出力検証

### Phase 5: CLI パーサ (cli パッケージ)

- [ ] `internal/cli/parser.go`: argv スキャナ (`-h`/`--help`, `-x`, `-xyz` クラスタ, `--name`, COUNT)
- [ ] デフォルト無選択時 `-i` のみ
- [ ] `internal/cli/help.go`: `printHelp(w io.Writer, prog string, reg *Registry)`
- [ ] `internal/cli/parser_test.go`:
  - `-ilfm 5` クラスタ
  - `-i -l -f -m` 個別
  - `-imlf 5` 順序保持
  - `--id --lastname --firstname --mail`
  - `--telehpne` alias
  - 不明フラグ
  - COUNT 既定 100
  - フラグ無し → id のみ

### Phase 6: 並列ランナー (runner パッケージ)

- [ ] `internal/runner/runner.go`:
  - `Run(w io.Writer, count int, fields []Field, deps Deps, masterSeed uint64) error`
  - count <= 1000 時の単一スレッド経路
  - chunk 分割 (`runtime.NumCPU()` 個)
  - per-worker `bytes.Buffer` (容量見積で初期確保)
  - per-row sub-seed: `splitmix64(masterSeed ^ row)`
  - `WaitGroup` 後に順次 flush
  - `bufio.NewWriterSize(w, 1<<20)`
- [ ] `internal/runner/runner_test.go`:
  - 単一スレッド経路と並列経路で出力一致
  - 異なる masterSeed で出力が変わる
  - count=0 で空出力

### Phase 7: エントリポイント (cmd/sample_account/main.go)

- [ ] registry 構築
- [ ] CLI パース → ヘルプ/エラー処理
- [ ] repo 構築 (embed)
- [ ] runner 実行
- [ ] 終了コード 0/1/2

### Phase 8: スナップショット & ベンチマーク

- [ ] `tests/snapshot_test.go`: 4 シナリオ
  - `-ilfmatpwcgbdorynq 5` (all flags)
  - `-ilfm 5` (id+name+mail)
  - `3` (default = id only)
  - `--telephone --agegroup --birthyear 4`
- [ ] `tests/expected/*.csv` 生成 (`SAMPLE_ACCOUNT_SEED=42 SAMPLE_ACCOUNT_NOW=1700000000 TZ=Asia/Tokyo`)
- [ ] `internal/runner/bench_test.go`: count=100, 10K, 1M
- [ ] `Makefile`: `build`, `test`, `snapshot`, `bench`, `bench-compare`, `lint`
- [ ] `bench-compare`: C++ `-O2` バイナリと wall-clock 比較スクリプト

### Phase 9: ドキュメント

- [ ] `README.md`:
  - 概要 / ビルド (Nix) / 実行例
  - C++ 版との差異 (RNG 違い、byte 一致せず、並列出力)
  - ベンチ結果 (C++ 比 30〜100x の実測)
- [ ] `CLAUDE.md` (Go 版):
  - アーキテクチャ概要 (3 層 → multi-package)
  - 列追加手順 (Field 実装 1 個 + Registry 登録 1 行)
  - テストフック (`SAMPLE_ACCOUNT_SEED` / `SAMPLE_ACCOUNT_NOW` / `TZ`)
  - Nix devShell 使い方

---

## レビューセクション (実装完了後に記入)

実装完了 (2026-05-08)。

### 実測ベンチ (Apple M4 Max, 16 cores, 全 17 列)

| count | C++ -O2 | Go | speedup |
|---|---|---|---|
| 100 | 0.040s | 0.030s | 1.31x |
| 1,000 | 0.056s | 0.052s | 1.08x |
| 10,000 | 0.068s | 0.044s | 1.53x |
| 100,000 | 0.244s | 0.053s | 4.58x |
| **1,000,000** | **1.900s** | **0.087s** | **21.94x** |

純生成カーネル (起動コスト除く) では `count=1M` で **45x** (1.9s → 42ms)。
小さい count ではプロセス起動コストが支配的になり差が小さくなる。

### カバレッジ実測値

| パッケージ | カバレッジ |
|---|---|
| `internal/cli` | 90.6% |
| `internal/field` | 99.2% |
| `internal/gen` | 86.0% |
| `internal/repo` | 86.1% |
| `internal/runner` | 92.9% |
| **全体** | **85.4%** (目標 80% 超達成) |

### バイナリサイズ

7.4 MB (`-ldflags="-s -w"` 込み、データ 5 MB を内蔵)。

### 想定外 / 設計変更した点

1. **`os.Getenv` の per-row コストが想定外に高かった**
   - 当初実装で `RollDate()` 内で `os.Getenv("SAMPLE_ACCOUNT_NOW")` を毎行呼んでいた
   - プロファイルで sync/atomic に 49% を奪われていた (Go の `os.Getenv` は内部で atomic int32 読み込み)
   - `Rng.nowUnix`, `AgeGen.nowYear` でコンストラクタ時にキャッシュ → **6x → 22x の大幅改善**
2. **PersonGen のメモリ最適化**
   - `LastName(n)` / `FirstName(n)` は当初 `r.LastKanji + "," + r.LastKana` で行毎に文字列を生成
   - 起動時に pre-join した文字列を slice に保持する形に変更 → アロケーションが行毎に消える
3. **CSV パーサの最終形**
   - `encoding/csv` ではなく自作スプリッタ (`bufio.Scanner` + `strings.IndexByte`) を採用
   - 引用符なし固定列の単純フォーマットなので自作のほうが高速
4. **Go 1.23 → 1.26 への切り替え**
   - 当初 Go 1.23 を計画していたが nixpkgs 最新で `go_1_23` が EoL 削除されていた
   - 最新 `go_1_26` (1.26.2) に切替。`math/rand/v2` の API は変更なし

### 残課題 / 改善余地

- 純カーネル時間 42 ms に対してプロセス起動 +I/O で +45 ms かかっている
  - go:embed パース処理が一部 (117K 行の address.csv) を起動時に走る
  - 必要なら parquet 等の binary 形式に切り替えると更に縮む
- `internal/runner/runner.go` の `runParallel` カバレッジが 90.9% — エラーパスの一部が未カバー

---

## 追加機能 (2026-05-08): CI / Release 自動化

### Phase 1: release-please

- [x] `.github/release-please-config.json` (release-type: go, bootstrap-sha=9d8f54b, extra-files で version.go 自動更新)
- [x] `.github/.release-please-manifest.json` (`. : "0.5.0"`)
- [x] `internal/version/version.go` に `// x-release-please-version` マーカー追加

### Phase 2: CI ワークフロー

- [x] `.github/workflows/ci.yml` (ubuntu-latest 単一)
- [x] `actions/setup-go@v5` with `go-version-file: go.mod` (キャッシュ込み)
- [x] go vet + golangci-lint v6 + go test -race -cover + snapshot test (TZ=Asia/Tokyo)
- [x] coverage.out を artifact 化

### Phase 3: GoReleaser

- [x] `.goreleaser.yaml` v2 形式
- [x] ターゲット 4 種: linux/amd64, linux/arm64, darwin/arm64, windows/amd64 (`ignore` で他を除外)
- [x] tar.gz (linux/macOS) + zip (windows) + checksums.txt
- [x] release-please の changelog を上書きしないよう `mode: keep-existing` + `changelog.disable: true`
- [x] `.github/workflows/release-please.yml` の 2nd job で goreleaser を実行 (PAT 不要のため tag 連鎖制限を回避)

### Phase 4: ドキュメント

- [x] README.md にバイナリ DL セクション + Conventional Commits 規約
- [x] CLAUDE.md にリリースフロー手順 + repo settings 前提条件
- [x] .gitignore に `/.goreleaser.cache/` 追加

### ローカル検証実績

```
$ goreleaser check         → 1 configuration file(s) validated
$ goreleaser release --snapshot --clean --skip=publish → 4 ターゲットすべてビルド成功
  - sample_account_*_linux_amd64.tar.gz   (ELF 64-bit, x86-64, stripped)
  - sample_account_*_linux_arm64.tar.gz   (ELF 64-bit, ARM aarch64, stripped)
  - sample_account_*_darwin_arm64.tar.gz  (Mach-O 64-bit arm64) ← 起動 + 出力一致確認済
  - sample_account_*_windows_amd64.zip    (PE32+ x86-64)
  - checksums.txt
```
