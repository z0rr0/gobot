package db

import (
	"maps"
	"testing"
)

func TestSetToString(t *testing.T) {
	testCases := []struct {
		name string
		set  map[string]struct{}
		want string
	}{
		{name: "empty"},
		{name: "one", set: map[string]struct{}{"one": {}}, want: "[\"one\"]"},
		{name: "two", set: map[string]struct{}{"one": {}, "two": {}}, want: "[\"one\",\"two\"]"},
		{name: "order", set: map[string]struct{}{"two": {}, "one": {}}, want: "[\"one\",\"two\"]"},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			got, err := setToString(tc.set)
			if err != nil {
				t.Fatalf("failed to convert set to string: %v", err)
			}
			if got != tc.want {
				t.Errorf("failed compare strings, got '%v' want '%v'", got, tc.want)
			}
		})
	}
}

func TestStringToSet(t *testing.T) {
	testCases := []struct {
		name string
		str  string
		want map[string]struct{}
	}{
		{name: "empty"},
		{name: "one", str: "[\"one\"]", want: map[string]struct{}{"one": {}}},
		{name: "two", str: "[\"one\",\"two\"]", want: map[string]struct{}{"one": {}, "two": {}}},
		{name: "three", str: "[\"one\",\"two\",\"three\"]", want: map[string]struct{}{"one": {}, "two": {}, "three": {}}},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			got, err := stringToSet(tc.str)
			if err != nil {
				t.Fatalf("failed to convert string to set: %v", err)
			}
			if !maps.Equal(got, tc.want) {
				t.Errorf("failed compare maps, got\n%+v\n want\n%+v", got, tc.want)
			}
		})
	}
}
