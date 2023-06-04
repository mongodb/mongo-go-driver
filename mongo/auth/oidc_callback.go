// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// Package auth provides structured used in authentication.
package auth

import (
	"context"
)

// IDPServerInfo contains the information used by callbacks to authenticate with the Identity Provider.
type IDPServerInfo struct {
	Issuer        string
	ClientID      string
	RequestScopes []string
}

// IDPRefreshInfo contains the refresh callback token.
type IDPRefreshInfo struct {
	RefreshToken *string
}

// IDPServerResp is the response from the ID provider.
type IDPServerResp struct {
	AccessToken      string
	ExpiresInSeconds *int
	RefreshToken     *string
}

// OidcOnRequest defines the request callback.
type OidcOnRequest func(ctx context.Context, username string, ServerInfo IDPServerInfo) (IDPServerResp, error)

// OidcOnRefresh defines the refresh callback.
type OidcOnRefresh func(ctx context.Context, username string, ServerInfo IDPServerInfo, refreshInfo IDPRefreshInfo) (IDPServerResp, error)
