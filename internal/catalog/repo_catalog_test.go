package catalog_test

import (
	"os"
	"path/filepath"

	"github.com/hpcsc/feature-compose/internal/catalog"
)

var repoCatalogDir = func() string {
	dir, err := filepath.Abs("../../catalog")
	if err != nil {
		panic(err)
	}
	return dir
}()

func repoCatalog() catalog.Catalog {
	return catalog.New(os.DirFS(repoCatalogDir))
}
