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

package filter_test

import (
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/filter"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/mocks"
	"github.com/cloudfoundry/sonde-go/events"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Filter", func() {
	var (
		fhClient    *mocks.FirehoseClient
		heartbeater *mocks.Heartbeater
	)

	BeforeEach(func() {
		fhClient = mocks.NewFirehoseClient()
		heartbeater = mocks.NewHeartbeater()
	})

	It("can accept an empty filter and blocks all events", func() {
		emptyFilter := []string{}
		f, err := filter.New(fhClient, emptyFilter, heartbeater)
		Expect(err).To(BeNil())
		Expect(f).NotTo(BeNil())
		messages, errs := f.Connect()

		fhClient.SendEvents(
			events.Envelope_HttpStart,
			events.Envelope_HttpStop,
			events.Envelope_HttpStartStop,
			events.Envelope_LogMessage,
			events.Envelope_ValueMetric,
			events.Envelope_CounterEvent,
			events.Envelope_Error,
			events.Envelope_ContainerMetric,
		)

		Consistently(messages).ShouldNot(Receive())
		Consistently(errs).ShouldNot(Receive())
	})

	It("can accept a single event to filter", func() {
		singleFilter := []string{"Error"}
		f, err := filter.New(fhClient, singleFilter, heartbeater)
		Expect(err).To(BeNil())
		Expect(f).NotTo(BeNil())
		messages, errs := f.Connect()

		eventType := events.Envelope_Error
		event := &events.Envelope{EventType: &eventType}
		fhClient.Messages <- event
		Eventually(messages).Should(Receive(Equal(event)))

		fhClient.SendEvents(
			events.Envelope_HttpStart,
			events.Envelope_HttpStop,
			events.Envelope_HttpStartStop,
			events.Envelope_LogMessage,
			events.Envelope_ValueMetric,
			events.Envelope_CounterEvent,
			events.Envelope_ContainerMetric,
		)
		Consistently(messages).ShouldNot(Receive())
		Consistently(errs).ShouldNot(Receive())
	})

	It("can accept multiple events to filter", func() {
		multiFilter := []string{"Error", "LogMessage"}
		f, err := filter.New(fhClient, multiFilter, heartbeater)
		Expect(err).To(BeNil())
		Expect(f).NotTo(BeNil())
		messages, errs := f.Connect()

		eventType := events.Envelope_Error
		event := &events.Envelope{EventType: &eventType}
		fhClient.Messages <- event
		Eventually(messages).Should(Receive(Equal(event)))

		eventType = events.Envelope_LogMessage
		event = &events.Envelope{EventType: &eventType}
		fhClient.Messages <- event
		Eventually(messages).Should(Receive(Equal(event)))

		fhClient.SendEvents(
			events.Envelope_HttpStart,
			events.Envelope_HttpStop,
			events.Envelope_HttpStartStop,
			events.Envelope_ValueMetric,
			events.Envelope_CounterEvent,
			events.Envelope_ContainerMetric,
		)
		Consistently(messages).ShouldNot(Receive())
		Consistently(errs).ShouldNot(Receive())
	})

	It("rejects invalid events", func() {
		invalidFilter := []string{"Error", "FakeEvent111"}
		f, err := filter.New(fhClient, invalidFilter, heartbeater)
		Expect(err).NotTo(BeNil())
		Expect(f).To(BeNil())
	})

	It("increments the heartbeater", func() {
		multiFilter := []string{"Error", "LogMessage"}
		f, _ := filter.New(fhClient, multiFilter, heartbeater)
		messages, _ := f.Connect()

		fhClient.SendEvents(
			events.Envelope_HttpStart,
			events.Envelope_HttpStop,
			events.Envelope_HttpStartStop,
		)

		Eventually(func() int {
			return heartbeater.GetCount("filter.events")
		}).Should(Equal(3))

		go fhClient.SendEvents(
			events.Envelope_LogMessage,
			events.Envelope_ValueMetric,
			events.Envelope_CounterEvent,
			events.Envelope_ContainerMetric,
		)

		<-messages

		Eventually(func() int {
			return heartbeater.GetCount("filter.events")
		}).Should(Equal(7))
	})
})
