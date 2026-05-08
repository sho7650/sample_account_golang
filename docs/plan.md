# 計画書 (公開版)

実装フェーズの全体像。チェック付きの最新ステータスは [tasks/todo.md](../tasks/todo.md) を参照。

## フェーズ一覧

| Phase | 内容 | 想定時間 |
|---|---|---|
| 1 | Nix flake で Go ツールチェーン構築 | 0.5h |
| 2 | データ層 (repo パッケージ + go:embed) | 1h |
| 3 | 生成層 (gen パッケージ) | 1h |
| 4 | フィールド層 (field パッケージ, 17 列) | 1.5h |
| 5 | CLI パーサ (cli パッケージ) | 1h |
| 6 | 並列ランナー (runner パッケージ) | 1h |
| 7 | エントリポイント (cmd/sample_account/main.go) | 0.25h |
| 8 | スナップショット & ベンチマーク | 1.5h |
| 9 | ドキュメント (README, CLAUDE.md) | 0.5h |
| **計** | | **~8.25h** |

## 速度向上戦略 (3 軸)

1. **アルゴリズム最適化**: 線形探索 → 二分探索 (prefecture)、O(P) → O(1) (address offset)
2. **I/O 最適化**: `bufio.Writer` 1 MiB + `strconv.Append*` で `[]byte` 直書き
3. **並列化**: 行レンジを `NumCPU()` 個にチャンク分割、per-worker `bytes.Buffer`、順次 flush

## 確定済み技術選択

- Go: 1.26 (nixpkgs `go_1_26`, 1.26.2)
- RNG: `math/rand/v2` PCG (`NewPCG(seed1, seed2 uint64) *PCG`)
- 並列出力: 各 worker が `bytes.Buffer` を埋め、メインが順番に `bufio.Writer` へコピー
- データ: `//go:embed data/*.csv` でバイナリ同梱
- CLI: 自作 argv スキャナ (出現順保持・getopt クラスタ対応)
- Nix: `flake-utils.eachDefaultSystem` + `buildGoModule` (vendorHash = null)

## 期待性能

C++ `-O2` 単スレ比 **30〜100x**。`count=1,000,000` 全列出力で 100ms 未満を目標。
実測は Phase 8 で確定し、README に記載する。

## 非機能目標

- カバレッジ 80%+
- バイナリサイズ < 15 MB
- CWD 依存ゼロ (どこからでも実行可)
- TZ 依存は `Asia/Tokyo` をスナップショット用に強制
