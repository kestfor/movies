package main

import (
	"bytes"
	"fmt"
	"os"

	"movies/backend/internal/usecase/achievements"
)

func main() {
	var output bytes.Buffer
	output.WriteString("# Каталог ачивок КиноКруга\n\n")
	output.WriteString("Файл сгенерирован из `catalog.go`. Ручные изменения будут перезаписаны.\n\n")
	output.WriteString("| # | Код | Название | Условие | Метрика | Порог | XP | Политика | Секретная |\n")
	output.WriteString("| ---: | --- | --- | --- | --- | ---: | ---: | --- | :---: |\n")
	for _, definition := range achievements.Definitions() {
		secret := "нет"
		if definition.Secret {
			secret = "да"
		}
		fmt.Fprintf(&output, "| %d | `%s` | %s | %s | `%s` | %d | %d | `%s` | %s |\n",
			definition.SortOrder, definition.Code, definition.Title, definition.Description,
			definition.Metric, definition.Target, definition.XP, definition.AwardPolicy, secret)
	}
	if err := os.WriteFile("CATALOG.md", output.Bytes(), 0o644); err != nil {
		panic(err)
	}
}
