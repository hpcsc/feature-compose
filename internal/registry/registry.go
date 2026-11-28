package registry

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/hpcsc/feature-compose/internal/catalog"
	"github.com/hpcsc/feature-compose/internal/config"
)

type Change struct {
	Path *string
	Ref  *string
}

type Synced struct {
	Name      string
	Local     bool
	OldCommit string
	NewCommit string
	Err       error
}

func (s Synced) String() string {
	switch {
	case s.Err != nil:
		return s.Name + ": FAIL"
	case s.Local:
		return s.Name + ": a local folder, so sync gets nothing"
	case s.OldCommit == "":
		return s.Name + ": cloned again, at " + ShortCommit(s.NewCommit)
	case s.OldCommit == s.NewCommit:
		return s.Name + ": no new commit, at " + ShortCommit(s.NewCommit)
	}
	return s.Name + ": from " + ShortCommit(s.OldCommit) + " to " + ShortCommit(s.NewCommit)
}

type Instance struct {
	file   config.File
	clones string
}

func Open() (*Instance, error) {
	file, err := config.Load()
	if err != nil {
		return nil, err
	}
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("find the folder of the clones: %w", err)
		}
		dataHome = filepath.Join(home, ".local", "share")
	}
	return &Instance{file: file, clones: filepath.Join(dataHome, "feature-compose", "catalogs")}, nil
}

func (r *Instance) Catalogs(ctx context.Context) (catalog.Catalogs, error) {
	names := r.file.Names()
	if len(names) == 0 {
		return catalog.Catalogs{}, fmt.Errorf("the config file %s has no catalog: add one with feature-compose catalog add <name> <source>", r.file.Path())
	}
	var all []catalog.Catalog
	for _, name := range names {
		c, err := r.Catalog(ctx, name)
		if err != nil {
			return catalog.Catalogs{}, err
		}
		all = append(all, c)
	}
	return catalog.NewCatalogs(all...), nil
}

func (r *Instance) Catalog(ctx context.Context, name string) (catalog.Catalog, error) {
	entry, err := r.file.Entry(name)
	if err != nil {
		return catalog.Catalog{}, err
	}
	root, commit, err := r.root(ctx, name, entry)
	if err != nil {
		return catalog.Catalog{}, err
	}
	return catalog.FromOrigin(os.DirFS(filepath.Join(root, filepath.FromSlash(entry.Path))), r.origin(name, entry, commit)), nil
}

func (r *Instance) Folder(name string) (string, error) {
	if name == "" {
		names := r.file.Names()
		switch len(names) {
		case 0:
			return "", fmt.Errorf("the config file %s has no catalog: give the folder of the catalog with --catalog <folder>", r.file.Path())
		case 1:
			name = names[0]
		default:
			return "", fmt.Errorf("the config file %s has more than one catalog: give one with --catalog <name>, one of %s", r.file.Path(), strings.Join(names, ", "))
		}
	}
	entry, err := r.file.Entry(name)
	if err != nil {
		return "", err
	}
	if entry.IsGitURL() {
		return "", fmt.Errorf("the catalog %s is a clone of %s, and the next catalog sync removes each change in the clone: give --catalog <folder> with a clone of your own", name, entry.SourceWithoutCredentials())
	}
	folder, err := r.file.SourceFolder(entry)
	if err != nil {
		return "", err
	}
	return filepath.Join(folder, filepath.FromSlash(entry.Path)), nil
}

func (r *Instance) List(ctx context.Context) ([]catalog.Origin, error) {
	var origins []catalog.Origin
	for _, name := range r.file.Names() {
		entry, err := r.file.Entry(name)
		if err != nil {
			return nil, err
		}
		commit := ""
		if c := r.cloneOf(name, entry); entry.IsGitURL() && c.exists() {
			if commit, err = c.commit(ctx); err != nil {
				return nil, err
			}
		}
		origins = append(origins, r.origin(name, entry, commit))
	}
	return origins, nil
}

func (r *Instance) Refs(ctx context.Context, name string) ([]string, error) {
	entry, err := r.file.Entry(name)
	if err != nil {
		return nil, err
	}
	if !entry.IsGitURL() {
		return nil, fmt.Errorf("the catalog %s is a local folder, so it has no ref", name)
	}
	c := r.cloneOf(name, entry)
	if !c.exists() {
		return nil, r.noClone(name)
	}
	return c.refs(ctx)
}

func (r *Instance) Add(ctx context.Context, name string, entry config.Entry, steps catalog.Steps) (catalog.Origin, error) {
	var origin catalog.Origin
	err := r.changing(func() error {
		var err error
		origin, err = r.add(ctx, name, entry, steps)
		return err
	})
	return origin, err
}

func (r *Instance) Sync(ctx context.Context, names []string, steps catalog.Steps) ([]Synced, error) {
	var synced []Synced
	err := r.changing(func() error {
		if len(names) == 0 {
			names = r.file.Names()
		}
		for _, name := range names {
			if _, err := r.file.Entry(name); err != nil {
				return err
			}
		}
		for _, name := range names {
			entry, _ := r.file.Entry(name)
			synced = append(synced, r.synced(ctx, name, entry, steps))
		}
		return nil
	})
	return synced, err
}

func (r *Instance) Set(ctx context.Context, name string, change Change, steps catalog.Steps) (catalog.Origin, error) {
	var origin catalog.Origin
	err := r.changing(func() error {
		var err error
		origin, err = r.set(ctx, name, change, steps)
		return err
	})
	return origin, err
}

func (r *Instance) Remove(name string) error {
	return r.changing(func() error { return r.remove(name) })
}

func (r *Instance) changing(change func() error) error {
	if err := os.MkdirAll(r.clones, 0o755); err != nil {
		return fmt.Errorf("make the folder of the clones: %w", err)
	}
	lock, err := os.OpenFile(filepath.Join(r.clones, ".lock"), os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return fmt.Errorf("open the lock of the catalogs: %w", err)
	}
	defer lock.Close()
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return errors.New("another feature-compose catalog command changes the catalogs now: run this command again when it ends")
	}
	defer func() { _ = syscall.Flock(int(lock.Fd()), syscall.LOCK_UN) }()
	if r.file, err = config.Load(); err != nil {
		return err
	}
	return change()
}

func (r *Instance) add(ctx context.Context, name string, entry config.Entry, steps catalog.Steps) (catalog.Origin, error) {
	if _, err := r.file.Entry(name); err == nil {
		return catalog.Origin{}, fmt.Errorf("the config already has the catalog %s: change it with feature-compose catalog set, or remove it with feature-compose catalog remove", name)
	}
	entry, err := entry.WithAbsoluteSource()
	if err != nil {
		return catalog.Origin{}, err
	}
	changed, err := r.file.WithCatalog(name, entry)
	if err != nil {
		return catalog.Origin{}, err
	}
	if entry, err = changed.Entry(name); err != nil {
		return catalog.Origin{}, err
	}
	commit := ""
	if entry.IsGitURL() {
		c, err := newClone(ctx, entry, r.cloneFolder(name), steps)
		if err != nil {
			return catalog.Origin{}, err
		}
		if commit, err = r.checkedOut(ctx, c, entry); err != nil {
			_ = os.RemoveAll(c.folder)
			return catalog.Origin{}, err
		}
	} else if err := r.checkFolder(entry); err != nil {
		return catalog.Origin{}, err
	}
	if err := changed.Save(); err != nil {
		if entry.IsGitURL() {
			_ = os.RemoveAll(r.cloneFolder(name))
		}
		return catalog.Origin{}, err
	}
	r.file = changed
	return r.origin(name, entry, commit), nil
}

func (r *Instance) set(ctx context.Context, name string, change Change, steps catalog.Steps) (catalog.Origin, error) {
	entry, err := r.file.Entry(name)
	if err != nil {
		return catalog.Origin{}, err
	}
	if change.Path != nil {
		entry.Path = *change.Path
	}
	if change.Ref != nil {
		entry.Ref = *change.Ref
	}
	file, err := r.file.WithCatalog(name, entry)
	if err != nil {
		return catalog.Origin{}, err
	}
	if entry, err = file.Entry(name); err != nil {
		return catalog.Origin{}, err
	}
	if !entry.IsGitURL() {
		if err := r.checkFolder(entry); err != nil {
			return catalog.Origin{}, err
		}
		if err := file.Save(); err != nil {
			return catalog.Origin{}, err
		}
		r.file = file
		return r.origin(name, entry, ""), nil
	}
	c := r.cloneOf(name, entry)
	if !c.exists() {
		return catalog.Origin{}, r.noClone(name)
	}
	old, err := c.commit(ctx)
	if err != nil {
		return catalog.Origin{}, err
	}
	if change.Ref != nil {
		if err := c.fetch(ctx, steps); err != nil {
			return catalog.Origin{}, err
		}
		if err := c.checkout(ctx, entry.Ref); err != nil {
			return catalog.Origin{}, err
		}
	}
	commit, err := r.checkedPath(ctx, c, entry)
	if err == nil {
		err = file.Save()
	}
	if err != nil {
		_ = c.checkout(ctx, old)
		return catalog.Origin{}, err
	}
	r.file = file
	return r.origin(name, entry, commit), nil
}

func (r *Instance) remove(name string) error {
	entry, err := r.file.Entry(name)
	if err != nil {
		return err
	}
	file := r.file.WithoutCatalog(name)
	if err := file.Save(); err != nil {
		return err
	}
	r.file = file
	if !entry.IsGitURL() {
		return nil
	}
	if err := os.RemoveAll(r.cloneFolder(name)); err != nil {
		return fmt.Errorf("remove the clone of the catalog %s: %w", name, err)
	}
	return nil
}

func (r *Instance) synced(ctx context.Context, name string, entry config.Entry, steps catalog.Steps) Synced {
	if !entry.IsGitURL() {
		return Synced{Name: name, Local: true}
	}
	s := Synced{Name: name}
	c := r.cloneOf(name, entry)
	if !c.exists() {
		if c, s.Err = newClone(ctx, entry, c.folder, steps); s.Err != nil {
			return s
		}
		if s.NewCommit, s.Err = r.checkedOut(ctx, c, entry); s.Err != nil {
			s.NewCommit = ""
			_ = os.RemoveAll(c.folder)
		}
		return s
	}
	if s.OldCommit, s.Err = c.commit(ctx); s.Err != nil {
		return s
	}
	if s.Err = c.fetch(ctx, steps); s.Err != nil {
		return s
	}
	if s.NewCommit, s.Err = r.checkedOut(ctx, c, entry); s.Err != nil {
		s.NewCommit = ""
		_ = c.checkout(ctx, s.OldCommit)
	}
	return s
}

func (r *Instance) checkedOut(ctx context.Context, c clone, entry config.Entry) (string, error) {
	if err := c.checkout(ctx, entry.Ref); err != nil {
		return "", err
	}
	return r.checkedPath(ctx, c, entry)
}

func (r *Instance) checkedPath(ctx context.Context, c clone, entry config.Entry) (string, error) {
	if err := entry.CheckCatalogIn(c.folder); err != nil {
		return "", err
	}
	return c.commit(ctx)
}

func (r *Instance) checkFolder(entry config.Entry) error {
	if entry.Ref != "" {
		return fmt.Errorf("the source %s is a local folder, so it has no ref: give --ref only with a git URL", entry.Source)
	}
	folder, err := r.file.SourceFolder(entry)
	if err != nil {
		return err
	}
	if info, err := os.Stat(folder); err != nil || !info.IsDir() {
		return fmt.Errorf("the source %s is not a git URL and not a folder", entry.Source)
	}
	return entry.CheckCatalogIn(folder)
}

func (r *Instance) root(ctx context.Context, name string, entry config.Entry) (string, string, error) {
	if entry.IsGitURL() {
		c := r.cloneOf(name, entry)
		if !c.exists() {
			return "", "", r.noClone(name)
		}
		commit, err := c.commit(ctx)
		return c.folder, commit, err
	}
	folder, err := r.file.SourceFolder(entry)
	if err != nil {
		return "", "", err
	}
	if info, err := os.Stat(folder); err != nil || !info.IsDir() {
		return "", "", fmt.Errorf("the folder %s of the catalog %s is not a folder", folder, name)
	}
	return folder, "", nil
}

func (r *Instance) noClone(name string) error {
	return fmt.Errorf("the catalog %s has no clone in %s: run feature-compose catalog sync %s", name, r.cloneFolder(name), name)
}

func (r *Instance) origin(name string, entry config.Entry, commit string) catalog.Origin {
	return catalog.Origin{Name: name, Source: entry.SourceWithoutCredentials(), Path: entry.Path, Ref: entry.Ref, Commit: commit}
}

func (r *Instance) cloneOf(name string, entry config.Entry) clone {
	return clone{source: entry.Source, sourceWithoutCredentials: entry.SourceWithoutCredentials(), folder: r.cloneFolder(name)}
}

func (r *Instance) cloneFolder(name string) string {
	return filepath.Join(r.clones, name)
}

func ShortCommit(commit string) string {
	return commit[:min(len(commit), 7)]
}
