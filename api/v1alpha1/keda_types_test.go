/*
Copyright 2022.

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

package v1alpha1

import (
	"fmt"
	"testing"
)

type zero interface {
	zero() string
}

var (
	testNilOperatorLogLevel      *LogLevel
	testNilLogFormat             *LogFormat
	testNilLogTimeEncoding       *LogTimeEncoding
	testNilMetricsServerLogLevel *LogLevel

	testZeroOperatorLogLevel      LogLevel
	testZeroLogFormat             LogFormat
	testZeroLogTimeEncoding       LogTimeEncoding
	testZeroMetricsServerLogLevel LogLevel

	testOperatorLogLevelDebug   = CommonLogLevelDebug
	testOperatorLogLevelInfo    = CommonLogLevelInfo
	testOperatorLogLevelError   = CommonLogLevelError
	testLogFormatJSON           = LogFormatJSON
	testLogFormatConsole        = LogFormatConsole
	testTimeEncodingEpoch       = TimeEncodingEpoch
	testTimeEncodingMillis      = TimeEncodingMillis
	testTimeEncodingNano        = TimeEncodingNano
	testTimeEncodingISO8601     = TimeEncodingISO8601
	testTimeEncodingRFC3339     = TimeEncodingRFC3339
	testTimeEncodingRFC3339Nano = TimeEncodingRFC3339Nano
)

func Test_zero(t *testing.T) {
	type test struct {
		name string
		f    zero
		want string
	}

	tests := []test{
		{
			name: "OperatorLogLevel zero value",
			want: "info",
			f:    testNilOperatorLogLevel,
		},
		{
			name: "OperatorLogLevel nil",
			want: "info",
			f:    &testZeroOperatorLogLevel,
		},
		{
			name: "LogFormat zero value",
			want: "json",
			f:    &testZeroLogFormat,
		},
		{
			name: "LogFormat nil",
			want: "json",
			f:    testNilLogFormat,
		},
		{
			name: "LogTimeEncoding zero value",
			want: "rfc3339",
			f:    &testZeroLogTimeEncoding,
		},
		{
			name: "LogTimeEncoding nil",
			want: "rfc3339",
			f:    testNilLogTimeEncoding,
		},
		{
			name: "MetricsServerLogLevel zero value",
			want: "info",
			f:    &testZeroMetricsServerLogLevel,
		},
		{
			name: "MetricsServerLogLevel nil",
			want: "info",
			f:    testNilMetricsServerLogLevel,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.f.zero()
			if got != tt.want {
				t.Errorf("zero() = %s, want %s", got, tt.want)
			}
		})
	}
}

func Test_string(t *testing.T) {
	type test struct {
		name     string
		stringer fmt.Stringer
		want     string
	}
	tests := []test{
		{
			name:     "OperatorLogLevelDebug",
			stringer: &testOperatorLogLevelDebug,
			want:     "--zap-log-level=debug",
		},
		{
			name:     "OperatorLogLevelInfo",
			stringer: &testOperatorLogLevelInfo,
			want:     "--zap-log-level=info",
		},
		{
			name:     "OperatorLogLevelError",
			stringer: &testOperatorLogLevelError,
			want:     "--zap-log-level=error",
		},
		{
			name:     "LogFormatJSON",
			stringer: &testLogFormatJSON,
			want:     "--zap-encoder=json",
		},
		{
			name:     "LogFormatConsole",
			stringer: &testLogFormatConsole,
			want:     "--zap-encoder=console",
		},
		{
			name:     "TimeEncodingEpoch",
			stringer: &testTimeEncodingEpoch,
			want:     "--zap-time-encoding=epoch",
		},
		{
			name:     "TimeEncodingMillis",
			stringer: &testTimeEncodingMillis,
			want:     "--zap-time-encoding=millis",
		},
		{
			name:     "TimeEncodingNano",
			stringer: &testTimeEncodingNano,
			want:     "--zap-time-encoding=nano",
		},
		{
			name:     "TimeEncodingISO8601",
			stringer: &testTimeEncodingISO8601,
			want:     "--zap-time-encoding=iso8601",
		},
		{
			name:     "TimeEncodingRFC3339",
			stringer: &testTimeEncodingRFC3339,
			want:     "--zap-time-encoding=rfc3339",
		},
		{
			name:     "TimeEncodingRFC3339Nano",
			stringer: &testTimeEncodingRFC3339Nano,
			want:     "--zap-time-encoding=rfc3339nano",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.stringer.String()
			if got != tt.want {
				t.Errorf("String() = %s, want %s", got, tt.want)
			}
		})
	}
}
