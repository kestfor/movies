package achievements

import "testing"

func TestCatalogShape(t *testing.T) {
	definitions := Definitions()
	if err := ValidateCatalog(definitions); err != nil {
		t.Fatalf("validate catalog: %v", err)
	}
	if len(definitions) != 50 {
		t.Fatalf("definitions = %d, want 50", len(definitions))
	}
	secret, totalXP := 0, 0
	for index, definition := range definitions {
		if definition.SortOrder != index+1 {
			t.Fatalf("sort order at %d = %d", index, definition.SortOrder)
		}
		if definition.Secret {
			secret++
		}
		totalXP += definition.XP
	}
	if secret != 10 {
		t.Fatalf("secret definitions = %d, want 10", secret)
	}
	if totalXP != 9300 {
		t.Fatalf("total XP = %d, want 9300", totalXP)
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
