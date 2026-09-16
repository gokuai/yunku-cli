package registry

import "embed"

var (
	//go:embed personas.yaml
	personasYAML []byte
	//go:embed recipes.yaml
	recipesYAML []byte
	//go:embed personas.yaml recipes.yaml
	_ embed.FS
)

func PersonasYAML() []byte {
	return append([]byte(nil), personasYAML...)
}

func RecipesYAML() []byte {
	return append([]byte(nil), recipesYAML...)
}
