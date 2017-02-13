package descutil

import (
	"fmt"
	"time"

	"github.com/10gen/mongo-go-driver/desc"
	"github.com/10gen/mongo-go-driver/internal"
)

// BuildServerDesc builds a desc.Server from an endpoint, IsMasterResult, and a BuildInfoResult.
func BuildServerDesc(endpoint desc.Endpoint, isMasterResult *internal.IsMasterResult, buildInfoResult *internal.BuildInfoResult) *desc.Server {
	d := &desc.Server{
		Endpoint: endpoint,

		CanonicalEndpoint: desc.Endpoint(isMasterResult.Me),
		ElectionID:        isMasterResult.ElectionID,
		LastUpdateTime:    time.Now().UTC(),
		LastWriteTime:     isMasterResult.LastWriteTimestamp,
		MaxBatchCount:     isMasterResult.MaxWriteBatchSize,
		MaxDocumentSize:   isMasterResult.MaxBSONObjectSize,
		MaxMessageSize:    isMasterResult.MaxMessageSizeBytes,
		SetName:           isMasterResult.SetName,
		SetVersion:        isMasterResult.SetVersion,
		Tags:              nil, // TODO: get tags
		WireVersion: desc.Range{
			Min: isMasterResult.MinWireVersion,
			Max: isMasterResult.MaxWireVersion,
		},
		Version: desc.NewVersionWithDesc(buildInfoResult.Version, buildInfoResult.VersionArray...),
	}

	if d.CanonicalEndpoint == "" {
		d.CanonicalEndpoint = endpoint
	}

	if !isMasterResult.OK {
		d.LastError = fmt.Errorf("not ok")
		return d
	}

	for _, host := range isMasterResult.Hosts {
		d.Members = append(d.Members, desc.Endpoint(host).Canonicalize())
	}

	for _, passive := range isMasterResult.Passives {
		d.Members = append(d.Members, desc.Endpoint(passive).Canonicalize())
	}

	for _, arbiter := range isMasterResult.Arbiters {
		d.Members = append(d.Members, desc.Endpoint(arbiter).Canonicalize())
	}

	d.Type = desc.Standalone

	if isMasterResult.IsReplicaSet {
		d.Type = desc.RSGhost
	} else if isMasterResult.SetName != "" {
		if isMasterResult.IsMaster {
			d.Type = desc.RSPrimary
		} else if isMasterResult.Hidden {
			d.Type = desc.RSMember
		} else if isMasterResult.Secondary {
			d.Type = desc.RSSecondary
		} else if isMasterResult.ArbiterOnly {
			d.Type = desc.RSArbiter
		} else {
			d.Type = desc.RSMember
		}
	} else if isMasterResult.Msg == "isdbgrid" {
		d.Type = desc.Mongos
	}

	return d
}
