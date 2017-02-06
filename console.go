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

import (
	"log"

	"github.com/fatih/color"
)

const CONSOLE MODE = "console"

// Color set for different levels.
var colors = []func(a ...interface{}) string{
	color.New(color.FgBlue).SprintFunc(),   // Trace
	color.New(color.FgGreen).SprintFunc(),  // Info
	color.New(color.FgYellow).SprintFunc(), // Warn
	color.New(color.FgRed).SprintFunc(),    // Error
	color.New(color.FgHiRed).SprintFunc(),  // Fatal
}

type ConsoleConfig struct {
	// Minimum level of messages to be processed.
	Level LEVEL
	// Buffer size defines how many messages can be queued before hangs.
	BufferSize int64
}

type console struct {
	*log.Logger

	level     LEVEL
	msgChan   chan *Message
	quitChan  chan struct{}
	errorChan chan<- error
}

func newConsole() Logger {
	return &console{
		Logger:   log.New(color.Output, "", log.Ldate|log.Ltime),
		quitChan: make(chan struct{}),
	}
}

func (c *console) Level() LEVEL { return c.level }

func (c *console) Init(v interface{}) error {
	cfg, ok := v.(ConsoleConfig)
	if !ok {
		return ErrConfigObject{"ConsoleConfig", v}
	}

	if !isValidLevel(cfg.Level) {
		return ErrInvalidLevel{}
	}
	c.level = cfg.Level

	c.msgChan = make(chan *Message, cfg.BufferSize)
	return nil
}

func (c *console) ExchangeChans(errorChan chan<- error) (chan *Message, chan struct{}) {
	c.errorChan = errorChan
	return c.msgChan, c.quitChan
}

func (c *console) write(msg *Message) {
	c.Logger.Print(colors[msg.Level](msg.Body))
}

func (c *console) Start() {
	for {
		select {
		case msg := <-c.msgChan:
			c.write(msg)
		case <-c.quitChan:
			return
		}
	}
}

func (c *console) Flush() {
	for {
		if len(c.msgChan) == 0 {
			return
		}

		c.write(<-c.msgChan)
	}
}

func (c *console) Destroy() {
	close(c.msgChan)
	close(c.quitChan)
}

func init() {
	Register(CONSOLE, newConsole)
}
