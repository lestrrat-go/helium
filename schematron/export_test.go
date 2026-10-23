package schematron

import (
	"context"

	"github.com/lestrrat-go/helium"
)

func NodePathsForTesting(ctx context.Context, nodes []helium.Node) ([]string, error) {
	cache := make(nodePathCache)
	paths := make([]string, 0, len(nodes))
	for _, node := range nodes {
		path, err := cache.path(ctx, node)
		if err != nil {
			return nil, err
		}
		paths = append(paths, path)
	}
	return paths, nil
}
