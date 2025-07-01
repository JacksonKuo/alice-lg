package api

import (
	"strconv"
	"time"
)

// A Response is a general API response. All API responses
// contain meta information with API version and caching
// information.
type Response struct {
	Meta *Meta `json:"api"`
}

// ErrorResponse encodes an error message and code
type ErrorResponse struct {
	Message       string `json:"message"`
	Code          int    `json:"code"`
	Tag           string `json:"tag"`
	RouteserverID string `json:"routeserver_id"`
}

// CacheableResponse is a cache aware API response
type CacheableResponse interface {
	CacheTTL() time.Duration
}

// ConfigResponse is a response with client runtime configuration
type ConfigResponse struct {
	RejectReasons map[string]interface{} `json:"reject_reasons"`

	Noexport        Noexport               `json:"noexport"`
	NoexportReasons map[string]interface{} `json:"noexport_reasons"`

	RejectCandidates RejectCandidates `json:"reject_candidates"`

	Rpki Rpki `json:"rpki"`

	BGPCommunities          map[string]interface{} `json:"bgp_communities"`
	BGPBlackholeCommunities BGPCommunitiesSet      `json:"bgp_blackhole_communities"`

	NeighborsColumns      map[string]string `json:"neighbors_columns"`
	NeighborsColumnsOrder []string          `json:"neighbors_columns_order"`

	RoutesColumns      map[string]string `json:"routes_columns"`
	RoutesColumnsOrder []string          `json:"routes_columns_order"`

	LookupColumns      map[string]string `json:"lookup_columns"`
	LookupColumnsOrder []string          `json:"lookup_columns_order"`

	PrefixLookupEnabled bool `json:"prefix_lookup_enabled"`
}

// Noexport options
type Noexport struct {
	LoadOnDemand bool `json:"load_on_demand"`
}

// RejectCandidates contains a communities mapping
// of reasons for a rejection in the future.
type RejectCandidates struct {
	Communities map[string]interface{} `json:"communities"`
}

// Rpki is the validation status of a prefix
type Rpki struct {
	Enabled    bool       `json:"enabled"`
	Valid      [][]string `json:"valid"`
	Unknown    [][]string `json:"unknown"`
	NotChecked [][]string `json:"not_checked"`
	Invalid    [][]string `json:"invalid"`
}

// Meta contains response meta information
// like caching time and cache ttl or the API version
type Meta struct {
	Version         string           `json:"version"`
	CacheStatus     CacheStatus      `json:"cache_status"`
	ResultFromCache bool             `json:"result_from_cache"`
	TTL             time.Time        `json:"ttl"`
	StoreStatus     *StoreStatusMeta `json:"store_status,omitempty"`
}

// CacheStatus contains cache timing information.
type CacheStatus struct {
	CachedAt time.Time `json:"cached_at"`
	OrigTTL  int       `json:"orig_ttl"`
}

// SourceStatus is the current status of a source in
// a store.
type SourceStatus struct {
	RefreshInterval time.Duration `json:"refresh_interval"`
	LastRefresh     time.Time     `json:"last_refresh"`
	State           string        `json:"state"`
	Initialized     bool          `json:"initialized"`
}

// StoreStatus is meta data for a store
type StoreStatus struct {
	Initialized bool                     `json:"initialized"`
	Sources     map[string]*SourceStatus `json:"sources"`
}

// StoreStatusMeta is the meta response for all stores
type StoreStatusMeta struct {
	Routes    *StoreStatus `json:"routes,omitempty"`
	Neighbors *StoreStatus `json:"neighbors,omitempty"`
}

// Status ... TODO: ?
type Status struct {
	ServerTime   time.Time `json:"server_time"`
	LastReboot   time.Time `json:"last_reboot"`
	LastReconfig time.Time `json:"last_reconfig"`
	Message      string    `json:"message"`
	RouterID     string    `json:"router_id"`
	Version      string    `json:"version"`
	Backend      string    `json:"backend"`
}

// StatusResponse ??
type StatusResponse struct {
	Response
	Status Status `json:"status"`
}

// A RouteServer is a datasource with attributes.
type RouteServer struct {
	ID         string   `json:"id"`
	Type       string   `json:"type"`
	Name       string   `json:"name"`
	Group      string   `json:"group"`
	Blackholes []string `json:"blackholes"`

	Order int `json:"-"`
}

// RouteServers is a collection of routeservers.
type RouteServers []RouteServer

// Len implements sorting interface for routeservers
func (rs RouteServers) Len() int {
	return len(rs)
}

func (rs RouteServers) Less(i, j int) bool {
	return rs[i].Order < rs[j].Order
}

func (rs RouteServers) Swap(i, j int) {
	rs[i], rs[j] = rs[j], rs[i]
}

// A RouteServersResponse contains a list of routeservers.
type RouteServersResponse struct {
	RouteServers RouteServers `json:"routeservers"`
}

// A LookupRouteServer is a shorter representation of the
// route server data source.
type LookupRouteServer struct {
	ID   *string `json:"id"`
	Name string  `json:"name"`
}

// Community is a BGP community
type Community []int

func (com Community) String() string {
	if len(com) < 1 {
		return ""
	}
	s := ""
	for i, v := range com {
		if i > 0 {
			s += ":"
		}
		s += strconv.Itoa(v)
	}
	return s
}

// Communities is a collection of bgp communities
type Communities []Community

// Unique deduplicates communities.
/*
We can skip this. Worst case is, that the
cardinality is off.
func (communities Communities) Unique() Communities {
	seen := map[string]bool{}
	result := make(Communities, 0, len(communities))

	for _, com := range communities {
		key := com.String()
		if _, ok := seen[key]; !ok {
			// We have not seen this community yet
			result = append(result, com)
			seen[key] = true
		}
	}

	return result
}
*/

// ExtCommunity is a BGP extended community
type ExtCommunity []interface{}

func (com ExtCommunity) String() string {
	if len(com) < 1 {
		return ""
	}
	res := ""
	for i, v := range com {
		if i == 0 {
			res += v.(string)
			continue
		}
		if i > 0 {
			res += ":"
		}
		res += strconv.Itoa(v.(int))
	}
	return res
}

// ExtCommunities is a collection of extended bgp communities.
type ExtCommunities []ExtCommunity

// Unique deduplicates extended communities.
/*
func (communities ExtCommunities) Unique() ExtCommunities {
	seen := map[string]bool{}
	result := make(ExtCommunities, 0, len(communities))

	for _, com := range communities {
		key := com.String()
		if _, ok := seen[key]; !ok {
			// We have not seen this community yet
			result = append(result, com)
			seen[key] = true
		}
	}

	return result
}
*/

// BGPInfo is a set of BGP attributes
type BGPInfo struct {
	Origin           *string        `json:"origin"`
	AsPath           []int          `json:"as_path"`
	NextHop          *string        `json:"next_hop"`
	Communities      Communities    `json:"communities"`
	LargeCommunities Communities    `json:"large_communities"`
	ExtCommunities   ExtCommunities `json:"ext_communities"`
	LocalPref        int            `json:"local_pref"`
	Med              int            `json:"med"`
	OTC              *int           `json:"otc"`
}

// HasCommunity checks for the presence of a BGP community.
func (bgp *BGPInfo) HasCommunity(community Community) bool {
	if len(community) != 2 {
		return false // This can never match.
	}

	for _, com := range bgp.Communities {
		if len(com) != len(community) {
			continue // This can't match.
		}

		if com[0] == community[0] &&
			com[1] == community[1] {
			return true
		}
	}

	return false
}

// HasExtCommunity checks for the presence of an
// extended community.
func (bgp *BGPInfo) HasExtCommunity(community ExtCommunity) bool {
	if len(community) != 3 {
		return false // This can never match.
	}

	for _, com := range bgp.ExtCommunities {
		if len(com) != len(community) {
			continue // This can't match.
		}

		if com[0] == community[0] &&
			com[1] == community[1] &&
			com[2] == community[2] {
			return true
		}
	}

	return false
}

// HasLargeCommunity checks for the presence of a large community.
func (bgp *BGPInfo) HasLargeCommunity(community Community) bool {
	// TODO: This is an almost 1:1 match to the function above.
	if len(community) != 3 {
		return false // This can never match.
	}

	for _, com := range bgp.LargeCommunities {
		if len(com) != len(community) {
			continue // This can't match.
		}

		if com[0] == community[0] &&
			com[1] == community[1] &&
			com[2] == community[2] {
			return true
		}
	}

	return false
}
