package mermaid

import (
	"context"
	"fmt"
	"strings"
	"sync"

	goldmermaid "go.abhg.dev/goldmark/mermaid"
	"go.abhg.dev/goldmark/mermaid/mermaidcdp"
)

const mermaidJSSourceVersion = "11.12.1"

var (
	cachedJSSource    string
	errCachedJSSource error
	cacheJSSource     sync.Once
)

// SVGRenderer renders Mermaid source to SVG.
type SVGRenderer interface {
	Render(ctx context.Context, source string) ([]byte, error)
	Close() error
}

// Renderer is the default Mermaid renderer backed by mermaidcdp.
type Renderer struct {
	compiler *mermaidcdp.Compiler
}

// NewRenderer creates the default Mermaid renderer.
func NewRenderer() (*Renderer, error) {
	jsSource, err := loadJSSource(context.Background())
	if err != nil {
		return nil, err
	}

	compiler, err := mermaidcdp.New(&mermaidcdp.Config{
		JSSource: jsSource,
	})
	if err != nil {
		return nil, fmt.Errorf("initialize mermaid renderer: %w", err)
	}

	return &Renderer{
		compiler: compiler,
	}, nil
}

// Render converts Mermaid source to SVG bytes.
func (r *Renderer) Render(ctx context.Context, source string) ([]byte, error) {
	if r == nil || r.compiler == nil {
		return nil, fmt.Errorf("renderer is not initialized")
	}

	source = strings.TrimSpace(source)
	if source == "" {
		return nil, fmt.Errorf("mermaid source is empty")
	}

	response, err := r.compiler.Compile(ctx, &goldmermaid.CompileRequest{
		Source: source,
	})
	if err != nil {
		return nil, fmt.Errorf("compile mermaid diagram: %w", err)
	}

	svg := strings.TrimSpace(response.SVG)
	if svg == "" {
		return nil, fmt.Errorf("compile mermaid diagram: empty svg output")
	}
	return []byte(svg), nil
}

// Close releases renderer resources.
func (r *Renderer) Close() error {
	if r == nil || r.compiler == nil {
		return nil
	}
	return r.compiler.Close()
}

func loadJSSource(ctx context.Context) (string, error) {
	cacheJSSource.Do(func() {
		cachedJSSource, errCachedJSSource = mermaidcdp.DownloadJSSource(ctx, mermaidJSSourceVersion)
		if errCachedJSSource != nil {
			errCachedJSSource = fmt.Errorf("download mermaid javascript source: %w", errCachedJSSource)
		}
	})
	return cachedJSSource, errCachedJSSource
}
