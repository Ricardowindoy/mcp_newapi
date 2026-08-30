package logs

import "sort"

func sortByQuota(ms []ModelUsage) {
	sort.Slice(ms, func(i, j int) bool { return ms[i].Quota > ms[j].Quota })
}
