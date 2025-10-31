package config

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/alice-lg/alice-lg/pkg/api"
	"github.com/alice-lg/alice-lg/pkg/decoders"
)

// ErrInvalidCommunity creates an invalid community error
func ErrInvalidCommunity(s string) error {
	return fmt.Errorf("invalid community: %s", s)
}

// Helper parse communities from a section body
func parseAndMergeCommunities(
	communities api.BGPCommunityMap, body string,
) api.BGPCommunityMap {

	// Parse and merge communities
	lines := strings.Split(body, "\n")
	for _, line := range lines {
		kv := strings.SplitN(line, "=", 2)
		if len(kv) != 2 {
			log.Println("Skipping malformed BGP community:", line)
			continue
		}

		community := strings.TrimSpace(kv[0])
		label := strings.TrimSpace(kv[1])
		communities.Set(community, label)
	}

	return communities
}

// Parse a communities set with ranged communities
func parseRangeCommunitiesSet(body string) (*api.BGPCommunitiesSet, error) {
	comms := []api.BGPCommunityRange{}
	large := []api.BGPCommunityRange{}
	ext := []api.BGPCommunityRange{}

	lines := strings.Split(body, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue // Empty
		}
		if strings.HasPrefix(line, "#") {
			continue // Comment
		}
		comm, err := parseRangeCommunity(line)
		if err != nil {
			return nil, err
		}
		switch comm.Type() {
		case api.BGPCommunityTypeStd:
			comms = append(comms, comm)
		case api.BGPCommunityTypeLarge:
			large = append(large, comm)
		case api.BGPCommunityTypeExt:
			ext = append(ext, comm)
		}
	}

	set := &api.BGPCommunitiesSet{
		Standard: comms,
		Large:    large,
		Extended: ext,
	}
	return set, nil
}

func parseRangeCommunity(s string) (api.BGPCommunityRange, error) {
	tokens := strings.Split(s, ":")
	if len(tokens) < 2 {
		return nil, ErrInvalidCommunity(s)
	}

	// Extract ranges and make uniform structure
	parts := make([][]string, 0, len(tokens))
	for _, t := range tokens {
		values := strings.SplitN(t, "-", 2)
		if len(values) == 0 {
			return nil, ErrInvalidCommunity(s)
		}
		if len(values) == 1 {
			parts = append(parts, []string{values[0], values[0]})
		} else {
			parts = append(parts, []string{values[0], values[1]})
		}
	}
	if len(parts) <= 1 {
		return nil, ErrInvalidCommunity(s)
	}

	// Check if this might be an ext community
	isExt := false
	if _, err := strconv.Atoi(parts[0][0]); err != nil {
		isExt = true // At least it looks like...
	}

	if isExt && len(parts) != 3 {
		return nil, ErrInvalidCommunity(s)
	}
	if isExt {
		return api.BGPCommunityRange{
			[]string{parts[0][0], parts[0][0]},
			decoders.IntListFromStrings(parts[1]),
			decoders.IntListFromStrings(parts[2]),
		}, nil
	}
	comm := api.BGPCommunityRange{}
	for _, p := range parts {
		comm = append(comm, decoders.IntListFromStrings(p))
	}
	return comm, nil
}

// Parse rejection candidate section
func parseRejectionCandidateCommunities(comms api.BGPCommunityMap, s string) error {
	lines := strings.Split(s, "\n")
	n := 0
	for _, line := range lines {
		kv := strings.SplitN(line, "=", 2)
		if len(kv) != 2 {
			log.Println("Skipping malformed reject candidate BGP community:", line)
			continue
		}

		key := strings.TrimSpace(kv[0])
		if key != "communities" {
			log.Printf("unexpected key '%s' in section 'rejection_candidates'", key)
			continue
		}

		value := strings.TrimSpace(kv[1])
		for _, c := range strings.Split(value, ",") {
			n += 1
			comms.Set(c, fmt.Sprintf("reject-candidate-%d", n))
		}
	}

	return nil
}
