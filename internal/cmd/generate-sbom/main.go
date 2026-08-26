// Copyright (C) MongoDB, Inc. 2026-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// Command generate-sbom generates a CycloneDX SBOM for the mongo-go-driver
// module and writes it to sbom.json. It aggregates the modules required by
// packages in the driver library, excluding examples, tests, and test
// packages, and injects libmongocrypt as an optional component since it is
// not a Go module dependency.
//
// Set EXPECT_ERROR=1 to fail instead of writing changes when the generated
// SBOM differs from the committed sbom.json — used in CI to verify that
// sbom.json is up to date.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	cdx "github.com/CycloneDX/cyclonedx-go"
	"github.com/CycloneDX/cyclonedx-gomod/pkg/generate/mod"
	"github.com/CycloneDX/cyclonedx-gomod/pkg/licensedetect/local"
	"github.com/rs/zerolog"
)

const serialNumber = "urn:uuid:b7adcdf8-bafc-43c5-a529-a73130697171"

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "generate-sbom:", err)
		os.Exit(1)
	}
}

func run() error {
	moduleDir := "."
	if len(os.Args) > 1 {
		moduleDir = os.Args[1]
	}

	// Analyze only the driver module, not this tool's workspace module —
	// otherwise cyclonedx-gomod (a dependency of this module) would be pulled
	// into the generated SBOM.
	if err := os.Setenv("GOWORK", "off"); err != nil {
		return err
	}

	version, err := libmongocryptVersion(filepath.Join(moduleDir, "etc", "install-libmongocrypt.sh"))
	if err != nil {
		return err
	}

	bom, err := generateBOM(moduleDir)
	if err != nil {
		return err
	}

	goVersion, err := goDirectiveVersion(filepath.Join(moduleDir, "go.mod"))
	if err != nil {
		return err
	}

	bom.SerialNumber = serialNumber
	setToolsMetadata(bom)
	pinStdlibVersion(bom, "go"+goVersion)
	injectLibmongocrypt(bom, version)

	sbomPath := filepath.Join(moduleDir, "sbom.json")
	return reconcile(sbomPath, bom)
}

func generateBOM(moduleDir string) (*cdx.BOM, error) {
	gen, err := mod.NewGenerator(moduleDir,
		mod.WithComponentType(cdx.ComponentTypeLibrary),
		mod.WithIncludeStdlib(true),
		mod.WithShortPURLS(true), // strips ?type=module query strings; see CycloneDX/cyclonedx-gomod#662.
		mod.WithLicenseDetector(local.NewDetector(zerolog.Nop(), local.DefaultMinDetectionConfidence)),
		mod.WithLogger(zerolog.Nop()),
	)
	if err != nil {
		return nil, fmt.Errorf("creating SBOM generator: %w", err)
	}

	bom, err := gen.Generate()
	if err != nil {
		return nil, fmt.Errorf("generating SBOM: %w", err)
	}
	return bom, nil
}

// cyclonedxGomodVersion is the pinned cyclonedx-gomod release this tool's
// metadata handling is modeled after (see setToolsMetadata). Bump it
// alongside the dependency in go.mod.
//
// Detected licenses are deliberately left as evidence rather than asserted
// into the declared licenses field (which cyclonedx-gomod's CLI would do
// behind its -assert-licenses flag): silkbomb's `update --select-licenses`
// step categorizes license evidence against MongoDB's Inbound Open Source
// Policy downstream, and needs the evidence field intact to do so.
const cyclonedxGomodVersion = "v1.12.0"

// setToolsMetadata records cyclonedx-gomod as the tool that generated the
// BOM's components, mirroring what its CLI records for itself. We omit the
// file hashes the CLI includes for its own binary — hashing this program's
// binary wouldn't identify cyclonedx-gomod, which is used here as a library.
func setToolsMetadata(bom *cdx.BOM) {
	if bom.Metadata == nil {
		bom.Metadata = &cdx.Metadata{}
	}
	bom.Metadata.Tools = &cdx.ToolsChoice{
		Tools: &[]cdx.Tool{ //nolint:staticcheck
			{
				Vendor:  "CycloneDX",
				Name:    "cyclonedx-gomod",
				Version: cyclonedxGomodVersion,
				ExternalReferences: &[]cdx.ExternalReference{
					{Type: cdx.ERTypeVCS, URL: "https://github.com/CycloneDX/cyclonedx-gomod"},
					{Type: cdx.ERTypeWebsite, URL: "https://cyclonedx.org"},
				},
			},
		},
	}
}

var goDirectiveRE = regexp.MustCompile(`(?m)^go (\S+)`)

// goDirectiveVersion extracts the "go" directive from the module's go.mod.
func goDirectiveVersion(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", path, err)
	}
	m := goDirectiveRE.FindSubmatch(data)
	if m == nil || len(m[1]) == 0 {
		return "", fmt.Errorf("could not find go directive in %s", path)
	}
	return string(m[1]), nil
}

// pinStdlibVersion overrides the stdlib pseudo-component's version (and the
// bom-ref/purl derived from it) with goVersion. cyclonedx-gomod otherwise
// records the exact patch version of whichever Go toolchain happened to run
// it (e.g. "go1.26.3"), which makes the SBOM depend on the generating
// machine's installed toolchain rather than the driver's own go.mod — so a
// developer and CI running different Go patch versions would each consider
// the other's sbom.json out of date. The go.mod "go" directive is the
// project's actual, committed minimum version and is stable across machines.
func pinStdlibVersion(bom *cdx.BOM, goVersion string) {
	if bom.Components == nil {
		return
	}
	components := *bom.Components
	for i := range components {
		c := &components[i]
		if c.Name != "std" || c.Group != "" {
			continue
		}
		oldRef := c.BOMRef
		newRef := "pkg:golang/std@" + goVersion + "?type=module"

		c.Version = goVersion
		c.BOMRef = newRef
		c.PackageURL = "pkg:golang/std@" + goVersion

		if bom.Dependencies != nil {
			deps := *bom.Dependencies
			for j := range deps {
				if deps[j].Ref == oldRef {
					deps[j].Ref = newRef
				}
				if deps[j].Dependencies == nil {
					continue
				}
				dependsOn := *deps[j].Dependencies
				for k, ref := range dependsOn {
					if ref == oldRef {
						dependsOn[k] = newRef
					}
				}
			}
		}
	}
}

var libmongocryptTagRE = regexp.MustCompile(`(?m)^\s*LIBMONGOCRYPT_TAG="([^"]+)"`)

// libmongocryptVersion extracts the pinned libmongocrypt version from the
// install script, the single source of truth for the bundled version.
func libmongocryptVersion(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", path, err)
	}
	m := libmongocryptTagRE.FindSubmatch(data)
	if m == nil || len(m[1]) == 0 {
		return "", fmt.Errorf("could not find LIBMONGOCRYPT_TAG in %s", path)
	}
	return string(m[1]), nil
}

// injectLibmongocrypt adds libmongocrypt as an optional component and wires it
// in as a dependency of the main module component.
func injectLibmongocrypt(bom *cdx.BOM, version string) {
	purl := "pkg:github/mongodb/libmongocrypt@" + version

	component := cdx.Component{
		Type:        cdx.ComponentTypeLibrary,
		BOMRef:      purl,
		Supplier:    &cdx.OrganizationalEntity{Name: "MongoDB, Inc.", URL: &[]string{"https://mongodb.com"}},
		Author:      "MongoDB, Inc.",
		Group:       "mongodb",
		Name:        "libmongocrypt",
		Version:     version,
		Description: "Required C library for Client Side and Queryable Encryption in MongoDB",
		Scope:       cdx.ScopeOptional,
		Licenses:    &cdx.Licenses{{License: &cdx.License{ID: "Apache-2.0"}}},
		Copyright:   "Copyright 2019-present MongoDB, Inc.",
		CPE:         "cpe:2.3:a:mongodb:libmongocrypt:" + version + ":*:*:*:*:*:*:*",
		PackageURL:  purl,
		ExternalReferences: &[]cdx.ExternalReference{
			{URL: "https://github.com/mongodb/libmongocrypt.git", Type: cdx.ERTypeDistribution},
		},
	}

	components := derefComponents(bom.Components)
	components = append(components, component)
	bom.Components = &components

	deps := derefDependencies(bom.Dependencies)
	if bom.Metadata != nil && bom.Metadata.Component != nil {
		mainRef := bom.Metadata.Component.BOMRef
		for i := range deps {
			if deps[i].Ref != mainRef {
				continue
			}
			dependsOn := derefStrings(deps[i].Dependencies)
			dependsOn = append(dependsOn, purl)
			deps[i].Dependencies = &dependsOn
		}
	}
	deps = append(deps, cdx.Dependency{Ref: purl, Dependencies: &[]string{}})
	bom.Dependencies = &deps
}

// reconcile compares the newly generated BOM against the existing sbom.json
// (ignoring version and timestamp) and, depending on the outcome:
//   - unchanged: touches sbom.json (so Taskfile's timestamp method does not
//     re-run this generator on the next invocation) without rewriting it.
//   - changed, EXPECT_ERROR=1: leaves sbom.json untouched and returns an
//     error — used in CI to verify sbom.json is up to date.
//   - changed, otherwise: writes sbom.json with its version incremented.
func reconcile(sbomPath string, bom *cdx.BOM) error {
	newContent, err := normalizedBOM(bom)
	if err != nil {
		return err
	}

	currentVersion := 0
	oldContent := ""
	if data, err := os.ReadFile(sbomPath); err == nil {
		var old struct {
			Version int `json:"version"`
		}
		if err := json.Unmarshal(data, &old); err != nil {
			return fmt.Errorf("parsing %s: %w", sbomPath, err)
		}
		currentVersion = old.Version
		oldContent, err = normalized(data)
		if err != nil {
			return fmt.Errorf("parsing %s: %w", sbomPath, err)
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	if newContent == oldContent {
		now := time.Now()
		return os.Chtimes(sbomPath, now, now)
	}

	if os.Getenv("EXPECT_ERROR") == "1" {
		return fmt.Errorf("%s is out of date. Run 'task generate-sbom' and commit the result", sbomPath)
	}

	bom.Version = currentVersion + 1
	return writeBOM(sbomPath, bom)
}

func writeBOM(path string, bom *cdx.BOM) error {
	var buf bytes.Buffer
	enc := cdx.NewBOMEncoder(&buf, cdx.BOMFileFormatJSON)
	enc.SetPretty(true)
	if err := enc.EncodeVersion(bom, cdx.SpecVersion1_5); err != nil {
		return fmt.Errorf("encoding SBOM: %w", err)
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}

// normalizedBOM returns a stable JSON encoding of bom with its version and
// timestamp removed, for content comparison.
func normalizedBOM(bom *cdx.BOM) (string, error) {
	var buf bytes.Buffer
	enc := cdx.NewBOMEncoder(&buf, cdx.BOMFileFormatJSON)
	if err := enc.EncodeVersion(bom, cdx.SpecVersion1_5); err != nil {
		return "", fmt.Errorf("encoding SBOM: %w", err)
	}
	return normalized(buf.Bytes())
}

// normalized returns a stable JSON encoding of data with its version and
// timestamp removed, for content comparison.
func normalized(data []byte) (string, error) {
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return "", err
	}
	delete(m, "version")
	if md, ok := m["metadata"].(map[string]any); ok {
		delete(md, "timestamp")
	}
	out, err := json.Marshal(m)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func derefComponents(c *[]cdx.Component) []cdx.Component {
	if c == nil {
		return nil
	}
	return *c
}

func derefDependencies(d *[]cdx.Dependency) []cdx.Dependency {
	if d == nil {
		return nil
	}
	return *d
}

func derefStrings(s *[]string) []string {
	if s == nil {
		return nil
	}
	return *s
}
