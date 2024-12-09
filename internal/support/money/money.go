package money

func IntToFloat(sum int64) float64 {
	return float64(sum) / 100
}

func FloatToInt(sum float64) int64 {
	return int64(sum * 100)
}
