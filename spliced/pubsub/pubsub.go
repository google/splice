/*
Copyright 2018 Google Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package pubsub abstracts the pubsub client calls out of the main SpliceD code.
package pubsub

import (
	"golang.org/x/net/context"

	"cloud.google.com/go/pubsub"
)

var (
	// NewClient passes through the underlying NewClient
	NewClient = pubsub.NewClient
)

// NewJoinRequest pulls messages from the publisher.
func NewJoinRequest(ctx context.Context, client *pubsub.Client, topic string) (string, error) {
	data := ""
	sub := client.Subscription(topic)
	sub.ReceiveSettings.MaxOutstandingMessages = 1
	cctx, cancel := context.WithCancel(ctx)
	err := sub.Receive(cctx, func(ctx context.Context, msg *pubsub.Message) {
		data = string(msg.Data)
		msg.Ack()
		cancel()
	})
	return data, err
}
