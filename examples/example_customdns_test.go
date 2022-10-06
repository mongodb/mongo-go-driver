// Copyright (C) MongoDB, Inc. 2022-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package examples

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/miekg/dns"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func resolve(ctx context.Context, cache *dnsCache, in *dns.Conn, out *dns.Conn) {
	for ctx.Err() == nil {
		q, err := in.ReadMsg()
		if err != nil {
			// TODO: Handle error.
			continue
		}
		if len(q.Question) != 1 {
			// Multiple questions in a single query is not actually used in real life.
			continue
		}

		a, err := func() (*dns.Msg, error) {
			cache.lock.Lock()
			defer cache.lock.Unlock()

			now := time.Now()
			if rr, ok := cache.records[q.Question[0]]; ok && rr.exp.After(now) {
				a := new(dns.Msg)
				a.SetReply(q)
				a.Compress = false
				a.Answer = append(a.Answer, rr.record)
				return a, nil
			}

			err := out.WriteMsg(q)
			if err != nil {
				return nil, err
			}

			m, err := out.ReadMsg()
			if err != nil {
				return nil, err
			}

			l := len(m.Answer)
			for i, q := range m.Question {
				if i >= l {
					break
				}
				a := m.Answer[i]
				cache.records[q] = &RR{
					a,
					now.Add(time.Second * time.Duration(a.Header().Ttl)),
				}
			}
			return m, nil
		}()
		if err != nil {
			// TODO: Handle error.
			continue
		}

		if err := in.WriteMsg(a); err != nil {
			// TODO: Handle error.
			continue
		}
	}
}

type RR struct {
	record dns.RR
	exp    time.Time
}

type dnsCache struct {
	records map[dns.Question]*RR
	lock    *sync.Mutex
}

func (c *dnsCache) Purge() {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.records = make(map[dns.Question]*RR)
}

type Purgeable interface {
	Purge()
}

type conn struct {
	net.Conn
	cache Purgeable
}

func (c conn) Close() error {
	err := c.Conn.Close()
	c.cache.Purge()
	return err
}

type dialer struct {
	*net.Dialer
	cache *dnsCache
}

func NewDialer() dialer {
	cache := &dnsCache{
		records: make(map[dns.Question]*RR),
		lock:    &sync.Mutex{},
	}
	return dialer{
		Dialer: &net.Dialer{
			Resolver: &net.Resolver{
				PreferGo: true,
				Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
					outConn, err := net.Dial(network, address)
					conn, inConn := net.Pipe()
					if err == nil {
						go resolve(ctx, cache, &dns.Conn{Conn: inConn}, &dns.Conn{Conn: outConn})
					}
					return conn, err
				},
			},
		},
		cache: cache,
	}
}

func (d dialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	c, e := d.Dialer.DialContext(ctx, network, address)
	return conn{
		Conn:  c,
		cache: d.cache,
	}, e
}

func TestCustomDialer(t *testing.T) {
	client, err := mongo.NewClient(options.Client().ApplyURI("mongodb://testurl:27017").SetDialer(NewDialer()))
	if err != nil {
		t.Fatalf("error creating client: %v", err)
	}
	ctx := context.Background()
	err = client.Connect(ctx)
	if err != nil {
		t.Fatalf("error connecting: %v", err)
	}
	defer client.Disconnect(context.Background())
	coll := client.Database("test").Collection("test")
	_, err = coll.InsertOne(context.Background(), bson.D{{"text", "text"}})
	if err != nil {
		t.Fatalf("error inserting: %v", err)
	}
}
