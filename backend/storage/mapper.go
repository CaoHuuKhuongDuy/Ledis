package storage

func setDataToStrings(data *map[string]bool) []string {
	results := make([]string, 0, len(*data))
	for key := range *data {
		results = append(results, key)
	}
	return results
}
