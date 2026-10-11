package app

import "context"

// Cached aggregates are immutable snapshots and are invalidated after a successful commit.
func (s *Store) Facets(service string) (map[string][]string, error) {
	return s.FacetsContext(context.Background(), service)
}
func (s *Store) FacetsContext(ctx context.Context, service string) (map[string][]string, error) {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if result, ok := s.facetCache[service]; ok {
		return cloneFacets(result), nil
	}
	result, err := s.facets(ctx, service)
	if err != nil {
		return nil, err
	}
	s.facetCache[service] = result
	return cloneFacets(result), nil
}
func cloneFacets(in map[string][]string) map[string][]string {
	out := map[string][]string{}
	for key, values := range in {
		out[key] = append([]string{}, values...)
	}
	return out
}
func (s *Store) Statistics() (Statistics, error) { return s.StatisticsContext(context.Background()) }
func (s *Store) StatisticsContext(ctx context.Context) (Statistics, error) {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	if err := ctx.Err(); err != nil {
		return Statistics{}, err
	}
	if s.statisticsCache != nil {
		return cloneStatistics(*s.statisticsCache), nil
	}
	result, err := s.statistics(ctx)
	if err != nil {
		return result, err
	}
	s.statisticsCache = &result
	return cloneStatistics(result), nil
}
func cloneStatistics(in Statistics) Statistics {
	out := in
	out.LanguageTotals = map[string]int{}
	for key, value := range in.LanguageTotals {
		out.LanguageTotals[key] = value
	}
	out.Emby = map[string]int{}
	for key, value := range in.Emby {
		out.Emby[key] = value
	}
	out.Distributions = map[string]map[string][]StatisticBucket{}
	for service, traits := range in.Distributions {
		out.Distributions[service] = map[string][]StatisticBucket{}
		for key, buckets := range traits {
			out.Distributions[service][key] = append([]StatisticBucket{}, buckets...)
		}
	}
	return out
}

// Custom tags do not change Dashboard's media/audio aggregates.
func (s *Store) invalidateCustomFacets() {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	s.facetCache = map[string]map[string][]string{}
}
