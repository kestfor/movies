package achievements

import "testing"

func TestCatalogShape(t *testing.T) {
	definitions := Definitions()
	if err := ValidateCatalog(definitions); err != nil {
		t.Fatalf("validate catalog: %v", err)
	}
	if len(definitions) != 80 {
		t.Fatalf("definitions = %d, want 80", len(definitions))
	}
	secret, totalXP, prospective := 0, 0, 0
	for index, definition := range definitions {
		if definition.SortOrder != index+1 {
			t.Fatalf("sort order at %d = %d", index, definition.SortOrder)
		}
		if definition.Secret {
			secret++
		}
		if definition.AwardPolicy == AwardPolicySinceIntroduction {
			prospective++
		}
		totalXP += definition.XP
	}
	if secret != 22 {
		t.Fatalf("secret definitions = %d, want 22", secret)
	}
	if prospective != 30 {
		t.Fatalf("prospective definitions = %d, want 30", prospective)
	}
	if totalXP != 19050 {
		t.Fatalf("total XP = %d, want 19050", totalXP)
	}
}

func TestLevels(t *testing.T) {
	tests := []struct {
		xp    int
		level int
		title string
	}{
		{0, 1, "Зритель"},
		{49, 1, "Зритель"},
		{50, 2, "Зритель"},
		{4050, 10, "Куратор"},
		{8450, 14, "Легенда КиноКруга"},
		{9300, 14, "Легенда КиноКруга"},
		{9799, 14, "Легенда КиноКруга"},
		{9800, 15, "Алмаз КиноКруга"},
		{19050, 20, "Алмаз КиноКруга"},
	}
	for _, test := range tests {
		level := Level(test.xp)
		if level != test.level || RankTitle(level) != test.title {
			t.Fatalf("xp %d: got level=%d title=%q", test.xp, level, RankTitle(level))
		}
	}
}

func TestCatalogFingerprintStable(t *testing.T) {
	one, err := CatalogFingerprint(Definitions(), 1)
	if err != nil {
		t.Fatal(err)
	}
	reversed := Definitions()
	for left, right := 0, len(reversed)-1; left < right; left, right = left+1, right-1 {
		reversed[left], reversed[right] = reversed[right], reversed[left]
	}
	two, err := CatalogFingerprint(reversed, 1)
	if err != nil {
		t.Fatal(err)
	}
	if one != two {
		t.Fatalf("fingerprint depends on catalog ordering")
	}
	three, err := CatalogFingerprint(reversed, 2)
	if err != nil {
		t.Fatal(err)
	}
	if one == three {
		t.Fatalf("fingerprint ignored evaluator version")
	}
}
