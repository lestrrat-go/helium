package heliumcmd

func mergeExitCode(current, next int) int {
	if next > current {
		return next
	}
	return current
}
