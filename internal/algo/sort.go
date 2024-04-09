package algo

import "sort"

// Sort a map by its values in descending order
func SortMapByValue[T comparable](m map[T]float64) map[T]float64 {
	type kv struct {
		Key   T
		Value float64
	}
	var sorted []kv
	for k, v := range m {
		sorted = append(sorted, kv{k, v})
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Value > sorted[j].Value
	})
	sortedMap := make(map[T]float64)
	for _, kv := range sorted {
		sortedMap[kv.Key] = kv.Value
	}
	return sortedMap
}
