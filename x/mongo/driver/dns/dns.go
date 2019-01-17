// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package dns

import (
	"context"
	"errors"
	"fmt"
	"net"
	"runtime"
	"strings"
)

type lookup interface {
	LookupSRV(ctx context.Context, service, proto, name string) (string, []*net.SRV, error)
	LookupTXT(ctx context.Context, name string) ([]string, error)
}

// Resolver resolves DNS records.
type Resolver struct {
	// Holds the resolver to use for DNS lookups
	Lookup lookup

	// Value for rescanSRVFrequencyMS to pass in when testing. Normally SRV records
	// should not be polled more than once every 60 seconds.
	//TestingRescanFrequencyMS time.Duration
}

// DefaultResolver returns a Resolver that uses the default Resolver from the net package.
func DefaultResolver() Resolver {
	return Resolver{net.DefaultResolver}
}

// ResolveHostFromSrvRecords uses the srv string to get the hosts.
func (r *Resolver) ResolveHostFromSrvRecords(host string) ([]string, error) {
	parsedHosts := strings.Split(host, ",")

	if len(parsedHosts) != 1 {
		return nil, fmt.Errorf("URI with SRV must include one and only one hostname")
	}
	parsedHosts, err := r.fetchSeedlistFromSRV(parsedHosts[0])
	if err != nil {
		return nil, err
	}
	return parsedHosts, nil
}

// ResolveAdditionalQueryParametersFromTxtRecords gets the TXT record associated with the host
// and returns the connection arguments.
func (r *Resolver) ResolveAdditionalQueryParametersFromTxtRecords(host string) ([]string, error) {
	var connectionArgsFromTXT []string

	// error ignored because not finding a TXT record should not be
	// considered an error.
	recordsFromTXT, _ := r.Lookup.LookupTXT(context.Background(), host)

	// This is a temporary fix to get around bug https://github.com/golang/go/issues/21472.
	// It will currently incorrectly concatenate multiple TXT records to one
	// on windows.
	if runtime.GOOS == "windows" {
		recordsFromTXT = []string{strings.Join(recordsFromTXT, "")}
	}

	if len(recordsFromTXT) > 1 {
		return nil, errors.New("multiple records from TXT not supported")
	}
	if len(recordsFromTXT) > 0 {
		connectionArgsFromTXT = strings.FieldsFunc(recordsFromTXT[0], func(r rune) bool { return r == ';' || r == '&' })

		err := validateTXTResult(connectionArgsFromTXT)
		if err != nil {
			return nil, err
		}
	}

	return connectionArgsFromTXT, nil
}

func (r *Resolver) fetchSeedlistFromSRV(host string) ([]string, error) {
	var err error

	_, _, err = net.SplitHostPort(host)

	if err == nil {
		// we were able to successfully extract a port from the host,
		// but should not be able to when using SRV
		return nil, fmt.Errorf("URI with srv must not include a port number")
	}

	_, addresses, err := r.Lookup.LookupSRV(context.Background(), "mongodb", "tcp", host)
	if err != nil {
		return nil, err
	}

	parsedHosts := make([]string, len(addresses))
	for i, address := range addresses {
		trimmedAddressTarget := strings.TrimSuffix(address.Target, ".")
		err := validateSRVResult(trimmedAddressTarget, host)
		if err != nil {
			return nil, err
		}
		parsedHosts[i] = fmt.Sprintf("%s:%d", trimmedAddressTarget, address.Port)
	}
	return parsedHosts, nil
}

func validateSRVResult(recordFromSRV, inputHostName string) error {
	separatedInputDomain := strings.Split(inputHostName, ".")
	separatedRecord := strings.Split(recordFromSRV, ".")
	if len(separatedRecord) < 2 {
		return errors.New("DNS name must contain at least 2 labels")
	}
	if len(separatedRecord) < len(separatedInputDomain) {
		return errors.New("Domain suffix from SRV record not matched input domain")
	}

	inputDomainSuffix := separatedInputDomain[1:]
	domainSuffixOffset := len(separatedRecord) - (len(separatedInputDomain) - 1)

	recordDomainSuffix := separatedRecord[domainSuffixOffset:]
	for ix, label := range inputDomainSuffix {
		if label != recordDomainSuffix[ix] {
			return errors.New("Domain suffix from SRV record not matched input domain")
		}
	}
	return nil
}

var allowedTXTOptions = map[string]struct{}{
	"authsource": {},
	"replicaset": {},
}

func validateTXTResult(paramsFromTXT []string) error {
	for _, param := range paramsFromTXT {
		kv := strings.SplitN(param, "=", 2)
		if len(kv) != 2 {
			return errors.New("Invalid TXT record")
		}
		key := strings.ToLower(kv[0])
		if _, ok := allowedTXTOptions[key]; !ok {
			return fmt.Errorf("Cannot specify option '%s' in TXT record", kv[0])
		}
	}
	return nil
}
