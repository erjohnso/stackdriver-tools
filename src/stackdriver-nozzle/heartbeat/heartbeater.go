/*
 * Copyright 2017 Google Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package heartbeat

import (
	"errors"
	"time"

	"github.com/cloudfoundry/lager"
)

type Heartbeater interface {
	Start()
	Increment(string)
	Stop()
}

type heartbeater struct {
	logger  lager.Logger
	trigger <-chan time.Time
	counter chan string
	done    chan struct{}
	started bool
}

func NewHeartbeater(logger lager.Logger, trigger <-chan time.Time) Heartbeater {
	counter := make(chan string)
	done := make(chan struct{})
	return &heartbeater{
		logger:  logger,
		trigger: trigger,
		counter: counter,
		done:    done,
		started: false,
	}
}

func (h *heartbeater) Start() {
	h.started = true
	go func() {
		counters := map[string]uint{}
		for {
			select {
			case <-h.trigger:
				h.logger.Info(
					"heartbeater", lager.Data{"counters": counters},
				)
				counters = map[string]uint{}
			case name := <-h.counter:
				counters[name] += 1
			case <-h.done:
				h.logger.Info(
					"heartbeater", lager.Data{"counters": counters},
				)
				return
			}
		}
	}()
}

func (h *heartbeater) Increment(name string) {
	if h.started {
		h.counter <- name
	} else {
		h.logger.Error(
			"heartbeater",
			errors.New("attempted to increment counter without starting heartbeater"),
		)
	}
}

func (h *heartbeater) Stop() {
	h.done <- struct{}{}
	h.started = false
}
