package api

import (
	"fmt"
	"log"
	"net/url"
	"strconv"
	"strings"
)

// SearchKeys are filterable attributes
const (
	SearchKeySources          = "sources"
	SearchKeyASNS             = "asns"
	SearchKeyCommunities      = "communities"
	SearchKeyExtCommunities   = "ext_communities"
	SearchKeyLargeCommunities = "large_communities"
	SearchKeyAddrFamily       = "addr_family"
)

// Filterable objects provide methods for matching
// by ID, ASN, Community, etc...
type Filterable interface {
	MatchSourceID(sourceID string) bool
	MatchASN(asn int) bool
	MatchCommunity(community Community) bool
	MatchExtCommunity(community ExtCommunity) bool
	MatchLargeCommunity(community Community) bool
	MatchAddrFamily(family uint8) bool
}

// FilterValue can be anything
type FilterValue interface{}

// SearchFilter is a key value pair with
// an indicator how many results the predicate
// does cover.
type SearchFilter struct {
	Cardinality int         `json:"cardinality"`
	Name        string      `json:"name"`
	Value       FilterValue `json:"value"`
}

// A SearchFilterCmpFunc can be implemented for various
// types, e.g. for integer matching or string matchin.
type SearchFilterCmpFunc func(a FilterValue, b FilterValue) bool

// Compare integers
func searchFilterCmpInt(a FilterValue, b FilterValue) bool {
	return a.(int) == b.(int)
}

// Compare strings
func searchFilterCmpString(a FilterValue, b FilterValue) bool {
	var (
		valA string
		valB string
	)
	_, ptrA := a.(*string)
	_, ptrB := b.(*string)

	// Compare pointers, this is ok because we can assume
	// using pool values for both.
	if ptrA && ptrB {
		return a == b
	}

	// Otherwise fall back to string compare
	if ptrA {
		valA = *a.(*string)
	} else {
		valA = a.(string)
	}
	if ptrB {
		valB = *b.(*string)
	} else {
		valB = b.(string)
	}

	return valA == valB
}

// Compare communities
func searchFilterCmpCommunity(a FilterValue, b FilterValue) bool {
	ca := a.(Community)
	cb := b.(Community)

	if len(ca) != len(cb) {
		return false
	}

	// Compare components
	for i := range ca {
		if ca[i] != cb[i] {
			return false
		}
	}
	return true
}

// Compare extended communities
func searchFilterCmpExtCommunity(a FilterValue, b FilterValue) bool {
	ca := a.(ExtCommunity)
	cb := b.(ExtCommunity)

	if len(ca) != len(cb) || len(ca) != 3 || len(cb) != 3 {
		return false
	}

	return ca[0] == cb[0] && ca[1] == cb[1] && ca[2] == cb[2]
}

// Equal checks the equality of two filters
// by applying the appropriate compare function
// to the serach filter value.
func (f *SearchFilter) Equal(other *SearchFilter) bool {
	var cmp SearchFilterCmpFunc
	switch other.Value.(type) {
	case Community:
		cmp = searchFilterCmpCommunity
	case ExtCommunity:
		cmp = searchFilterCmpExtCommunity
	case int:
		cmp = searchFilterCmpInt
	case string:
		cmp = searchFilterCmpString
	case *string:
		cmp = searchFilterCmpString
	}

	if cmp == nil {
		log.Println("Unknown search filter value type")
		return false
	}

	return cmp(f.Value, other.Value)
}

// SearchFilterGroup contains filtergroups and
// an index.
type SearchFilterGroup struct {
	Key string `json:"key"`

	Filters    []*SearchFilter `json:"filters"`
	filtersIdx map[string]int
}

// FindFilter tries to lookup a filter in
// a search filter group.
func (g *SearchFilterGroup) FindFilter(filter *SearchFilter) *SearchFilter {
	for _, f := range g.Filters {
		if f.Equal(filter) {
			return f
		}
	}
	return nil
}

// Contains checks if a filter is present in a a group
func (g *SearchFilterGroup) Contains(filter *SearchFilter) bool {
	return g.FindFilter(filter) != nil
}

// filterValueAsString gets the string representation
// from a filter value
func filterValueAsString(value interface{}) string {
	switch v := value.(type) {
	case int:
		return strconv.Itoa(v)
	case uint8:
		return strconv.Itoa(int(v))
	case *string:
		return *v
	case string:
		return v
	case Community:
		return v.String()
	case ExtCommunity:
		return v.String()
	}
	panic("unexpected filter value: " + fmt.Sprintf("%v", value))
}

// GetFilterByValue retrieves a filter by matching
// a string representation of it's filter value.
func (g *SearchFilterGroup) GetFilterByValue(value interface{}) *SearchFilter {
	ref := filterValueAsString(value)
	idx, ok := g.filtersIdx[ref]
	if !ok {
		return nil // We don't have this particular filter
	}
	return g.Filters[idx]
}

// AddFilter adds a filter to a group
func (g *SearchFilterGroup) AddFilter(filter *SearchFilter) {
	// Check if a filter with this value is present, if not:
	// append and update index; otherwise incrementc cardinality
	if presentFilter := g.GetFilterByValue(filter.Value); presentFilter != nil {
		presentFilter.Cardinality++
		return
	}

	// Insert filter and update index
	idx := len(g.Filters)
	filter.Cardinality = 1
	g.Filters = append(g.Filters, filter)
	ref := filterValueAsString(filter.Value)
	g.filtersIdx[ref] = idx
}

// AddFilters adds a list of filters to a group.
func (g *SearchFilterGroup) AddFilters(filters []*SearchFilter) {
	for _, filter := range filters {
		g.AddFilter(filter)
	}
}

// Rebuild the filter index
func (g *SearchFilterGroup) rebuildIndex() {
	idx := make(map[string]int)
	for i, filter := range g.Filters {
		ref := filterValueAsString(filter.Value)
		idx[ref] = i
	}
	g.filtersIdx = idx // replace index
}

// A SearchFilterComparator compares route with a filter
type SearchFilterComparator func(route Filterable, value interface{}) bool

func searchFilterMatchSource(route Filterable, value interface{}) bool {
	sourceID, ok := value.(string)
	if !ok {
		return false
	}
	return route.MatchSourceID(sourceID)
}

func searchFilterMatchASN(route Filterable, value interface{}) bool {
	asn, ok := value.(int)
	if !ok {
		return false
	}

	return route.MatchASN(asn)
}

func searchFilterMatchCommunity(route Filterable, value interface{}) bool {
	community, ok := value.(Community)
	if !ok {
		return false
	}
	return route.MatchCommunity(community)
}

func searchFilterMatchExtCommunity(route Filterable, value interface{}) bool {
	community, ok := value.(ExtCommunity)
	if !ok {
		return false
	}
	return route.MatchExtCommunity(community)
}

func searchFilterMatchLargeCommunity(route Filterable, value interface{}) bool {
	community, ok := value.(Community)
	if !ok {
		return false
	}
	return route.MatchLargeCommunity(community)
}

func searchFilterMatchAddrFamily(route Filterable, value interface{}) bool {
	family, ok := value.(int)
	if !ok {
		return false
	}
	return route.MatchAddrFamily(uint8(family))
}

func selectCmpFuncByKey(key string) SearchFilterComparator {
	var cmp SearchFilterComparator
	switch key {
	case SearchKeySources:
		cmp = searchFilterMatchSource
	case SearchKeyASNS:
		cmp = searchFilterMatchASN
	case SearchKeyCommunities:
		cmp = searchFilterMatchCommunity
	case SearchKeyExtCommunities:
		cmp = searchFilterMatchExtCommunity
	case SearchKeyLargeCommunities:
		cmp = searchFilterMatchLargeCommunity
	case SearchKeyAddrFamily:
		cmp = searchFilterMatchAddrFamily
	default:
		cmp = nil
	}

	return cmp
}

// MatchAny checks if a route matches any filter
// in a filter group.
func (g *SearchFilterGroup) MatchAny(route Filterable) bool {
	// Check if we have any filter to match
	if len(g.Filters) == 0 {
		return true // no filter, everything matches
	}

	// Get comparator
	cmp := selectCmpFuncByKey(g.Key)
	if cmp == nil {
		return false // This should not have happened!
	}

	// Check if any of the given filters matches
	for _, filter := range g.Filters {
		if cmp(route, filter.Value) {
			return true
		}
	}
	return false
}

// MatchAll checks if a route matches all predicates
// in the filter group.
func (g *SearchFilterGroup) MatchAll(route Filterable) bool {
	// Check if we have any filter to match
	if len(g.Filters) == 0 {
		return true // no filter, everything matches. Like above.
	}

	// Get comparator
	cmp := selectCmpFuncByKey(g.Key)
	if cmp == nil {
		return false // This again should not have happened!
	}

	// Assert that all filters match.
	for _, filter := range g.Filters {
		if !cmp(route, filter.Value) {
			return false
		}
	}

	// Everythings fine.
	return true
}

// SearchFilters is a collection of filter groups
type SearchFilters []*SearchFilterGroup

// NewSearchFilters creates a new collection
// of search filter groups.
func NewSearchFilters() *SearchFilters {
	// Define groups: CAVEAT! the order is relevant
	groups := &SearchFilters{
		&SearchFilterGroup{
			Key:        SearchKeySources,
			Filters:    []*SearchFilter{},
			filtersIdx: make(map[string]int),
		},
		&SearchFilterGroup{
			Key:        SearchKeyASNS,
			Filters:    []*SearchFilter{},
			filtersIdx: make(map[string]int),
		},
		&SearchFilterGroup{
			Key:        SearchKeyCommunities,
			Filters:    []*SearchFilter{},
			filtersIdx: make(map[string]int),
		},
		&SearchFilterGroup{
			Key:        SearchKeyExtCommunities,
			Filters:    []*SearchFilter{},
			filtersIdx: make(map[string]int),
		},
		&SearchFilterGroup{
			Key:        SearchKeyLargeCommunities,
			Filters:    []*SearchFilter{},
			filtersIdx: make(map[string]int),
		},
		&SearchFilterGroup{
			Key:        SearchKeyAddrFamily,
			Filters:    []*SearchFilter{},
			filtersIdx: make(map[string]int),
		},
	}

	return groups
}

// GetGroupByKey retrieves a search filter group
// by a string.
func (s *SearchFilters) GetGroupByKey(key string) *SearchFilterGroup {
	// This is an optimization (this is basically a fixed hash map,
	// with hash(key) = position(key)
	switch key {
	case SearchKeySources:
		return (*s)[0]
	case SearchKeyASNS:
		return (*s)[1]
	case SearchKeyCommunities:
		return (*s)[2]
	case SearchKeyExtCommunities:
		return (*s)[3]
	case SearchKeyLargeCommunities:
		return (*s)[4]
	case SearchKeyAddrFamily:
		return (*s)[5]
	}
	return nil
}

// UpdateSourcesFromLookupRoute updates the source filter
func (s *SearchFilters) UpdateSourcesFromLookupRoute(r *LookupRoute) {
	// Add source
	s.GetGroupByKey(SearchKeySources).AddFilter(&SearchFilter{
		Name:  r.RouteServer.Name,
		Value: r.RouteServer.ID,
	})
}

// UpdateASNSFromLookupRoute updates the ASN filter
func (s *SearchFilters) UpdateASNSFromLookupRoute(r *LookupRoute) {
	// Add ASN from neighbor
	s.GetGroupByKey(SearchKeyASNS).AddFilter(&SearchFilter{
		Name:  r.Neighbor.Description,
		Value: r.Neighbor.ASN,
	})
}

// UpdateCommunitiesFromLookupRoute updates the communities filter
func (s *SearchFilters) UpdateCommunitiesFromLookupRoute(r *LookupRoute) {
	// Add communities
	communities := s.GetGroupByKey(SearchKeyCommunities)
	for _, c := range r.Route.BGP.Communities {
		communities.AddFilter(&SearchFilter{
			Name:  c.String(),
			Value: c,
		})
	}
	extCommunities := s.GetGroupByKey(SearchKeyExtCommunities)
	for _, c := range r.Route.BGP.ExtCommunities {
		extCommunities.AddFilter(&SearchFilter{
			Name:  c.String(),
			Value: c,
		})
	}
	largeCommunities := s.GetGroupByKey(SearchKeyLargeCommunities)
	for _, c := range r.Route.BGP.LargeCommunities {
		largeCommunities.AddFilter(&SearchFilter{
			Name:  c.String(),
			Value: c,
		})
	}
}

// UpdateFromLookupRoute updates a filter
// and its counters.
//
// Update filter struct to include route:
//   - Extract ASN, source, bgp communities,
//   - Find Filter in group, increment result count if required.
func (s *SearchFilters) UpdateFromLookupRoute(r *LookupRoute) {
	s.UpdateSourcesFromLookupRoute(r)
	s.UpdateASNSFromLookupRoute(r)
	s.UpdateCommunitiesFromLookupRoute(r)
}

// UpdateFromRoute updates a search filter, however as
// information of the route server or neighbor is not
// present, as this is not a lookup route, only
// communities are considered.
func (s *SearchFilters) UpdateFromRoute(r *Route) {

	// Add communities
	communities := s.GetGroupByKey(SearchKeyCommunities)
	for _, c := range r.BGP.Communities {
		communities.AddFilter(&SearchFilter{
			Name:  c.String(),
			Value: c,
		})
	}
	extCommunities := s.GetGroupByKey(SearchKeyExtCommunities)
	for _, c := range r.BGP.ExtCommunities {
		extCommunities.AddFilter(&SearchFilter{
			Name:  c.String(),
			Value: c,
		})
	}
	largeCommunities := s.GetGroupByKey(SearchKeyLargeCommunities)
	for _, c := range r.BGP.LargeCommunities {
		largeCommunities.AddFilter(&SearchFilter{
			Name:  c.String(),
			Value: c,
		})
	}
}

// SetFilterAddrFamily adds a filter to the addr family
// filter group if enabled.
func (s *SearchFilters) SetFilterAddrFamilies(ip4, ip6 bool) {
	if ip4 {
		s.addFilterAddrFamily(AddrFamilyIPv4)
	}
	if ip6 {
		s.addFilterAddrFamily(AddrFamilyIPv6)
	}
}

// Internal: set the actual addr family filter
func (s *SearchFilters) addFilterAddrFamily(af uint8) {
	name := "IPv4"
	if af == AddrFamilyIPv6 {
		name = "IPv6"
	}
	grp := s.GetGroupByKey(SearchKeyAddrFamily)
	grp.AddFilter(&SearchFilter{
		Name:        name,
		Value:       int(af),
		Cardinality: 1,
	})
}

// FiltersFromQuery builds a filter struct from
// query parameters.
//
// For example a query string of:
//
//	asns=2342,23123&communities=23:42&large_communities=23:42:42
//
// yields a filtering struct of
//
//	Groups[
//	    Group{"sources", []},
//	    Group{"asns", [Filter{Value: 2342},
//	                   Filter{Value: 23123}]},
//	    Group{"communities", ...
//	}
func FiltersFromQuery(query url.Values) (*SearchFilters, error) {
	queryFilters := NewSearchFilters()
	for key := range query {
		value := query.Get(key)
		switch key {
		case SearchKeySources:
			filters, err := parseQueryValueList(parseStringValue, value)
			if err != nil {
				return nil, err
			}
			queryFilters.GetGroupByKey(SearchKeySources).AddFilters(filters)

		case SearchKeyASNS:
			filters, err := parseQueryValueList(parseIntValue, value)
			if err != nil {
				return nil, err
			}
			queryFilters.GetGroupByKey(SearchKeyASNS).AddFilters(filters)

		case SearchKeyCommunities:
			filters, err := parseQueryValueList(parseCommunityValue, value)
			if err != nil {
				return nil, err
			}
			queryFilters.GetGroupByKey(SearchKeyCommunities).AddFilters(filters)

		case SearchKeyExtCommunities:
			filters, err := parseQueryValueList(parseExtCommunityValue, value)
			if err != nil {
				return nil, err
			}
			queryFilters.GetGroupByKey(SearchKeyExtCommunities).AddFilters(filters)

		case SearchKeyLargeCommunities:
			filters, err := parseQueryValueList(parseCommunityValue, value)
			if err != nil {
				return nil, err
			}
			queryFilters.GetGroupByKey(SearchKeyLargeCommunities).AddFilters(filters)

		case SearchKeyAddrFamily:
			// Parse as int values for address family
			filters, err := parseQueryValueList(parseIntValue, value)
			if err != nil {
				return nil, err
			}
			queryFilters.GetGroupByKey(SearchKeyAddrFamily).AddFilters(filters)
		}
	}
	return queryFilters, nil
}

// parseCommunityFilterText creates FilterValue from the
// text input which may be a api.Community or api.ExtCommunity.
func parseCommunityFilterText(text string) (string, *SearchFilter, error) {
	tokens := strings.Split(text, ":")
	if len(tokens) < 2 {
		return "", nil, fmt.Errorf("BGP community incomplete")
	}

	// Check if we are dealing with an ext. community
	maybeExt := false
	_, err := strconv.Atoi(tokens[0])
	if err != nil {
		maybeExt = true
	}

	// Parse filter value
	if maybeExt {
		filter, err := parseExtCommunityValue(text)
		if err != nil {
			return "", nil, err
		}
		return SearchKeyExtCommunities, filter, nil
	}

	filter, err := parseCommunityValue(text)
	if err != nil {
		return "", nil, fmt.Errorf("BGP community incomplete")
	}

	if len(tokens) == 2 {
		return SearchKeyCommunities, filter, nil
	}

	return SearchKeyLargeCommunities, filter, nil
}

// FiltersFromTokens parses the passed list of filters
// extracted from the query string and creates the filter.
func FiltersFromTokens(tokens []string) (*SearchFilters, error) {
	queryFilters := NewSearchFilters()
	for _, value := range tokens {
		if strings.HasPrefix(value, "#") { // Community query
			key, filter, err := parseCommunityFilterText(value[1:])
			if err != nil {
				return nil, err
			}
			queryFilters.GetGroupByKey(key).AddFilter(filter)
		}

	}
	return queryFilters, nil
}

// MatchRoute checks if a route matches all filters.
// Unless all filters are blank.
func (s *SearchFilters) MatchRoute(r Filterable) bool {
	sources := s.GetGroupByKey(SearchKeySources)
	if !sources.MatchAny(r) {
		return false
	}

	asns := s.GetGroupByKey(SearchKeyASNS)
	if !asns.MatchAny(r) {
		return false
	}

	communities := s.GetGroupByKey(SearchKeyCommunities)
	if !communities.MatchAll(r) {
		return false
	}

	extCommunities := s.GetGroupByKey(SearchKeyExtCommunities)
	if !extCommunities.MatchAll(r) {
		return false
	}

	largeCommunities := s.GetGroupByKey(SearchKeyLargeCommunities)
	if !largeCommunities.MatchAll(r) {
		return false
	}

	addrFamily := s.GetGroupByKey(SearchKeyAddrFamily)
	if !addrFamily.MatchAny(r) {
		return false
	}

	return true
}

// Combine two search filters
func (s *SearchFilters) Combine(other *SearchFilters) *SearchFilters {
	result := make(SearchFilters, len(*s))
	for id, group := range *s {
		otherGroup := (*other)[id]
		combined := &SearchFilterGroup{
			Key:     group.Key,
			Filters: []*SearchFilter{},
		}
		combined.Filters = append(combined.Filters, group.Filters...)
		for _, f := range otherGroup.Filters {
			if combined.Contains(f) {
				continue
			}
			combined.Filters = append(combined.Filters, f)
		}
		combined.rebuildIndex()
		result[id] = combined
	}

	return &result
}

// Sub makes a diff of two search filters
func (s *SearchFilters) Sub(other *SearchFilters) *SearchFilters {
	result := make(SearchFilters, len(*s))

	for id, group := range *s {
		otherGroup := (*other)[id]
		diff := &SearchFilterGroup{
			Key:     group.Key,
			Filters: []*SearchFilter{},
		}

		// Combine filters
		for _, f := range group.Filters {
			if otherGroup.Contains(f) {
				continue // Let's skip this
			}
			diff.Filters = append(diff.Filters, f)
		}

		diff.rebuildIndex()
		result[id] = diff
	}

	return &result
}

// MergeProperties merges two search filters
func (s *SearchFilters) MergeProperties(other *SearchFilters) {
	for id, group := range *s {
		otherGroup := (*other)[id]
		for _, filter := range group.Filters {
			otherFilter := otherGroup.FindFilter(filter)
			if otherFilter == nil {
				// Filter not present on other side, ignore this.
				continue
			}
			filter.Name = otherFilter.Name
			filter.Cardinality = otherFilter.Cardinality
		}
	}
}

// HasGroup checks if a group with a given key exists
// and filters are present.
func (s *SearchFilters) HasGroup(key string) bool {
	group := s.GetGroupByKey(key)
	return len(group.Filters) > 0
}

// A NeighborFilter includes only a name and ASN.
// We are using a slightly simpler solution for
// neighbor queries.
type NeighborFilter struct {
	name string
	asn  int
}

// NeighborFilterFromQuery constructs a NeighborFilter
// from query parameters.
//
// Right now we support filtering by name (partial match)
// and ASN.
//
// The latter is used to find related peers on all route servers.
func NeighborFilterFromQuery(q url.Values) *NeighborFilter {
	asn := 0
	name := q.Get("name")
	asnVal := q.Get("asn")
	if asnVal != "" {
		asn, _ = strconv.Atoi(asnVal)
	}

	filter := &NeighborFilter{
		name: name,
		asn:  asn,
	}
	return filter
}

// NeighborFilterFromQueryString decodes query values from
// string into a NeighborFilter.
//
// This is intended as a helper method to make testing easier.
func NeighborFilterFromQueryString(q string) *NeighborFilter {
	values, _ := url.ParseQuery(q)
	return NeighborFilterFromQuery(values)
}

// Match neighbor with filter: Check if the neighbor
// in question has the required parameters.
func (s *NeighborFilter) Match(neighbor *Neighbor) bool {
	if s.name != "" && neighbor.MatchName(s.name) {
		return true
	}
	if s.asn > 0 && neighbor.MatchASN(s.asn) {
		return true
	}
	return false
}
