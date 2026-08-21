package renditions

import (
	"reflect"
	"testing"
)

func TestHeights(t *testing.T) {
	tests := []struct {
		source int
		want   []int
	}{
		{480, []int{144, 240, 360, 480}},
		{720, []int{144, 240, 360, 480, 720}},
		{768, []int{144, 240, 360, 480, 720}},
		{1080, []int{144, 240, 360, 480, 720, 1080}},
		{1440, []int{144, 240, 360, 480, 720, 1080, 1440}},
		{2160, []int{144, 240, 360, 480, 720, 1080, 1440}},
		{90, []int{90}},
	}
	for _, test := range tests {
		if got := Heights(test.source); !reflect.DeepEqual(got, test.want) {
			t.Errorf("Heights(%d)=%v want %v", test.source, got, test.want)
		}
	}
}
