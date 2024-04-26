// Copyright (C) MongoDB, Inc. 2024-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package driverutil

import (
	"errors"
	"fmt"
	"math"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/internal/bsonutil"
	"go.mongodb.org/mongo-driver/internal/handshake"
	"go.mongodb.org/mongo-driver/internal/ptrutil"
	"go.mongodb.org/mongo-driver/mongo/address"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/tag"
	"go.mongodb.org/mongo-driver/x/mongo/driver/description"
)

// CompositeServerSelector combines multiple selectors into a single selector by
// applying them in order to the candidates list.
//
// For example, if the initial candidates list is [s0, s1, s2, s3] and two
// selectors are provided where the first matches s0 and s1 and the second
// matches s1 and s2, the following would occur during server selection:
//
// 1. firstSelector([s0, s1, s2, s3]) -> [s0, s1]
// 2. secondSelector([s0, s1]) -> [s1]
//
// The final list of candidates returned by the composite selector would be
// [s1].
type CompositeServerSelector struct {
	Selectors []description.ServerSelector
}

var _ description.ServerSelector = &CompositeServerSelector{}

// SelectServer combines multiple selectors into a single selector.
func (selector *CompositeServerSelector) SelectServer(
	topo description.Topology,
	candidates []description.Server,
) ([]description.Server, error) {
	var err error
	for _, sel := range selector.Selectors {
		candidates, err = sel.SelectServer(topo, candidates)
		if err != nil {
			return nil, err
		}
	}

	return candidates, nil
}

// LatencyServerSelector creates a ServerSelector which selects servers based on
// their average RTT values.
type LatencyServerSelector struct {
	Latency time.Duration
}

var _ description.ServerSelector = &LatencyServerSelector{}

// SelectServer selects servers based on average RTT.
func (selector *LatencyServerSelector) SelectServer(
	topo description.Topology,
	candidates []description.Server,
) ([]description.Server, error) {
	if selector.Latency < 0 {
		return candidates, nil
	}
	if topo.Kind == description.TopologyKindLoadBalanced {
		// In LoadBalanced mode, there should only be one server in the topology and
		// it must be selected.
		return candidates, nil
	}

	switch len(candidates) {
	case 0, 1:
		return candidates, nil
	default:
		min := time.Duration(math.MaxInt64)
		for _, candidate := range candidates {
			if candidate.AverageRTTSet {
				if candidate.AverageRTT < min {
					min = candidate.AverageRTT
				}
			}
		}

		if min == math.MaxInt64 {
			return candidates, nil
		}

		max := min + selector.Latency

		viableIndexes := make([]int, 0, len(candidates))
		for i, candidate := range candidates {
			if candidate.AverageRTTSet {
				if candidate.AverageRTT <= max {
					viableIndexes = append(viableIndexes, i)
				}
			}
		}
		if len(viableIndexes) == len(candidates) {
			return candidates, nil
		}
		result := make([]description.Server, len(viableIndexes))
		for i, idx := range viableIndexes {
			result[i] = candidates[idx]
		}
		return result, nil
	}
}

// ReadPrefServerSelector selects servers based on the provided read preference.
type ReadPrefServerSelector struct {
	ReadPref          *readpref.ReadPref
	IsOutputAggregate bool
}

var _ description.ServerSelector = &ReadPrefServerSelector{}

// SelectServer selects servers based on read preference.
func (selector *ReadPrefServerSelector) SelectServer(
	topo description.Topology,
	candidates []description.Server,
) ([]description.Server, error) {
	if topo.Kind == description.TopologyKindLoadBalanced {
		// In LoadBalanced mode, there should only be one server in the topology and
		// it must be selected. We check this before checking MaxStaleness support
		// because there's no monitoring in this mode, so the candidate server
		// wouldn't have a wire version set, which would result in an error.
		return candidates, nil
	}

	switch topo.Kind {
	case description.TopologyKindSingle:
		return candidates, nil
	case description.TopologyKindReplicaSetNoPrimary, description.TopologyKindReplicaSetWithPrimary:
		return selectForReplicaSet(selector.ReadPref, selector.IsOutputAggregate, topo, candidates)
	case description.TopologyKindSharded:
		return selectByKind(candidates, description.ServerKindMongos), nil
	}

	return nil, nil
}

// WriteServerSelector selects all the writable servers.
type WriteServerSelector struct{}

var _ description.ServerSelector = &WriteServerSelector{}

// SelectServer selects all writable servers.
func (selector *WriteServerSelector) SelectServer(
	topo description.Topology,
	candidates []description.Server,
) ([]description.Server, error) {
	switch topo.Kind {
	case description.TopologyKindSingle, description.TopologyKindLoadBalanced:
		return candidates, nil
	default:
		// Determine the capacity of the results slice.
		selected := 0
		for _, candidate := range candidates {
			switch candidate.Kind {
			case description.ServerKindMongos, description.ServerKindRSPrimary, description.ServerKindStandalone:
				selected++
			}
		}

		// Append candidates to the results slice.
		result := make([]description.Server, 0, selected)
		for _, candidate := range candidates {
			switch candidate.Kind {
			case description.ServerKindMongos, description.ServerKindRSPrimary, description.ServerKindStandalone:
				result = append(result, candidate)
			}
		}
		return result, nil
	}
}

// ServerSelector is an interface implemented by types that can perform server
// selection given a topology description and list of candidate servers. The
// selector should filter the provided candidates list and return a subset that
// matches some criteria.
type ServerSelector interface {
	SelectServer(description.Topology, []description.Server) ([]description.Server, error)
}

// ServerSelectorFunc is a function that can be used as a ServerSelector.
type ServerSelectorFunc func(description.Topology, []description.Server) ([]description.Server, error)

// SelectServer implements the ServerSelector interface.
func (ssf ServerSelectorFunc) SelectServer(
	t description.Topology,
	s []description.Server,
) ([]description.Server, error) {
	return ssf(t, s)
}

func sliceStringEqual(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	for i, v := range a {
		if v != b[i] {
			return false
		}
	}

	return true
}

func equalWireVersion(wv1, wv2 *description.VersionRange) bool {
	if wv1 == nil && wv2 == nil {
		return true
	}

	if wv1 == nil || wv2 == nil {
		return false
	}

	return wv1.Min == wv2.Min && wv1.Max == wv2.Max
}

// EqualServers compares two server descriptions and returns true if they are
// equal.
func EqualServers(srv1, srv2 description.Server) bool {
	if srv1.CanonicalAddr.String() != srv2.CanonicalAddr.String() {
		return false
	}

	if !sliceStringEqual(srv1.Arbiters, srv2.Arbiters) {
		return false
	}

	if !sliceStringEqual(srv1.Hosts, srv2.Hosts) {
		return false
	}

	if !sliceStringEqual(srv1.Passives, srv2.Passives) {
		return false
	}

	if srv1.Primary != srv2.Primary {
		return false
	}

	if srv1.SetName != srv2.SetName {
		return false
	}

	if srv1.Kind != srv2.Kind {
		return false
	}

	if srv1.LastError != nil || srv2.LastError != nil {
		if srv1.LastError == nil || srv2.LastError == nil {
			return false
		}
		if srv1.LastError.Error() != srv2.LastError.Error() {
			return false
		}
	}

	//if !s.WireVersion.Equals(other.WireVersion) {
	if !equalWireVersion(srv1.WireVersion, srv2.WireVersion) {
		return false
	}

	if len(srv1.Tags) != len(srv2.Tags) || !srv1.Tags.ContainsAll(srv2.Tags) {
		return false
	}

	if srv1.SetVersion != srv2.SetVersion {
		return false
	}

	if srv1.ElectionID != srv2.ElectionID {
		return false
	}

	if ptrutil.CompareInt64(srv1.SessionTimeoutMinutes, srv2.SessionTimeoutMinutes) != 0 {
		return false
	}

	// If TopologyVersion is nil for both servers, CompareToIncoming will return -1 because it assumes that the
	// incoming response is newer. We want the descriptions to be considered equal in this case, though, so an
	// explicit check is required.
	if srv1.TopologyVersion == nil && srv2.TopologyVersion == nil {
		return true
	}

	//return s.TopologyVersion.CompareToIncoming(other.TopologyVersion) == 0

	return CompareTopologyVersions(srv1.TopologyVersion, srv2.TopologyVersion) == 0
}

// IsServerLoadBalanced checks if a description.Server describes a server that
// is load balanced.
func IsServerLoadBalanced(srv description.Server) bool {
	return srv.Kind == description.ServerKindLoadBalancer || srv.ServiceID != nil
}

// stringSliceFromRawElement decodes the provided BSON element into a []string.
// This internally calls StringSliceFromRawValue on the element's value. The
// error conditions outlined in that function's documentation apply for this
// function as well.
func stringSliceFromRawElement(element bson.RawElement) ([]string, error) {
	return bsonutil.StringSliceFromRawValue(element.Key(), element.Value())
}

func decodeStringMap(element bson.RawElement, name string) (map[string]string, error) {
	doc, ok := element.Value().DocumentOK()
	if !ok {
		return nil, fmt.Errorf("expected '%s' to be a document but it's a BSON %s", name, element.Value().Type)
	}
	elements, err := doc.Elements()
	if err != nil {
		return nil, err
	}
	m := make(map[string]string)
	for _, element := range elements {
		key := element.Key()
		value, ok := element.Value().StringValueOK()
		if !ok {
			return nil, fmt.Errorf("expected '%s' to be a document of strings, but found a BSON %s", name, element.Value().Type)
		}
		m[key] = value
	}
	return m, nil
}

// NewTopologyVersion creates a TopologyVersion based on doc
func NewTopologyVersion(doc bson.Raw) (*description.TopologyVersion, error) {
	elements, err := doc.Elements()
	if err != nil {
		return nil, err
	}
	var tv description.TopologyVersion
	var ok bool
	for _, element := range elements {
		switch element.Key() {
		case "processId":
			tv.ProcessID, ok = element.Value().ObjectIDOK()
			if !ok {
				return nil, fmt.Errorf("expected 'processId' to be a objectID but it's a BSON %s", element.Value().Type)
			}
		case "counter":
			tv.Counter, ok = element.Value().Int64OK()
			if !ok {
				return nil, fmt.Errorf("expected 'counter' to be an int64 but it's a BSON %s", element.Value().Type)
			}
		}
	}
	return &tv, nil
}

// NewVersionRange creates a new VersionRange given a min and a max.
func NewVersionRange(min, max int32) description.VersionRange {
	return description.VersionRange{Min: min, Max: max}
}

// VersionRangeIncludes returns a bool indicating whether the supplied integer
// is included in the range.
func VersionRangeIncludes(versionRange description.VersionRange, v int32) bool {
	return v >= versionRange.Min && v <= versionRange.Max
}

// CompareTopologyVersions compares the receiver, which represents the currently
// known TopologyVersion for a server, to an incoming TopologyVersion extracted
// from a server command response.
//
// This returns -1 if the receiver version is less than the response, 0 if the
// versions are equal, and 1 if the receiver version is greater than the
// response. This comparison is not commutative.
func CompareTopologyVersions(receiver, response *description.TopologyVersion) int {
	if receiver == nil || response == nil {
		return -1
	}
	if receiver.ProcessID != response.ProcessID {
		return -1
	}
	if receiver.Counter == response.Counter {
		return 0
	}
	if receiver.Counter < response.Counter {
		return -1
	}
	return 1
}

func NewEventServerDescription(srv description.Server) event.ServerDescription {
	evtSrv := event.ServerDescription{
		Addr:                  srv.Addr,
		Arbiters:              srv.Arbiters,
		Compression:           srv.Compression,
		CanonicalAddr:         srv.CanonicalAddr,
		ElectionID:            srv.ElectionID,
		IsCryptd:              srv.IsCryptd,
		HelloOK:               srv.HelloOK,
		Hosts:                 srv.Hosts,
		Kind:                  srv.Kind.String(),
		LastWriteTime:         srv.LastWriteTime,
		MaxBatchCount:         srv.MaxBatchCount,
		MaxDocumentSize:       srv.MaxDocumentSize,
		MaxMessageSize:        srv.MaxMessageSize,
		Members:               srv.Members,
		Passive:               srv.Passive,
		Passives:              srv.Passives,
		Primary:               srv.Primary,
		ReadOnly:              srv.ReadOnly,
		ServiceID:             srv.ServiceID,
		SessionTimeoutMinutes: srv.SessionTimeoutMinutes,
		SetName:               srv.SetName,
		SetVersion:            srv.SetVersion,
		Tags:                  srv.Tags,
	}

	if srv.WireVersion != nil {
		evtSrv.MaxWireVersion = srv.WireVersion.Max
		evtSrv.MinWireVersion = srv.WireVersion.Min
	}

	if srv.TopologyVersion != nil {
		evtSrv.TopologyVersionProcessID = srv.TopologyVersion.ProcessID
		evtSrv.TopologyVersionCounter = srv.TopologyVersion.Counter
	}

	return evtSrv
}

func NewEventServerTopology(topo description.Topology) event.TopologyDescription {
	evtSrvs := make([]event.ServerDescription, len(topo.Servers))
	for idx, srv := range topo.Servers {
		evtSrvs[idx] = NewEventServerDescription(srv)
	}

	evtTopo := event.TopologyDescription{
		Servers:               evtSrvs,
		SetName:               topo.SetName,
		Kind:                  topo.Kind.String(),
		SessionTimeoutMinutes: topo.SessionTimeoutMinutes,
		CompatibilityErr:      topo.CompatibilityErr,
	}

	return evtTopo
}

// NewServerDescription creates a new server description from the given hello
// command response.
func NewServerDescription(addr address.Address, response bson.Raw) description.Server {
	desc := description.Server{Addr: addr, CanonicalAddr: addr, LastUpdateTime: time.Now().UTC()}
	elements, err := response.Elements()
	if err != nil {
		desc.LastError = err
		return desc
	}
	var ok bool
	var isReplicaSet, isWritablePrimary, hidden, secondary, arbiterOnly bool
	var msg string
	var versionRange description.VersionRange
	for _, element := range elements {
		switch element.Key() {
		case "arbiters":
			var err error
			desc.Arbiters, err = stringSliceFromRawElement(element)
			if err != nil {
				desc.LastError = err
				return desc
			}
		case "arbiterOnly":
			arbiterOnly, ok = element.Value().BooleanOK()
			if !ok {
				desc.LastError = fmt.Errorf("expected 'arbiterOnly' to be a boolean but it's a BSON %s", element.Value().Type)
				return desc
			}
		case "compression":
			var err error
			desc.Compression, err = stringSliceFromRawElement(element)
			if err != nil {
				desc.LastError = err
				return desc
			}
		case "electionId":
			desc.ElectionID, ok = element.Value().ObjectIDOK()
			if !ok {
				desc.LastError = fmt.Errorf("expected 'electionId' to be a objectID but it's a BSON %s", element.Value().Type)
				return desc
			}
		case "iscryptd":
			desc.IsCryptd, ok = element.Value().BooleanOK()
			if !ok {
				desc.LastError = fmt.Errorf("expected 'iscryptd' to be a boolean but it's a BSON %s", element.Value().Type)
				return desc
			}
		case "helloOk":
			desc.HelloOK, ok = element.Value().BooleanOK()
			if !ok {
				desc.LastError = fmt.Errorf("expected 'helloOk' to be a boolean but it's a BSON %s", element.Value().Type)
				return desc
			}
		case "hidden":
			hidden, ok = element.Value().BooleanOK()
			if !ok {
				desc.LastError = fmt.Errorf("expected 'hidden' to be a boolean but it's a BSON %s", element.Value().Type)
				return desc
			}
		case "hosts":
			var err error
			desc.Hosts, err = stringSliceFromRawElement(element)
			if err != nil {
				desc.LastError = err
				return desc
			}
		case "isWritablePrimary":
			isWritablePrimary, ok = element.Value().BooleanOK()
			if !ok {
				desc.LastError = fmt.Errorf("expected 'isWritablePrimary' to be a boolean but it's a BSON %s", element.Value().Type)
				return desc
			}
		case handshake.LegacyHelloLowercase:
			isWritablePrimary, ok = element.Value().BooleanOK()
			if !ok {
				desc.LastError = fmt.Errorf("expected legacy hello to be a boolean but it's a BSON %s", element.Value().Type)
				return desc
			}
		case "isreplicaset":
			isReplicaSet, ok = element.Value().BooleanOK()
			if !ok {
				desc.LastError = fmt.Errorf("expected 'isreplicaset' to be a boolean but it's a BSON %s", element.Value().Type)
				return desc
			}
		case "lastWrite":
			lastWrite, ok := element.Value().DocumentOK()
			if !ok {
				desc.LastError = fmt.Errorf("expected 'lastWrite' to be a document but it's a BSON %s", element.Value().Type)
				return desc
			}
			dateTime, err := lastWrite.LookupErr("lastWriteDate")
			if err == nil {
				dt, ok := dateTime.DateTimeOK()
				if !ok {
					desc.LastError = fmt.Errorf("expected 'lastWriteDate' to be a datetime but it's a BSON %s", dateTime.Type)
					return desc
				}
				desc.LastWriteTime = time.Unix(dt/1000, dt%1000*1000000).UTC()
			}
		case "logicalSessionTimeoutMinutes":
			i64, ok := element.Value().AsInt64OK()
			if !ok {
				desc.LastError = fmt.Errorf("expected 'logicalSessionTimeoutMinutes' to be an integer but it's a BSON %s", element.Value().Type)
				return desc
			}

			desc.SessionTimeoutMinutes = &i64
		case "maxBsonObjectSize":
			i64, ok := element.Value().AsInt64OK()
			if !ok {
				desc.LastError = fmt.Errorf("expected 'maxBsonObjectSize' to be an integer but it's a BSON %s", element.Value().Type)
				return desc
			}
			desc.MaxDocumentSize = uint32(i64)
		case "maxMessageSizeBytes":
			i64, ok := element.Value().AsInt64OK()
			if !ok {
				desc.LastError = fmt.Errorf("expected 'maxMessageSizeBytes' to be an integer but it's a BSON %s", element.Value().Type)
				return desc
			}
			desc.MaxMessageSize = uint32(i64)
		case "maxWriteBatchSize":
			i64, ok := element.Value().AsInt64OK()
			if !ok {
				desc.LastError = fmt.Errorf("expected 'maxWriteBatchSize' to be an integer but it's a BSON %s", element.Value().Type)
				return desc
			}
			desc.MaxBatchCount = uint32(i64)
		case "me":
			me, ok := element.Value().StringValueOK()
			if !ok {
				desc.LastError = fmt.Errorf("expected 'me' to be a string but it's a BSON %s", element.Value().Type)
				return desc
			}
			desc.CanonicalAddr = address.Address(me).Canonicalize()
		case "maxWireVersion":
			verMax, ok := element.Value().AsInt64OK()
			versionRange.Max = int32(verMax)
			if !ok {
				desc.LastError = fmt.Errorf("expected 'maxWireVersion' to be an integer but it's a BSON %s", element.Value().Type)
				return desc
			}
		case "minWireVersion":
			verMin, ok := element.Value().AsInt64OK()
			versionRange.Min = int32(verMin)
			if !ok {
				desc.LastError = fmt.Errorf("expected 'minWireVersion' to be an integer but it's a BSON %s", element.Value().Type)
				return desc
			}
		case "msg":
			msg, ok = element.Value().StringValueOK()
			if !ok {
				desc.LastError = fmt.Errorf("expected 'msg' to be a string but it's a BSON %s", element.Value().Type)
				return desc
			}
		case "ok":
			okay, ok := element.Value().AsInt64OK()
			if !ok {
				desc.LastError = fmt.Errorf("expected 'ok' to be a boolean but it's a BSON %s", element.Value().Type)
				return desc
			}
			if okay != 1 {
				desc.LastError = errors.New("not ok")
				return desc
			}
		case "passives":
			var err error
			desc.Passives, err = stringSliceFromRawElement(element)
			if err != nil {
				desc.LastError = err
				return desc
			}
		case "passive":
			desc.Passive, ok = element.Value().BooleanOK()
			if !ok {
				desc.LastError = fmt.Errorf("expected 'passive' to be a boolean but it's a BSON %s", element.Value().Type)
				return desc
			}
		case "primary":
			primary, ok := element.Value().StringValueOK()
			if !ok {
				desc.LastError = fmt.Errorf("expected 'primary' to be a string but it's a BSON %s", element.Value().Type)
				return desc
			}
			desc.Primary = address.Address(primary)
		case "readOnly":
			desc.ReadOnly, ok = element.Value().BooleanOK()
			if !ok {
				desc.LastError = fmt.Errorf("expected 'readOnly' to be a boolean but it's a BSON %s", element.Value().Type)
				return desc
			}
		case "secondary":
			secondary, ok = element.Value().BooleanOK()
			if !ok {
				desc.LastError = fmt.Errorf("expected 'secondary' to be a boolean but it's a BSON %s", element.Value().Type)
				return desc
			}
		case "serviceId":
			oid, ok := element.Value().ObjectIDOK()
			if !ok {
				desc.LastError = fmt.Errorf("expected 'serviceId' to be an ObjectId but it's a BSON %s", element.Value().Type)
			}
			desc.ServiceID = &oid
		case "setName":
			desc.SetName, ok = element.Value().StringValueOK()
			if !ok {
				desc.LastError = fmt.Errorf("expected 'setName' to be a string but it's a BSON %s", element.Value().Type)
				return desc
			}
		case "setVersion":
			i64, ok := element.Value().AsInt64OK()
			if !ok {
				desc.LastError = fmt.Errorf("expected 'setVersion' to be an integer but it's a BSON %s", element.Value().Type)
				return desc
			}
			desc.SetVersion = uint32(i64)
		case "tags":
			m, err := decodeStringMap(element, "tags")
			if err != nil {
				desc.LastError = err
				return desc
			}
			desc.Tags = tag.NewTagSetFromMap(m)
		case "topologyVersion":
			doc, ok := element.Value().DocumentOK()
			if !ok {
				desc.LastError = fmt.Errorf("expected 'topologyVersion' to be a document but it's a BSON %s", element.Value().Type)
				return desc
			}

			desc.TopologyVersion, err = NewTopologyVersion(doc)
			if err != nil {
				desc.LastError = err
				return desc
			}
		}
	}

	for _, host := range desc.Hosts {
		desc.Members = append(desc.Members, address.Address(host).Canonicalize())
	}

	for _, passive := range desc.Passives {
		desc.Members = append(desc.Members, address.Address(passive).Canonicalize())
	}

	for _, arbiter := range desc.Arbiters {
		desc.Members = append(desc.Members, address.Address(arbiter).Canonicalize())
	}

	desc.Kind = description.ServerKindStandalone

	if isReplicaSet {
		desc.Kind = description.ServerKindRSGhost
	} else if desc.SetName != "" {
		if isWritablePrimary {
			desc.Kind = description.ServerKindRSPrimary
		} else if hidden {
			desc.Kind = description.ServerKindRSMember
		} else if secondary {
			desc.Kind = description.ServerKindRSSecondary
		} else if arbiterOnly {
			desc.Kind = description.ServerKindRSArbiter
		} else {
			desc.Kind = description.ServerKindRSMember
		}
	} else if msg == "isdbgrid" {
		desc.Kind = description.ServerKindMongos
	}

	desc.WireVersion = &versionRange

	return desc
}

func verifyMaxStaleness(rp *readpref.ReadPref, topo description.Topology) error {
	maxStaleness, set := rp.MaxStaleness()
	if !set {
		return nil
	}

	if maxStaleness < 90*time.Second {
		return fmt.Errorf("max staleness (%s) must be greater than or equal to 90s", maxStaleness)
	}

	if len(topo.Servers) < 1 {
		// Maybe we should return an error here instead?
		return nil
	}

	// we'll assume all candidates have the same heartbeat interval.
	s := topo.Servers[0]
	idleWritePeriod := 10 * time.Second

	if maxStaleness < s.HeartbeatInterval+idleWritePeriod {
		return fmt.Errorf(
			"max staleness (%s) must be greater than or equal to the heartbeat interval (%s) plus idle write period (%s)",
			maxStaleness, s.HeartbeatInterval, idleWritePeriod,
		)
	}

	return nil
}

func selectByKind(candidates []description.Server, kind description.ServerKind) []description.Server {
	// Record the indices of viable candidates first and then append those to the returned slice
	// to avoid appending costly Server structs directly as an optimization.
	viableIndexes := make([]int, 0, len(candidates))
	for i, s := range candidates {
		if s.Kind == kind {
			viableIndexes = append(viableIndexes, i)
		}
	}
	if len(viableIndexes) == len(candidates) {
		return candidates
	}
	result := make([]description.Server, len(viableIndexes))
	for i, idx := range viableIndexes {
		result[i] = candidates[idx]
	}
	return result
}

func selectSecondaries(rp *readpref.ReadPref, candidates []description.Server) []description.Server {
	secondaries := selectByKind(candidates, description.ServerKindRSSecondary)
	if len(secondaries) == 0 {
		return secondaries
	}
	if maxStaleness, set := rp.MaxStaleness(); set {
		primaries := selectByKind(candidates, description.ServerKindRSPrimary)
		if len(primaries) == 0 {
			baseTime := secondaries[0].LastWriteTime
			for i := 1; i < len(secondaries); i++ {
				if secondaries[i].LastWriteTime.After(baseTime) {
					baseTime = secondaries[i].LastWriteTime
				}
			}

			var selected []description.Server
			for _, secondary := range secondaries {
				estimatedStaleness := baseTime.Sub(secondary.LastWriteTime) + secondary.HeartbeatInterval
				if estimatedStaleness <= maxStaleness {
					selected = append(selected, secondary)
				}
			}

			return selected
		}

		primary := primaries[0]

		var selected []description.Server
		for _, secondary := range secondaries {
			estimatedStaleness := secondary.LastUpdateTime.Sub(secondary.LastWriteTime) -
				primary.LastUpdateTime.Sub(primary.LastWriteTime) + secondary.HeartbeatInterval
			if estimatedStaleness <= maxStaleness {
				selected = append(selected, secondary)
			}
		}
		return selected
	}

	return secondaries
}

func selectByTagSet(candidates []description.Server, tagSets []tag.Set) []description.Server {
	if len(tagSets) == 0 {
		return candidates
	}

	for _, ts := range tagSets {
		// If this tag set is empty, we can take a fast path because the empty list
		// is a subset of all tag sets, so all candidate servers will be selected.
		if len(ts) == 0 {
			return candidates
		}

		var results []description.Server
		for _, s := range candidates {
			// ts is non-empty, so only servers with a non-empty set of tags need to be checked.
			if len(s.Tags) > 0 && s.Tags.ContainsAll(ts) {
				results = append(results, s)
			}
		}

		if len(results) > 0 {
			return results
		}
	}

	return []description.Server{}
}

func selectForReplicaSet(
	rp *readpref.ReadPref,
	isOutputAggregate bool,
	topo description.Topology,
	candidates []description.Server,
) ([]description.Server, error) {
	if err := verifyMaxStaleness(rp, topo); err != nil {
		return nil, err
	}

	// If underlying operation is an aggregate with an output stage, only apply read preference
	// if all candidates are 5.0+. Otherwise, operate under primary read preference.
	if isOutputAggregate {
		for _, s := range candidates {
			if s.WireVersion.Max < 13 {
				return selectByKind(candidates, description.ServerKindRSPrimary), nil
			}
		}
	}

	switch rp.Mode() {
	case readpref.PrimaryMode:
		return selectByKind(candidates, description.ServerKindRSPrimary), nil
	case readpref.PrimaryPreferredMode:
		selected := selectByKind(candidates, description.ServerKindRSPrimary)

		if len(selected) == 0 {
			selected = selectSecondaries(rp, candidates)
			return selectByTagSet(selected, rp.TagSets()), nil
		}

		return selected, nil
	case readpref.SecondaryPreferredMode:
		selected := selectSecondaries(rp, candidates)
		selected = selectByTagSet(selected, rp.TagSets())
		if len(selected) > 0 {
			return selected, nil
		}
		return selectByKind(candidates, description.ServerKindRSPrimary), nil
	case readpref.SecondaryMode:
		selected := selectSecondaries(rp, candidates)
		return selectByTagSet(selected, rp.TagSets()), nil
	case readpref.NearestMode:
		selected := selectByKind(candidates, description.ServerKindRSPrimary)
		selected = append(selected, selectSecondaries(rp, candidates)...)
		return selectByTagSet(selected, rp.TagSets()), nil
	}

	return nil, fmt.Errorf("unsupported mode: %d", rp.Mode())
}
