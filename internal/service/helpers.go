package service

func trimTo(value string, max int) string {
	if len(value) <= max {
		return value
	}
	return value[:max]
}
