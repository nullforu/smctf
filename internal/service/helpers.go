package service

func trimTo(v string, max int) string {
	if len(v) <= max {
		return v
	}
	return v[:max]
}
