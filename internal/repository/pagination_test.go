package repository

import "testing"

func TestNormalizePageRequestBounds(t *testing.T) {
	tests := []struct {
		name string
		in   PageRequest
		want PageRequest
	}{
		{name: "defaults when zero", in: PageRequest{}, want: PageRequest{Page: DefaultPage, PageSize: DefaultPageSize}},
		{name: "page floored", in: PageRequest{Page: -5, PageSize: 10}, want: PageRequest{Page: DefaultPage, PageSize: 10}},
		{name: "size floored", in: PageRequest{Page: 2, PageSize: -1}, want: PageRequest{Page: 2, PageSize: DefaultPageSize}},
		{name: "size capped", in: PageRequest{Page: 2, PageSize: MaxPageSize + 50}, want: PageRequest{Page: 2, PageSize: MaxPageSize}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := normalizePageRequest(tc.in)
			if got != tc.want {
				t.Fatalf("normalizePageRequest(%+v) = %+v, want %+v", tc.in, got, tc.want)
			}
		})
	}
}

func TestCalcTotalPages(t *testing.T) {
	tests := []struct {
		total    int64
		pageSize int
		want     int
	}{
		{total: 0, pageSize: 10, want: 0},
		{total: 10, pageSize: 0, want: 0},
		{total: 1, pageSize: 20, want: 1},
		{total: 20, pageSize: 20, want: 1},
		{total: 21, pageSize: 20, want: 2},
	}
	for _, tc := range tests {
		got := calcTotalPages(tc.total, tc.pageSize)
		if got != tc.want {
			t.Fatalf("calcTotalPages(%d, %d) = %d, want %d", tc.total, tc.pageSize, got, tc.want)
		}
	}
}
