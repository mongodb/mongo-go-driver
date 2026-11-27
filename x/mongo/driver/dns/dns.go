// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// Package dns is intended for internal use only. It is made available to
// facilitate use cases that require access to internal MongoDB driver
// functionality and state. The API of this package is not stable and there is
// no backward compatibility guarantee.
//
// WARNING: THIS PACKAGE IS EXPERIMENTAL AND MAY BE MODIFIED OR REMOVED WITHOUT
// NOTICE! USE WITH EXTREME CAUTION!
package dns

import (
	"errors"
	"fmt"
	"net"
	"runtime"
	"strings"

	"golang.org/x/net/idna"
	"golang.org/x/net/publicsuffix"
)

// Resolver resolves DNS records.
type Resolver struct {
	// Holds the functions to use for DNS lookups
	LookupSRV func(string, string, string) (string, []*net.SRV, error)
	LookupTXT func(string) ([]string, error)
}

// DefaultResolver is a Resolver that uses the default Resolver from the net package.
var DefaultResolver = &Resolver{net.LookupSRV, net.LookupTXT}

// ParseHostsOptions holds the options for ParseHostsWithOptions.
type ParseHostsOptions struct {
	// SRVServiceName is the SRV service name to look up. Defaults to "mongodb" if empty.
	SRVServiceName string

	// AllowedHostsSuffix, if non-empty, is the normalized (leading ".", lowercase, A-label)
	// domain suffix that returned SRV hostnames are validated against, in place of the domain
	// derived from host. Use NormalizeAllowedHostsSuffix to produce this value.
	AllowedHostsSuffix string

	// StopOnErr causes ParseHostsWithOptions to return an error as soon as one is encountered
	// validating a returned hostname, rather than dropping the offending hostname and continuing.
	StopOnErr bool
}

// ParseHosts uses the srv string and service name to get the hosts.
func (r *Resolver) ParseHosts(host string, srvName string, stopOnErr bool) ([]string, error) {
	return r.ParseHostsWithOptions(host, ParseHostsOptions{
		SRVServiceName: srvName,
		StopOnErr:      stopOnErr,
	})
}

// ParseHostsWithOptions uses the srv string and the provided options to get the hosts.
func (r *Resolver) ParseHostsWithOptions(host string, opts ParseHostsOptions) ([]string, error) {
	parsedHosts := strings.Split(host, ",")

	if len(parsedHosts) != 1 {
		return nil, fmt.Errorf("URI with SRV must include one and only one hostname")
	}
	return r.fetchSeedlistFromSRV(parsedHosts[0], opts)
}

// GetConnectionArgsFromTXT gets the TXT record associated with the host and returns the connection arguments.
func (r *Resolver) GetConnectionArgsFromTXT(host string) ([]string, error) {
	var connectionArgsFromTXT []string

	// error ignored because not finding a TXT record should not be
	// considered an error.
	recordsFromTXT, _ := r.LookupTXT(host)

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

func (r *Resolver) fetchSeedlistFromSRV(host string, opts ParseHostsOptions) ([]string, error) {
	var err error

	_, _, err = net.SplitHostPort(host)

	if err == nil {
		// we were able to successfully extract a port from the host,
		// but should not be able to when using SRV
		return nil, fmt.Errorf("URI with srv must not include a port number")
	}

	// default to "mongodb" as service name if not supplied
	srvName := opts.SRVServiceName
	if srvName == "" {
		srvName = "mongodb"
	}
	_, addresses, err := r.LookupSRV(srvName, "tcp", host)
	if err != nil && strings.Contains(err.Error(), "cannot unmarshal DNS message") {
		return nil, fmt.Errorf("see https://pkg.go.dev/go.mongodb.org/mongo-driver/mongo#hdr-Potential_DNS_Issues: %w", err)
	} else if err != nil {
		return nil, err
	}

	parsedHosts := make([]string, 0, len(addresses))
	for _, address := range addresses {
		normalizedTarget, normErr := normalizeHostname(address.Target)
		if normErr != nil {
			if opts.StopOnErr {
				return nil, fmt.Errorf("invalid SRV record target %q: %w", address.Target, normErr)
			}
			continue
		}
		if err = validateSRVResult(normalizedTarget, host, opts.AllowedHostsSuffix); err != nil {
			if opts.StopOnErr {
				return nil, err
			}
			continue
		}
		parsedHosts = append(parsedHosts, fmt.Sprintf("%s:%d", normalizedTarget, address.Port))
	}
	return parsedHosts, nil
}

func validateSRVResult(recordFromSRV, inputHostName, allowedHostsSuffix string) error {
	normalizedRecord, err := normalizeHostname(recordFromSRV)
	if err != nil {
		return err
	}

	// The {domainname} that returned hostnames are validated against must have the same
	// normalizations applied, so that neither trailing dots, case, nor Unicode/Punycode encoding
	// can affect the comparison.
	normalizedHost, err := normalizeHostname(inputHostName)
	if err != nil {
		return err
	}

	// When srvAllowedHostsSuffix is configured, it replaces the domain derived from the
	// connection string hostname.
	if allowedHostsSuffix != "" {
		if !strings.HasSuffix(normalizedRecord, allowedHostsSuffix) {
			return errors.New("domain suffix from SRV record not matched input domain")
		}
		return nil
	}

	separatedInputDomain := strings.Split(normalizedHost, ".")
	separatedRecord := strings.Split(normalizedRecord, ".")
	if l := len(separatedInputDomain); l < 3 && len(separatedRecord) <= l {
		return fmt.Errorf("server record (%d levels) should have more domain levels than parent URI (%d levels)", len(separatedRecord), l)
	}
	if len(separatedRecord) < len(separatedInputDomain) {
		return errors.New("domain suffix from SRV record not matched input domain")
	}

	inputDomainSuffix := separatedInputDomain
	if len(inputDomainSuffix) > 2 {
		inputDomainSuffix = inputDomainSuffix[1:]
	}
	domainSuffixOffset := len(separatedRecord) - len(inputDomainSuffix)

	recordDomainSuffix := separatedRecord[domainSuffixOffset:]
	for ix, label := range inputDomainSuffix {
		if label != recordDomainSuffix[ix] {
			return errors.New("domain suffix from SRV record not matched input domain")
		}
	}
	return nil
}

func normalizeHostname(hostname string) (string, error) {
	aLabel, err := idna.ToASCII(strings.TrimSuffix(hostname, "."))
	if err != nil {
		return "", fmt.Errorf("invalid hostname %q: %w", hostname, err)
	}
	return strings.ToLower(aLabel), nil
}

// NormalizeAllowedHostsSuffix normalizes and validates a srvAllowedHostsSuffix URI option value.
// The returned value has a leading ".".
func NormalizeAllowedHostsSuffix(value string) (string, error) {
	// 1. Any leading or trailing "." MUST be stripped.
	trimmed := strings.Trim(value, ".")
	if trimmed == "" {
		return "", errors.New("srvAllowedHostsSuffix must not be empty")
	}

	// 2. Convert to A-label (Punycode) form. 3. Normalize to lowercase using ASCII case folding.
	normalized, err := normalizeHostname(trimmed)
	if err != nil {
		return "", fmt.Errorf("invalid srvAllowedHostsSuffix: %w", err)
	}

	_, isReserved := map[string]struct{}{
		// RFC 6761 special use names.
		"test":      {},
		"localhost": {},
		"invalid":   {},
		"example":   {},
		// RFC 6762 multicast DNS.
		"local": {},
		// Reserved by ICANN for private use.
		"internal": {},
		// Not officially reserved by ICANN but commonly used privately.
		"corp": {},
		"home": {},
		"mail": {},
	}[normalized]

	// 4. The resulting value MUST contain at least two "." separated labels, unless it is a
	// single label that appears in the list of names reserved for private or special use.
	if !strings.Contains(normalized, ".") && !isReserved {
		return "", fmt.Errorf("srvAllowedHostsSuffix %q must contain at least two domain labels", value)
	}

	// 5. Drivers SHOULD raise an error if the resulting value is a public suffix, unless it is
	// one of the reserved names above.
	if !isReserved && isPublicSuffix(normalized) {
		return "", fmt.Errorf("srvAllowedHostsSuffix %q must not be a public suffix", value)
	}

	// 6. A "." MUST be prepended.
	return "." + normalized, nil
}

func isPublicSuffix(domain string) bool {
	suffix, _ := publicsuffix.PublicSuffix(domain)
	return suffix == domain
}

var allowedTXTOptions = map[string]struct{}{
	"authsource":   {},
	"replicaset":   {},
	"loadbalanced": {},
}

func validateTXTResult(paramsFromTXT []string) error {
	for _, param := range paramsFromTXT {
		kv := strings.SplitN(param, "=", 2)
		if len(kv) != 2 {
			return errors.New("invalid TXT record")
		}
		key := strings.ToLower(kv[0])
		if _, ok := allowedTXTOptions[key]; !ok {
			return fmt.Errorf("cannot specify option '%s' in TXT record", kv[0])
		}
	}
	return nil
}
