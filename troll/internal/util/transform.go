package util

func TransSex(sex string) int {
	switch sex {
	case "女":
		return 0
	case "男":
		return 1
	case "保密":
		return 2
	default:
		return -1
	}
}
