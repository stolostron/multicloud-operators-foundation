// Copyright (c) 2020 Red Hat, Inc.

package app

import (
	"testing"
	"time"
)

func Test_convertLeaderElectionOptions(t *testing.T) {
	second1 := 1 * time.Second
	second2 := 2 * time.Second
	second3 := 3 * time.Second

	tests := []struct {
		name                  string
		leaseDuration         int
		renewDeadline         int
		retryPeriod           int
		wantLeaseDurationTime *time.Duration
		wantRenewDeadlineTime *time.Duration
		wantRetryPeriodTime   *time.Duration
	}{
		{
			name:                  "do not set the options",
			leaseDuration:         -1,
			renewDeadline:         -1,
			retryPeriod:           -1,
			wantLeaseDurationTime: nil,
			wantRenewDeadlineTime: nil,
			wantRetryPeriodTime:   nil,
		},
		{
			name:                  "set value for the options",
			leaseDuration:         1,
			renewDeadline:         2,
			retryPeriod:           3,
			wantLeaseDurationTime: &second1,
			wantRenewDeadlineTime: &second2,
			wantRetryPeriodTime:   &second3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotLeaseDurationTime, gotRenewDeadlineTime, gotRetryPeriodTime := convertLeaderElectionOptions(tt.leaseDuration, tt.renewDeadline, tt.retryPeriod)
			if gotLeaseDurationTime != tt.wantLeaseDurationTime {
				if *gotLeaseDurationTime != *tt.wantLeaseDurationTime {
					t.Errorf("convertLeaderElectionOptions() gotLeaseDurationTime = %v, want %v", gotLeaseDurationTime, tt.wantLeaseDurationTime)
				}
			}
			if gotRenewDeadlineTime != tt.wantRenewDeadlineTime {
				if *gotRenewDeadlineTime != *tt.wantRenewDeadlineTime {
					t.Errorf("convertLeaderElectionOptions() gotRenewDeadlineTime = %v, want %v", gotRenewDeadlineTime, tt.wantRenewDeadlineTime)
				}
			}
			if gotRetryPeriodTime != tt.wantRetryPeriodTime {
				if *gotRetryPeriodTime != *tt.wantRetryPeriodTime {
					t.Errorf("convertLeaderElectionOptions() gotRetryPeriodTime = %v, want %v", gotRetryPeriodTime, tt.wantRetryPeriodTime)
				}
			}
		})
	}
}
