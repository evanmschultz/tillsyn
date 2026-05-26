package pretoolgate

import (
	"testing"
)

func TestGateSpec_EditPresenceAndNil(t *testing.T) {
	tests := []struct {
		name string
		spec GateSpec
		want bool // true if Edit is present (even if empty)
	}{
		{
			name: "Edit present but empty slice",
			spec: GateSpec{Edit: []string{}},
			want: true,
		},
		{
			name: "Edit is nil",
			spec: GateSpec{Edit: nil},
			want: false,
		},
		{
			name: "Edit with files",
			spec: GateSpec{Edit: []string{"//abs/file.go"}},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.spec.Edit != nil
			if got != tt.want {
				t.Errorf("Edit != nil: got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGateSpec_BuiltinEdits(t *testing.T) {
	tests := []struct {
		name string
		spec GateSpec
		want []string
	}{
		{
			name: "Edit with files",
			spec: GateSpec{Edit: []string{"//abs/file1.go", "//abs/file2.go"}},
			want: []string{"//abs/file1.go", "//abs/file2.go"},
		},
		{
			name: "Edit empty slice",
			spec: GateSpec{Edit: []string{}},
			want: []string{},
		},
		{
			name: "Edit nil (read-only role)",
			spec: GateSpec{Edit: nil},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.spec.BuiltinEdits()
			if len(got) != len(tt.want) {
				t.Errorf("BuiltinEdits() len: got %d, want %d", len(got), len(tt.want))
			}
			for i, v := range got {
				if i >= len(tt.want) || v != tt.want[i] {
					t.Errorf("BuiltinEdits()[%d]: got %q, want %q", i, v, tt.want[i])
				}
			}
			// Verify nil vs empty distinction.
			if (got == nil) != (tt.want == nil) {
				t.Errorf("BuiltinEdits() nil: got %v, want %v", got == nil, tt.want == nil)
			}
		})
	}
}

func TestGateSpec_BuiltinBashDeny(t *testing.T) {
	tests := []struct {
		name string
		spec GateSpec
		want []string
	}{
		{
			name: "BashDeny with patterns",
			spec: GateSpec{BashDeny: []string{"git commit", "git push"}},
			want: []string{"git commit", "git push"},
		},
		{
			name: "BashDeny empty",
			spec: GateSpec{BashDeny: []string{}},
			want: []string{},
		},
		{
			name: "BashDeny nil",
			spec: GateSpec{BashDeny: nil},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.spec.BuiltinBashDeny()
			if len(got) != len(tt.want) {
				t.Errorf("BuiltinBashDeny() len: got %d, want %d", len(got), len(tt.want))
			}
			for i, v := range got {
				if i >= len(tt.want) || v != tt.want[i] {
					t.Errorf("BuiltinBashDeny()[%d]: got %q, want %q", i, v, tt.want[i])
				}
			}
			if (got == nil) != (tt.want == nil) {
				t.Errorf("BuiltinBashDeny() nil: got %v, want %v", got == nil, tt.want == nil)
			}
		})
	}
}

func TestGateSpec_CodexWritableDirs(t *testing.T) {
	tests := []struct {
		name string
		spec GateSpec
		want []string
	}{
		{
			name: "WritableDirs with paths",
			spec: GateSpec{WritableDirs: []string{"/abs/dir1", "/abs/dir2"}},
			want: []string{"/abs/dir1", "/abs/dir2"},
		},
		{
			name: "WritableDirs empty",
			spec: GateSpec{WritableDirs: []string{}},
			want: []string{},
		},
		{
			name: "WritableDirs nil",
			spec: GateSpec{WritableDirs: nil},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.spec.CodexWritableDirs()
			if len(got) != len(tt.want) {
				t.Errorf("CodexWritableDirs() len: got %d, want %d", len(got), len(tt.want))
			}
			for i, v := range got {
				if i >= len(tt.want) || v != tt.want[i] {
					t.Errorf("CodexWritableDirs()[%d]: got %q, want %q", i, v, tt.want[i])
				}
			}
			if (got == nil) != (tt.want == nil) {
				t.Errorf("CodexWritableDirs() nil: got %v, want %v", got == nil, tt.want == nil)
			}
		})
	}
}

func TestGateSpec_CodexBashDeny(t *testing.T) {
	tests := []struct {
		name string
		spec GateSpec
		want []string
	}{
		{
			name: "BashDeny with git patterns",
			spec: GateSpec{BashDeny: []string{"git commit", "mage install", "go get"}},
			want: []string{"git commit", "mage install", "go get"},
		},
		{
			name: "BashDeny empty",
			spec: GateSpec{BashDeny: []string{}},
			want: []string{},
		},
		{
			name: "BashDeny nil",
			spec: GateSpec{BashDeny: nil},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.spec.CodexBashDeny()
			if len(got) != len(tt.want) {
				t.Errorf("CodexBashDeny() len: got %d, want %d", len(got), len(tt.want))
			}
			for i, v := range got {
				if i >= len(tt.want) || v != tt.want[i] {
					t.Errorf("CodexBashDeny()[%d]: got %q, want %q", i, v, tt.want[i])
				}
			}
			if (got == nil) != (tt.want == nil) {
				t.Errorf("CodexBashDeny() nil: got %v, want %v", got == nil, tt.want == nil)
			}
		})
	}
}

func TestGateSpec_NetworkAccess(t *testing.T) {
	tests := []struct {
		name string
		spec GateSpec
		want bool
	}{
		{
			name: "Network true",
			spec: GateSpec{Network: true},
			want: true,
		},
		{
			name: "Network false (default)",
			spec: GateSpec{Network: false},
			want: false,
		},
		{
			name: "Zero value defaults to false",
			spec: GateSpec{},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.spec.NetworkAccess()
			if got != tt.want {
				t.Errorf("NetworkAccess(): got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGateSpec_CompleteSpec(t *testing.T) {
	// Comprehensive test with all fields populated.
	spec := GateSpec{
		Edit:         []string{"//abs/file1.go", "//abs/file2_test.go"},
		WritableDirs: []string{"/abs/droplet-dir"},
		BashDeny: []string{
			"git commit", "git push", "git add", "git reset",
			"mage install", "go get", "go mod",
		},
		Network: false,
	}

	// All accessors should return the expected values.
	if len(spec.BuiltinEdits()) != 2 {
		t.Errorf("BuiltinEdits(): got %d files, want 2", len(spec.BuiltinEdits()))
	}

	if len(spec.CodexWritableDirs()) != 1 {
		t.Errorf("CodexWritableDirs(): got %d dirs, want 1", len(spec.CodexWritableDirs()))
	}

	if len(spec.CodexBashDeny()) != 7 {
		t.Errorf("CodexBashDeny(): got %d patterns, want 7", len(spec.CodexBashDeny()))
	}

	if spec.NetworkAccess() {
		t.Errorf("NetworkAccess(): got true, want false")
	}
}

func TestGateSpec_ReadOnlyRole(t *testing.T) {
	// A read-only role (plan-qa, build-qa, closeout) has Edit: []string{} (present but empty).
	spec := GateSpec{
		Edit:         []string{}, // Present-empty = read-only
		WritableDirs: nil,        // Not applicable
		BashDeny:     []string{"git commit", "git push"},
		Network:      false,
	}

	// Edit should be present (not nil).
	if spec.Edit == nil {
		t.Errorf("Edit should be present (not nil) for read-only role")
	}

	// BuiltinEdits should return an empty slice, not nil.
	edits := spec.BuiltinEdits()
	if edits == nil {
		t.Errorf("BuiltinEdits(): got nil, want empty slice []")
	}
	if len(edits) != 0 {
		t.Errorf("BuiltinEdits(): got %d items, want 0", len(edits))
	}
}

func TestGateSpec_NilReceiver(t *testing.T) {
	// All accessors should handle nil receiver gracefully.
	var nilSpec *GateSpec

	if nilSpec.BuiltinEdits() != nil {
		t.Errorf("BuiltinEdits() on nil receiver: got non-nil, want nil")
	}
	if nilSpec.BuiltinBashDeny() != nil {
		t.Errorf("BuiltinBashDeny() on nil receiver: got non-nil, want nil")
	}
	if nilSpec.CodexWritableDirs() != nil {
		t.Errorf("CodexWritableDirs() on nil receiver: got non-nil, want nil")
	}
	if nilSpec.CodexBashDeny() != nil {
		t.Errorf("CodexBashDeny() on nil receiver: got non-nil, want nil")
	}
	if nilSpec.NetworkAccess() {
		t.Errorf("NetworkAccess() on nil receiver: got true, want false")
	}
}

func TestGateSpec_ZeroValue(t *testing.T) {
	// All fields should be nil/false by default.
	spec := GateSpec{}

	if spec.Edit != nil {
		t.Errorf("Zero GateSpec.Edit: got non-nil, want nil")
	}
	if spec.WritableDirs != nil {
		t.Errorf("Zero GateSpec.WritableDirs: got non-nil, want nil")
	}
	if spec.BashDeny != nil {
		t.Errorf("Zero GateSpec.BashDeny: got non-nil, want nil")
	}
	if spec.Network {
		t.Errorf("Zero GateSpec.Network: got true, want false")
	}

	if spec.BuiltinEdits() != nil {
		t.Errorf("BuiltinEdits() on zero value: got non-nil, want nil")
	}
	if spec.CodexWritableDirs() != nil {
		t.Errorf("CodexWritableDirs() on zero value: got non-nil, want nil")
	}
}

func TestGateSpec_AllFieldsCombinations(t *testing.T) {
	tests := []struct {
		name string
		spec GateSpec
		want struct {
			edits        []string
			bashDeny     []string
			writableDirs []string
			network      bool
		}
	}{
		{
			name: "all fields set",
			spec: GateSpec{
				Edit:         []string{"//a.go", "//b.go"},
				WritableDirs: []string{"/tmp"},
				BashDeny:     []string{"git push"},
				Network:      true,
			},
			want: struct {
				edits        []string
				bashDeny     []string
				writableDirs []string
				network      bool
			}{
				edits:        []string{"//a.go", "//b.go"},
				bashDeny:     []string{"git push"},
				writableDirs: []string{"/tmp"},
				network:      true,
			},
		},
		{
			name: "minimal spec",
			spec: GateSpec{
				Edit:    []string{"//single.go"},
				Network: false,
			},
			want: struct {
				edits        []string
				bashDeny     []string
				writableDirs []string
				network      bool
			}{
				edits:        []string{"//single.go"},
				bashDeny:     nil,
				writableDirs: nil,
				network:      false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.spec.BuiltinEdits(); len(got) != len(tt.want.edits) {
				t.Errorf("BuiltinEdits() len: got %d, want %d", len(got), len(tt.want.edits))
			}
			if got := tt.spec.BuiltinBashDeny(); (got == nil) != (tt.want.bashDeny == nil) {
				t.Errorf("BuiltinBashDeny() nil: got %v, want %v", got == nil, tt.want.bashDeny == nil)
			}
			if got := tt.spec.CodexWritableDirs(); (got == nil) != (tt.want.writableDirs == nil) {
				t.Errorf("CodexWritableDirs() nil: got %v, want %v", got == nil, tt.want.writableDirs == nil)
			}
			if got := tt.spec.NetworkAccess(); got != tt.want.network {
				t.Errorf("NetworkAccess(): got %v, want %v", got, tt.want.network)
			}
		})
	}
}
