package claudecode_test

import (
	"testing"

	"github.com/jackchuka/ccli/internal/agent"
	"github.com/jackchuka/ccli/internal/claudecode"
)

func TestListRules(t *testing.T) {
	tests := []struct {
		name       string
		paths      claudecode.Paths
		wantScopes map[agent.Scope]int
		wantMin    int
	}{
		{
			name: "global and project rules",
			paths: claudecode.Paths{
				RulesDir:   "testdata/rules",
				ProjectDir: "testdata/project",
			},
			wantScopes: map[agent.Scope]int{
				agent.ScopeGlobal:  1,
				agent.ScopeProject: 1,
			},
			wantMin: 2,
		},
		{
			name: "global only",
			paths: claudecode.Paths{
				RulesDir: "testdata/rules",
			},
			wantScopes: map[agent.Scope]int{
				agent.ScopeGlobal: 1,
			},
			wantMin: 1,
		},
		{
			name: "nonexistent directories",
			paths: claudecode.Paths{
				RulesDir:   "testdata/nonexistent",
				ProjectDir: "testdata/also-nonexistent",
			},
			wantMin: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := claudecode.NewAgent(tt.paths)
			rules, err := a.ListRules()
			if err != nil {
				t.Fatalf("ListRules: %v", err)
			}
			if len(rules) < tt.wantMin {
				t.Fatalf("got %d rules, want >= %d", len(rules), tt.wantMin)
			}
			scopes := map[agent.Scope]int{}
			for _, r := range rules {
				scopes[r.Scope]++
			}
			for scope, want := range tt.wantScopes {
				if got := scopes[scope]; got != want {
					t.Errorf("%s rules = %d, want %d", scope, got, want)
				}
			}
		})
	}
}

func TestGetRule(t *testing.T) {
	a := claudecode.NewAgent(claudecode.Paths{
		RulesDir:   "testdata/rules",
		ProjectDir: "testdata/project",
	})

	t.Run("existing rule", func(t *testing.T) {
		rule, err := a.GetRule("code-style.md")
		if err != nil {
			t.Fatalf("GetRule: %v", err)
		}
		if rule.Scope != agent.ScopeGlobal {
			t.Errorf("scope = %q, want %q", rule.Scope, agent.ScopeGlobal)
		}
		if len(rule.Paths) != 2 {
			t.Errorf("paths count = %d, want 2", len(rule.Paths))
		}
	})

	t.Run("nonexistent rule", func(t *testing.T) {
		_, err := a.GetRule("nonexistent")
		if err == nil {
			t.Fatal("expected error for nonexistent rule")
		}
	})
}
