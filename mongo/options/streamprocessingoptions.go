// Copyright (C) MongoDB, Inc. 2026-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

import "go.mongodb.org/mongo-driver/v2/bson"

// CreateStreamProcessorOptions represents arguments that can be used to
// configure a CreateStreamProcessor operation.
//
// See corresponding setter methods for documentation.
type CreateStreamProcessorOptions struct {
	DLQ                 bson.Raw
	StreamMetaFieldName *string
	Tier                *string
	FailoverEnabled     *bool
}

// CreateStreamProcessorOptionsBuilder configures the createStreamProcessor command.
type CreateStreamProcessorOptionsBuilder struct {
	Opts []func(*CreateStreamProcessorOptions) error
}

// CreateStreamProcessor creates a new CreateStreamProcessorOptionsBuilder.
func CreateStreamProcessor() *CreateStreamProcessorOptionsBuilder {
	return &CreateStreamProcessorOptionsBuilder{}
}

// List returns a list of CreateStreamProcessorOptions setter functions.
func (c *CreateStreamProcessorOptionsBuilder) List() []func(*CreateStreamProcessorOptions) error {
	return c.Opts
}

// SetDLQ sets the dead-letter-queue configuration document.
func (c *CreateStreamProcessorOptionsBuilder) SetDLQ(dlq bson.Raw) *CreateStreamProcessorOptionsBuilder {
	c.Opts = append(c.Opts, func(o *CreateStreamProcessorOptions) error {
		o.DLQ = dlq
		return nil
	})
	return c
}

// SetStreamMetaFieldName sets the field name to use for stream metadata.
func (c *CreateStreamProcessorOptionsBuilder) SetStreamMetaFieldName(name string) *CreateStreamProcessorOptionsBuilder {
	c.Opts = append(c.Opts, func(o *CreateStreamProcessorOptions) error {
		o.StreamMetaFieldName = &name
		return nil
	})
	return c
}

// SetTier sets the compute tier for the new processor.
func (c *CreateStreamProcessorOptionsBuilder) SetTier(tier string) *CreateStreamProcessorOptionsBuilder {
	c.Opts = append(c.Opts, func(o *CreateStreamProcessorOptions) error {
		o.Tier = &tier
		return nil
	})
	return c
}

// SetFailoverEnabled sets whether failover should be enabled.
func (c *CreateStreamProcessorOptionsBuilder) SetFailoverEnabled(b bool) *CreateStreamProcessorOptionsBuilder {
	c.Opts = append(c.Opts, func(o *CreateStreamProcessorOptions) error {
		o.FailoverEnabled = &b
		return nil
	})
	return c
}

// StreamProcessorFailoverOptions describes a failover request attached to a
// startStreamProcessor command.
type StreamProcessorFailoverOptions struct {
	Region string  // required when failover is sent
	Mode   *string // "GRACEFUL" (default) or "FORCED"
	DryRun *bool
}

// StreamProcessorFailoverOptionsBuilder builds a StreamProcessorFailoverOptions value.
type StreamProcessorFailoverOptionsBuilder struct {
	Opts []func(*StreamProcessorFailoverOptions) error
}

// StreamProcessorFailover creates a new StreamProcessorFailoverOptionsBuilder.
func StreamProcessorFailover() *StreamProcessorFailoverOptionsBuilder {
	return &StreamProcessorFailoverOptionsBuilder{}
}

// List returns a list of StreamProcessorFailoverOptions setter functions.
func (b *StreamProcessorFailoverOptionsBuilder) List() []func(*StreamProcessorFailoverOptions) error {
	return b.Opts
}

// SetRegion sets the target region for failover. Required when failover is sent.
func (b *StreamProcessorFailoverOptionsBuilder) SetRegion(region string) *StreamProcessorFailoverOptionsBuilder {
	b.Opts = append(b.Opts, func(o *StreamProcessorFailoverOptions) error {
		o.Region = region
		return nil
	})
	return b
}

// SetMode sets the failover mode. Valid values: "GRACEFUL" (default), "FORCED".
func (b *StreamProcessorFailoverOptionsBuilder) SetMode(mode string) *StreamProcessorFailoverOptionsBuilder {
	b.Opts = append(b.Opts, func(o *StreamProcessorFailoverOptions) error {
		o.Mode = &mode
		return nil
	})
	return b
}

// SetDryRun configures whether the failover request should be validated
// without being executed.
func (b *StreamProcessorFailoverOptionsBuilder) SetDryRun(v bool) *StreamProcessorFailoverOptionsBuilder {
	b.Opts = append(b.Opts, func(o *StreamProcessorFailoverOptions) error {
		o.DryRun = &v
		return nil
	})
	return b
}

// StartStreamProcessorOptions represents arguments that can be used to
// configure a StartStreamProcessor operation.
//
// The spec reserves a StartAfter field for future use; drivers MUST NOT send
// it to the wire today. It is intentionally absent from this struct.
type StartStreamProcessorOptions struct {
	Workers              *int32
	ClearCheckpoints     *bool
	StartAtOperationTime *bson.Timestamp
	Tier                 *string
	EnableAutoScaling    *bool
	Failover             *StreamProcessorFailoverOptions
}

// StartStreamProcessorOptionsBuilder configures the startStreamProcessor command.
type StartStreamProcessorOptionsBuilder struct {
	Opts []func(*StartStreamProcessorOptions) error
}

// StartStreamProcessor creates a new StartStreamProcessorOptionsBuilder.
func StartStreamProcessor() *StartStreamProcessorOptionsBuilder {
	return &StartStreamProcessorOptionsBuilder{}
}

// List returns a list of StartStreamProcessorOptions setter functions.
func (s *StartStreamProcessorOptionsBuilder) List() []func(*StartStreamProcessorOptions) error {
	return s.Opts
}

// SetWorkers sets the workers field on the command.
func (s *StartStreamProcessorOptionsBuilder) SetWorkers(n int32) *StartStreamProcessorOptionsBuilder {
	s.Opts = append(s.Opts, func(o *StartStreamProcessorOptions) error {
		o.Workers = &n
		return nil
	})
	return s
}

// SetClearCheckpoints sets the options.clearCheckpoints flag.
func (s *StartStreamProcessorOptionsBuilder) SetClearCheckpoints(b bool) *StartStreamProcessorOptionsBuilder {
	s.Opts = append(s.Opts, func(o *StartStreamProcessorOptions) error {
		o.ClearCheckpoints = &b
		return nil
	})
	return s
}

// SetStartAtOperationTime sets the options.startAtOperationTime BSON timestamp.
func (s *StartStreamProcessorOptionsBuilder) SetStartAtOperationTime(ts bson.Timestamp) *StartStreamProcessorOptionsBuilder {
	s.Opts = append(s.Opts, func(o *StartStreamProcessorOptions) error {
		o.StartAtOperationTime = &ts
		return nil
	})
	return s
}

// SetTier sets the options.tier value. Valid values: "SP2", "SP5", "SP10",
// "SP30", "SP50".
func (s *StartStreamProcessorOptionsBuilder) SetTier(tier string) *StartStreamProcessorOptionsBuilder {
	s.Opts = append(s.Opts, func(o *StartStreamProcessorOptions) error {
		o.Tier = &tier
		return nil
	})
	return s
}

// SetEnableAutoScaling sets the options.enableAutoScaling flag.
func (s *StartStreamProcessorOptionsBuilder) SetEnableAutoScaling(b bool) *StartStreamProcessorOptionsBuilder {
	s.Opts = append(s.Opts, func(o *StartStreamProcessorOptions) error {
		o.EnableAutoScaling = &b
		return nil
	})
	return s
}

// SetFailover attaches a failover sub-document to the command.
func (s *StartStreamProcessorOptionsBuilder) SetFailover(f *StreamProcessorFailoverOptions) *StartStreamProcessorOptionsBuilder {
	s.Opts = append(s.Opts, func(o *StartStreamProcessorOptions) error {
		o.Failover = f
		return nil
	})
	return s
}

// GetStreamProcessorStatsOptions represents arguments that can be used to
// configure a GetStreamProcessorStats operation.
type GetStreamProcessorStatsOptions struct {
	Verbose *bool
}

// GetStreamProcessorStatsOptionsBuilder configures the getStreamProcessorStats command.
type GetStreamProcessorStatsOptionsBuilder struct {
	Opts []func(*GetStreamProcessorStatsOptions) error
}

// GetStreamProcessorStats creates a new GetStreamProcessorStatsOptionsBuilder.
func GetStreamProcessorStats() *GetStreamProcessorStatsOptionsBuilder {
	return &GetStreamProcessorStatsOptionsBuilder{}
}

// List returns a list of GetStreamProcessorStatsOptions setter functions.
func (g *GetStreamProcessorStatsOptionsBuilder) List() []func(*GetStreamProcessorStatsOptions) error {
	return g.Opts
}

// SetVerbose sets the options.verbose flag.
func (g *GetStreamProcessorStatsOptionsBuilder) SetVerbose(b bool) *GetStreamProcessorStatsOptionsBuilder {
	g.Opts = append(g.Opts, func(o *GetStreamProcessorStatsOptions) error {
		o.Verbose = &b
		return nil
	})
	return g
}

// GetStreamProcessorSamplesOptions represents arguments that can be used to
// configure a GetStreamProcessorSamples call.
//
// If CursorID is absent or zero, a new sample cursor is opened via
// startSampleStreamProcessor (and a Limit may be provided). If non-zero, the
// next batch is fetched via getMoreSampleStreamProcessor (and a BatchSize may
// be provided).
type GetStreamProcessorSamplesOptions struct {
	CursorID  *int64
	Limit     *int32
	BatchSize *int32
}

// GetStreamProcessorSamplesOptionsBuilder configures a GetStreamProcessorSamples call.
type GetStreamProcessorSamplesOptionsBuilder struct {
	Opts []func(*GetStreamProcessorSamplesOptions) error
}

// GetStreamProcessorSamples creates a new GetStreamProcessorSamplesOptionsBuilder.
func GetStreamProcessorSamples() *GetStreamProcessorSamplesOptionsBuilder {
	return &GetStreamProcessorSamplesOptionsBuilder{}
}

// List returns a list of GetStreamProcessorSamplesOptions setter functions.
func (g *GetStreamProcessorSamplesOptionsBuilder) List() []func(*GetStreamProcessorSamplesOptions) error {
	return g.Opts
}

// SetCursorID sets the cursor ID from a previous call. If absent or zero, a
// new sample cursor is opened.
func (g *GetStreamProcessorSamplesOptionsBuilder) SetCursorID(id int64) *GetStreamProcessorSamplesOptionsBuilder {
	g.Opts = append(g.Opts, func(o *GetStreamProcessorSamplesOptions) error {
		o.CursorID = &id
		return nil
	})
	return g
}

// SetLimit sets the maximum number of documents to sample. Only sent on the
// initial call (when CursorID is absent or zero); ignored on subsequent calls.
func (g *GetStreamProcessorSamplesOptionsBuilder) SetLimit(n int32) *GetStreamProcessorSamplesOptionsBuilder {
	g.Opts = append(g.Opts, func(o *GetStreamProcessorSamplesOptions) error {
		o.Limit = &n
		return nil
	})
	return g
}

// SetBatchSize sets the desired batch size. Only sent on subsequent calls
// (when CursorID is non-zero); ignored on the initial call.
func (g *GetStreamProcessorSamplesOptionsBuilder) SetBatchSize(n int32) *GetStreamProcessorSamplesOptionsBuilder {
	g.Opts = append(g.Opts, func(o *GetStreamProcessorSamplesOptions) error {
		o.BatchSize = &n
		return nil
	})
	return g
}
