package main

import (
	"reflect"
	"testing"
)

func TestBuildPartitionKeys(t *testing.T) {
	tests := []struct {
		name       string
		recordData FirehoseEventRecordData
		expected   map[string]string
	}{
		{
			name: "valid record data",
			recordData: FirehoseEventRecordData{
				TSEpochMillis: 1713589641958, // Sat, 20 Apr 2024 05:07:21 GMT UTC
			},
			expected: map[string]string{
				"year":  "2024",
				"month": "04",
				"day":   "20",
				"hour":  "05",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := buildPartitionKeys(test.recordData)
			if !reflect.DeepEqual(result, test.expected) {
				t.Errorf("buildPartitionKeys(%v) = %v, expected %v", test.recordData, result, test.expected)
			}
		})
	}
}
