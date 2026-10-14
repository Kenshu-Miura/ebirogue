package main

// gnawResult はかつおぶしを1回かじった結果。強化値-3以下なら食べきる。
func gnawResult(satiety, maxSatiety, sharpness int) (newSatiety, newSharpness int, consumed bool) {
	return recoveredValue(satiety, maxSatiety, 15, false), sharpness - 1, sharpness <= -3
}
