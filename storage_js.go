//go:build js

package main

import (
	"errors"
	"syscall/js"
)

// storage_js.go はブラウザ (WASM) 向けの保存先実装。
// 中断セーブ・設定をブラウザの localStorage へ読み書きするため、
// OS のファイルシステムが無い WASM でも中断セーブが機能する。

// errStorageNotExist は該当キーが localStorage に無いことを表す
var errStorageNotExist = errors.New("storage: 保存データが存在しません")

// errStorageUnavailable は localStorage 自体が使えないことを表す
var errStorageUnavailable = errors.New("storage: localStorage が利用できません")

// localStorage はブラウザの window.localStorage を返す。
// プライベートモードや未対応環境では undefined になり得る。
func localStorage() js.Value {
	defer func() { _ = recover() }()
	return js.Global().Get("localStorage")
}

// storageRead は localStorage から指定キーの文字列を読み込む
func storageRead(name string) (data []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			data, err = nil, errStorageUnavailable
		}
	}()
	ls := localStorage()
	if !ls.Truthy() {
		return nil, errStorageUnavailable
	}
	v := ls.Call("getItem", name)
	// 未設定のキーは null（Truthy でない）が返る
	if !v.Truthy() {
		return nil, errStorageNotExist
	}
	return []byte(v.String()), nil
}

// storageWrite は localStorage へ指定キーで文字列を書き込む。
// 容量超過などで setItem が例外を投げる場合はエラーとして返す。
func storageWrite(name string, data []byte) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errStorageUnavailable
		}
	}()
	ls := localStorage()
	if !ls.Truthy() {
		return errStorageUnavailable
	}
	ls.Call("setItem", name, string(data))
	return nil
}

// storageRemove は localStorage から指定キーを削除する
func storageRemove(name string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errStorageUnavailable
		}
	}()
	ls := localStorage()
	if !ls.Truthy() {
		return nil
	}
	ls.Call("removeItem", name)
	return nil
}

// storageIsNotExist は「保存データが存在しない」エラーかどうかを返す
func storageIsNotExist(err error) bool {
	return errors.Is(err, errStorageNotExist)
}
