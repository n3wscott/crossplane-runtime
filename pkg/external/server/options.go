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

package server

import (
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

// DefaultServerOptions returns a set of default gRPC server options.
func DefaultServerOptions() []grpc.ServerOption {
	return []grpc.ServerOption{
		// Configure keepalive parameters
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Time:    60 * time.Second, // send pings every 60 seconds if there's no activity
			Timeout: 20 * time.Second, // wait 20 seconds for ping ack before considering the connection dead
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             30 * time.Second, // minimum allowed time between client pings
			PermitWithoutStream: true,             // allow pings even without active streams
		}),
		// Set reasonable size limits
		grpc.MaxRecvMsgSize(4 * 1024 * 1024), // 4 MiB
		grpc.MaxSendMsgSize(4 * 1024 * 1024), // 4 MiB
	}
}

// WithMaxConcurrentStreams returns a gRPC server option that limits the maximum number of concurrent streams.
func WithMaxConcurrentStreams(max uint32) grpc.ServerOption {
	return grpc.MaxConcurrentStreams(max)
}

// WithMaxConnectionAge returns a gRPC server option that sets the maximum connection age.
func WithMaxConnectionAge(d time.Duration) grpc.ServerOption {
	return grpc.MaxConnectionAge(d)
}

// WithMaxConnectionAgeGrace returns a gRPC server option that sets the maximum connection age grace period.
func WithMaxConnectionAgeGrace(d time.Duration) grpc.ServerOption {
	return grpc.MaxConnectionAgeGrace(d)
}

// WithMaxConnectionIdle returns a gRPC server option that sets the maximum connection idle time.
func WithMaxConnectionIdle(d time.Duration) grpc.ServerOption {
	return grpc.MaxConnectionIdle(d)
}