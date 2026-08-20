//go:build !js

package main

import "os"

// storage.go はセーブ・設定データの保存先を抽象化する（ネイティブ版）。
// ネイティブ環境ではカレントディレクトリのファイルへ読み書きする。
// WASM 版は storage_js.go がブラウザの localStorage を使う。

// storageRead は指定キーのデータを読み込む
func storageRead(name string) ([]byte, error) {
	return os.ReadFile(name)
}

// storageWrite は指定キーへデータを書き込む
func storageWrite(name string, data []byte) error {
	return os.WriteFile(name, data, 0644)
}

// storageRemove は指定キーのデータを削除する
func storageRemove(name string) error {
	return os.Remove(name)
}

// storageIsNotExist は「保存先が存在しない」ことを表すエラーかどうかを返す
func storageIsNotExist(err error) bool {
	return os.IsNotExist(err)
}
