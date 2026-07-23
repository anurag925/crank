package seedgen

import "testing"

func Test_updateSeeder(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		seederPath string
		info       *StructInfo
		opts       Options
		wantErr    bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotErr := updateSeeder(tt.seederPath, tt.info, tt.opts)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("updateSeeder() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("updateSeeder() succeeded unexpectedly")
			}
		})
	}
}
