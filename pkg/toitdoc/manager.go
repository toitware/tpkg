package toitdoc

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/toitware/toit.git/tools/tpkg/config"
	"github.com/toitware/toit.git/tools/tpkg/pkg/tpkg"
	"go.uber.org/zap"
)

type manager struct {
	sync.RWMutex

	logger    *zap.Logger
	manager   *tpkg.Manager
	generator *generator
	cfg       config.Toitdocs
	ui        tpkg.UI

	toitdocs map[pkgIdentifier]*toitdoc
	loading  map[pkgIdentifier]*loader
}

func provideManager(logger *zap.Logger, tpkgManager *tpkg.Manager, generator *generator, cfg *config.Config, ui tpkg.UI) (*manager, Manager, error) {
	res := &manager{
		logger:    logger,
		manager:   tpkgManager,
		generator: generator,
		cfg:       cfg.Toitdocs,
		ui:        ui,
		toitdocs:  map[pkgIdentifier]*toitdoc{},
		loading:   map[pkgIdentifier]*loader{},
	}
	return res, res, nil
}

type Manager interface {
	Get(ctx context.Context, desc *tpkg.Desc) (Doc, error)
}

type pkgIdentifier struct {
	Version string
	URL     string
}

func descIdentifier(desc *tpkg.Desc) pkgIdentifier {
	return pkgIdentifier{
		Version: desc.Version,
		URL:     desc.URL,
	}
}

type Doc interface {
	JSONPath() string
	ViewerIndexPath() string
}

const (
	toitdocPath     = "toitdoc.json"
	viewerIndexPath = "viewer_index.html"
)

type toitdoc struct {
	desc *tpkg.Desc
	path string
}

func (t *toitdoc) JSONPath() string {
	return filepath.Join(t.path, toitdocPath)
}
func (t *toitdoc) ViewerIndexPath() string {
	return filepath.Join(t.path, viewerIndexPath)
}

var _ Doc = (*toitdoc)(nil)

type loader struct {
	sync.Mutex

	ctx    context.Context
	cancel context.CancelFunc

	signal chan struct{}
	err    error

	doc *toitdoc
}

func newLoader() *loader {
	ctx, cancel := context.WithCancel(context.Background())
	return &loader{
		ctx:    ctx,
		cancel: cancel,
		signal: make(chan struct{}),
	}
}

func (l *loader) Wait(ctx context.Context) (*toitdoc, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-l.signal:
		return l.doc, l.err
	case <-l.ctx.Done():
		if l.err == nil && l.doc == nil {
			return nil, l.ctx.Err()
		}
		return l.doc, l.err
	}
}

func (l *loader) start(desc *tpkg.Desc, mgr *manager) (doc *toitdoc, err error) {
	defer func() {
		mgr.storeResult(desc, doc)
		l.close(doc, err)
	}()

	path := filepath.Join(mgr.cfg.CachePath, tpkg.URLVersionToRelPath(desc.URL, desc.Version))
	// If the directory already exists, we just reuse it.
	if stat, err := os.Stat(path); err == nil && stat.IsDir() {
		return &toitdoc{desc: desc, path: path}, nil
	}

	tmpDir, err := ioutil.TempDir("", "pkg-*")
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			os.RemoveAll(tmpDir)
		}
	}()

	repoDir := filepath.Join(tmpDir, "repo")
	if _, err := tpkg.DownloadGit(l.ctx, repoDir, desc.URL, desc.Version, desc.Hash, mgr.ui); err != nil {
		return nil, err
	}

	// Download dependent packages.
	projectPaths, err := tpkg.NewProjectPaths(repoDir, "", "")
	if err != nil {
		return nil, err
	}

	projectManager := tpkg.NewProjectPkgManager(mgr.manager, projectPaths)
	if err := projectManager.Install(l.ctx, false); err != nil {
		return nil, err
	}

	// Generate toitdoc JSON file.
	jsonPath := filepath.Join(tmpDir, toitdocPath)
	if err := mgr.generator.generateDocs(l.ctx, repoDir, desc, jsonPath); err != nil {
		return nil, err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, err
	}

	// Generate index.html file for the toitdocs viewer.
	viewerIndexTmplPath := filepath.Join(mgr.cfg.ViewerPath, "index.html")
	stat, err := os.Stat(viewerIndexTmplPath)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadFile(viewerIndexTmplPath)
	if err != nil {
		return nil, err
	}

	body, err = rewriteIndexFile(desc, body)
	if err != nil {
		return nil, err
	}

	viewerIndexPath := filepath.Join(tmpDir, viewerIndexPath)
	if err := ioutil.WriteFile(viewerIndexPath, body, stat.Mode()); err != nil {
		return nil, err
	}

	// Move file to toitdocs staging.
	if err := moveDirectory(tmpDir, path); err != nil {
		return nil, err
	}

	return &toitdoc{desc: desc, path: path}, nil
}

func (l *loader) close(doc *toitdoc, err error) {
	l.Lock()
	defer l.Unlock()
	select {
	case <-l.signal:
		return
	default:
	}
	l.doc = doc
	l.err = err
	close(l.signal)
	l.cancel()
}

func (m *manager) Get(ctx context.Context, desc *tpkg.Desc) (Doc, error) {
	ident := descIdentifier(desc)
	m.RLock()
	doc, ok := m.toitdocs[ident]
	if ok {
		m.RUnlock()
		return doc, nil
	}

	loader, ok := m.loading[ident]
	m.RUnlock()
	if ok {
		return loader.Wait(ctx)
	}
	return m.load(ctx, desc, ident)
}

func (m *manager) load(ctx context.Context, desc *tpkg.Desc, ident pkgIdentifier) (*toitdoc, error) {
	m.Lock()
	loader, ok := m.loading[ident]
	if ok {
		m.Unlock()
		return loader.Wait(ctx)
	}

	loader = newLoader()
	m.loading[ident] = loader
	m.Unlock()

	go loader.start(desc, m)
	return loader.Wait(ctx)
}

func (m *manager) storeResult(desc *tpkg.Desc, doc *toitdoc) {
	m.Lock()
	defer m.Unlock()

	ident := descIdentifier(desc)
	if doc != nil {
		m.toitdocs[ident] = doc
	}
	delete(m.loading, ident)
}

func moveDirectory(from, to string) (err error) {
	if err := os.MkdirAll(filepath.Dir(to), 0755); err != nil {
		return err
	}
	defer func() {
		if err != nil {
			os.RemoveAll(to)
		}
	}()

	if err := os.Rename(from, to); err == nil {
		return nil
	}

	files, err := ioutil.ReadDir(from)
	if err != nil {
		return err
	}
	for _, info := range files {
		if err != nil {
			return err
		}
		newPath := strings.Replace(info.Name(), from, to, 1)
		if info.IsDir() {
			return os.MkdirAll(newPath, 0755)
		} else {
			return os.Rename(info.Name(), newPath)
		}
	}
	return nil
}

type htmlVariable struct {
	regexp *regexp.Regexp
	value  string
}

func (m *htmlVariable) Patch(b []byte) []byte {
	matches := m.regexp.FindAllSubmatchIndex(b, -1)
	for _, match := range matches {
		if len(match) != 4 {
			continue
		}
		b = append(b[:match[2]], append([]byte(m.value), b[match[3]:]...)...)
	}
	return b
}

func createHTMLVariables(pairs ...string) ([]*htmlVariable, error) {
	var res []*htmlVariable
	for i := 0; i < len(pairs); i++ {
		regexp, err := regexp.Compile(pairs[i])
		if err != nil {
			return nil, err
		}
		i++
		value := pairs[i]
		res = append(res, &htmlVariable{
			regexp: regexp,
			value:  value,
		})
	}
	return res, nil
}

func rewriteIndexFile(desc *tpkg.Desc, body []byte) ([]byte, error) {
	vars, err := createHTMLVariables(
		`<base[^\>]+href="([^"]*)">`, "/"+desc.URL+"@"+desc.Version+"/docs/",
		`<meta name="toitdoc-root-library"[^\>]+content="([^"]*)"[^\>]*/>`, "src",
		`<meta name="toitdoc-path"[^\>]+content="([^"]*)"[^\>]*/>`, "/toitdocs/",
		`<meta name="toitdoc-mode"[^\>]+content="([^"]*)"[^\>]*/>`, "package",
		`<meta name="toitdoc-package-name"[^\>]+content="([^"]*)"[^\>]*/>`, desc.Name,
	)
	if err != nil {
		return nil, err
	}

	for _, v := range vars {
		body = v.Patch(body)
	}
	return body, nil
}
