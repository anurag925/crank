package scaffold

import "testing"

func Test_wireCompositionRoot(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		projectDir string
		r          Resource
		wantErr    bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotErr := wireCompositionRoot(tt.projectDir, tt.r)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("wireCompositionRoot() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("wireCompositionRoot() succeeded unexpectedly")
			}
		})
	}
}
