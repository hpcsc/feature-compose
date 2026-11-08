package catalog

import (
	"fmt"
	"io/fs"
	"maps"
	"path"
	"path/filepath"
	"slices"
	"strings"
)

type template struct {
	path         string
	folder       folder
	pathInFolder string
	contents     string
}

func readTemplates(fsys fs.FS, root string, folders map[string]folder) ([]template, error) {
	var templates []template
	err := fs.WalkDir(fsys, root, func(name string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		catalogPath := strings.TrimPrefix(name, root+"/")
		folderName, pathInFolder, _ := strings.Cut(catalogPath, "/")
		declared, ok := folders[folderName]
		if !ok || pathInFolder == "" {
			return fmt.Errorf("template %s is not in a folder of the starter: the folders are %s",
				catalogPath, strings.Join(slices.Sorted(maps.Keys(folders)), ", "))
		}
		contents, err := fs.ReadFile(fsys, name)
		if err != nil {
			return err
		}
		templates = append(templates, template{path: catalogPath, folder: declared, pathInFolder: pathInFolder, contents: string(contents)})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("read the templates under %s: %w", root, err)
	}
	return templates, nil
}

func (t template) render(s Selection) (file, error) {
	renderedPath, err := s.render(t.path, t.pathInFolder)
	if err != nil {
		return file{}, err
	}
	repoPath := path.Join(t.folder.path, renderedPath)
	if !filepath.IsLocal(filepath.FromSlash(repoPath)) {
		return file{}, fmt.Errorf("template %s renders to %s, which is outside the output folder", t.path, repoPath)
	}
	contents, err := s.render(t.path, t.contents)
	if err != nil {
		return file{}, err
	}
	return file{path: repoPath, contents: contents}, nil
}
