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
	catalogPath  string
	owner        string
	featureName  string
	folder       folder
	pathInFolder string
	contents     string
	delimiters   delimiters
	entry        string
	when         expression
}

func readTemplates(fsys fs.FS, root, featureName string) ([]template, error) {
	owner := "the starter"
	if featureName != "" {
		owner = "the feature " + featureName
	}
	var templates []template
	err := fs.WalkDir(fsys, root, func(name string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		contents, err := fs.ReadFile(fsys, name)
		if err != nil {
			return err
		}
		templates = append(templates, template{catalogPath: strings.TrimPrefix(name, root+"/"), owner: owner, featureName: featureName, contents: string(contents)})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("read the templates of %s under %s: %w", owner, root, err)
	}
	return templates, nil
}

func (t template) place(folders map[string]folder, rules []delimitersRule) (template, error) {
	folderName, inFolder := t.folderIn(folders)
	if !inFolder {
		return template{}, fmt.Errorf("the template %s of %s is not in a folder of the starter: the folders are %s",
			t.catalogPath, t.owner, strings.Join(slices.Sorted(maps.Keys(folders)), ", "))
	}
	_, pathInFolder := t.splitPath()
	t.folder, t.pathInFolder, t.delimiters = folders[folderName], pathInFolder, defaultDelimiters
	for _, rule := range rules {
		if rule.matches(t.catalogPath) {
			t.delimiters = rule.delimiters
			break
		}
	}
	return t, nil
}

func (t template) folderIn(folders map[string]folder) (string, bool) {
	folderName, pathInFolder := t.splitPath()
	if pathInFolder == "" {
		return "", false
	}
	_, ok := folders[folderName]
	return folderName, ok
}

func (t template) splitPath() (folderName, pathInFolder string) {
	folderName, pathInFolder, _ = strings.Cut(t.catalogPath, "/")
	return folderName, strings.TrimSuffix(pathInFolder, ".tmpl")
}

func (t template) description() string {
	if t.entry == "" {
		return "the template " + t.catalogPath + " of " + t.owner
	}
	return "the template " + t.catalogPath + " of the files entry " + t.entry + " of " + t.owner
}

func (t template) render(s Selection) (file, error) {
	label := t.catalogPath + " of " + t.owner
	renderedPath, err := s.render(label, t.pathInFolder, defaultDelimiters, t.featureName)
	if err != nil {
		return file{}, err
	}
	repoPath := path.Join(t.folder.path, renderedPath)
	if !filepath.IsLocal(filepath.FromSlash(repoPath)) {
		return file{}, fmt.Errorf("the template %s of %s renders to %s, which is outside the output folder", t.catalogPath, t.owner, repoPath)
	}
	contents, err := s.render(label, t.contents, t.delimiters, t.featureName)
	if err != nil {
		return file{}, err
	}
	return file{path: repoPath, contents: contents, owner: t.owner}, nil
}
