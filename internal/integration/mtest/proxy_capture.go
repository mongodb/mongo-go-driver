// Copyright (C) MongoDB, Inc. 2025-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mtest

import (
	"sync"
)

// ProxyCapture provides a FIFO channel for handshake messages passed
// through the mtest proxyDialer.
type ProxyCapture struct {
	messages chan *ProxyMessage
	mu       sync.Mutex
}

func newProxyCapture(bufferSize int) *ProxyCapture {
	return &ProxyCapture{
		messages: make(chan *ProxyMessage, bufferSize),
	}
}

func (hc *ProxyCapture) Capture(msg *ProxyMessage) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	hc.messages <- msg
}

func (hc *ProxyCapture) TryNext() *ProxyMessage {
	select {
	case msg := <-hc.messages:
		return msg
	default:
		return nil
	}
}

// Drain removes all messages from the channel and returns them as a slice.
func (hc *ProxyCapture) Drain() []*ProxyMessage {
	messages := []*ProxyMessage{}
	for {
		select {
		case msg := <-hc.messages:
			messages = append(messages, msg)
		default:
			return messages
		}
	}
}
