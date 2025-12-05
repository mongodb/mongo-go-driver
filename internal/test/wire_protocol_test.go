// Copyright (C) MongoDB, Inc. 2025-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package test

import (
	"context"
	"testing"

	"go.mongodb.org/mongo-driver/v2/internal/require"
	"go.mongodb.org/mongo-driver/v2/internal/test/container"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/description"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/topology"
)

func TestMinimumSupportedWireVersion(t *testing.T) {
	tests := []struct {
		name    string
		image   string
		setup   func()
		wantErr bool
	}{
		{
			name:    "unsupported mongo version 4.0",
			image:   "mongo:4.0",
			setup:   nil,
			wantErr: true,
		},
		{
			name:  "unsupported mongo version 4.0 with topology override",
			image: "mongo:4.0",
			setup: func() {
				topology.MinSupportedMongoDBVersion = "4.0"
				topology.SupportedWireVersions = description.VersionRange{Min: 7, Max: 25}
			},
			wantErr: false,
		},
		{
			name:    "supported mongo version 5.0",
			image:   "mongo:5.0",
			setup:   nil,
			wantErr: false,
		},
		{
			name:    "supported mongo version 6.0",
			image:   "mongo:6.0",
			setup:   nil,
			wantErr: false,
		},
		{
			name:    "supported mongo version 7.0",
			image:   "mongo:7.0",
			setup:   nil,
			wantErr: false,
		},
		{
			name:    "supported mongo version 8.0",
			image:   "mongo:8.0",
			setup:   nil,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			client, teardown := container.NewMongo(t, context.Background(), container.WithMongoImage(tt.image))
			defer teardown(t)

			err := client.Ping(context.Background(), nil)
			require.Equal(t, tt.wantErr, err != nil, "Ping() error = %v", err)
		})
	}
}
