/*
Copyright 2025 The Crossplane Authors.
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

package client

import (
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

// DefaultDialOptions returns a set of default gRPC dial options.
// These options can be used when creating a gRPC connection to a server.
func DefaultDialOptions() []grpc.DialOption {
	return []grpc.DialOption{
		// This configures a gRPC client to use round robin load balancing.
		// See https://github.com/grpc/grpc/blob/v1.58.0/doc/load-balancing.md#load-balancing-policies
		grpc.WithDefaultServiceConfig(lbRoundRobin),
		
		// Configure keepalive parameters
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                30 * time.Second, // send pings every 30 seconds if there is no activity
			Timeout:             10 * time.Second, // wait 10 seconds for ping ack before considering the connection dead
			PermitWithoutStream: true,             // send pings even without active streams
		}),
	}
}

// WithTimeout returns a gRPC dial option with the specified timeout.
func WithTimeout(timeout time.Duration) grpc.DialOption {
	return grpc.WithTimeout(timeout)
}

// WithBlock returns a gRPC dial option that blocks until the connection is established.
func WithBlock() grpc.DialOption {
	return grpc.WithBlock()
}

// WithBackoffMaxDelay returns a gRPC dial option with the specified maximum backoff delay.
func WithBackoffMaxDelay(maxDelay time.Duration) grpc.DialOption {
	return grpc.WithBackoffMaxDelay(maxDelay)
}

// WithMaxRetries returns a gRPC dial option with a retry policy with the specified maximum number of retries.
func WithMaxRetries(maxRetries uint) grpc.DialOption {
	retryPolicy := `{
		"methodConfig": [{
			"name": [{"service": "external.v1alpha1.ExternalService"}],
			"retryPolicy": {
				"maxAttempts": ` + string(rune('0'+maxRetries)) + `,
				"initialBackoff": "0.1s",
				"maxBackoff": "1s",
				"backoffMultiplier": 2.0,
				"retryableStatusCodes": ["UNAVAILABLE"]
			}
		}]
	}`
	return grpc.WithDefaultServiceConfig(retryPolicy)
}

// WithUserAgent returns a gRPC dial option with the specified user agent.
func WithUserAgent(userAgent string) grpc.DialOption {
	return grpc.WithUserAgent(userAgent)
}