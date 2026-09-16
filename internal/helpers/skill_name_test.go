package helpers

import "testing"

func TestNormalizeSkillName(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		" Create Plan ":               "create-plan",
		"implementation__strategy":    "implementation-strategy",
		"code-change verification":    "code-change-verification",
		"self---healing___executor  ": "self-healing-executor",
		"@#$%^":                       "",
		"":                            "",
	}

	for input, want := range cases {
		got := NormalizeSkillName(input)
		if got != want {
			t.Fatalf("NormalizeSkillName(%q) = %q, want %q", input, got, want)
		}
	}
}
