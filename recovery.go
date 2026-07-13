package main

// recoveredValue returns a recovered stat without allowing it to exceed its maximum.
func recoveredValue(current, maximum, amount int, fullRecovery bool) int {
	if fullRecovery || current+amount > maximum {
		return maximum
	}
	return current + amount
}
