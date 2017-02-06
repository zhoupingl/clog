// Copyright 2017 Unknwon
//
// Licensed under the Apache License, Version 2.0 (the "License"): you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package clog

import "fmt"

// Logger is an interface for a logger adapter with specific mode and level.
type Logger interface {
	// Level returns minimum level of given logger.
	Level() LEVEL
	// Init accepts a config struct specific for given logger and performs any necessary initialization.
	Init(interface{}) error
	// ExchangeChans accepts error channel, and returns message receive and quit channels.
	ExchangeChans(chan<- error) (chan *Message, chan struct{})
	// Start starts message processing.
	Start()
	// Flush makes sure all messages have been written to the output and safe to exit.
	Flush()
	// Destroy releases all resources.
	Destroy()
}

type Factory func() Logger

// factories keeps factory function of registered loggers.
var factories = map[MODE]Factory{}

func Register(mode MODE, f Factory) {
	if f == nil {
		panic("clog: register function is nil")
	}
	if factories[mode] != nil {
		panic("clog: register duplicated mode '" + mode + "'")
	}
	factories[mode] = f
}

type receiver struct {
	Logger
	mode     MODE
	msgChan  chan *Message
	quitChan chan struct{}
}

func (r *receiver) close() {
	r.quitChan <- struct{}{}
	r.Flush()
	r.Destroy()
}

var (
	// receivers is a list of loggers with their message channel for broadcasting.
	receivers []*receiver

	errorChan = make(chan error, 5)
	quitChan  = make(chan struct{})
)

func init() {
	go func() {
		for {
			select {
			case err := <-errorChan:
				fmt.Println("clog: unable to write message: %v", err)
			case <-quitChan:
				return
			}
		}
	}()
}

// NewLogger initializes and appends a new logger to the receiver list.
// Calling this function multiple times will overwrite previous logger with same mode.
func NewLogger(mode MODE, cfg interface{}) error {
	factory, ok := factories[mode]
	if !ok {
		return fmt.Errorf("unknown mode '%s'", mode)
	}

	logger := factory()
	if err := logger.Init(cfg); err != nil {
		return fmt.Errorf("fail to initialize: %v", err)
	}
	msgChan, quitChan := logger.ExchangeChans(errorChan)

	// Check and replace previous logger.
	hasFound := false
	for i := range receivers {
		if receivers[i].mode == mode {
			hasFound = true

			// Release previous logger.
			receivers[i].close()

			// Update info to new one.
			receivers[i].Logger = logger
			receivers[i].msgChan = msgChan
			receivers[i].quitChan = quitChan
			break
		}
	}
	if !hasFound {
		receivers = append(receivers, &receiver{
			Logger:   logger,
			mode:     mode,
			msgChan:  msgChan,
			quitChan: quitChan,
		})
	}

	go logger.Start()
	return nil
}
