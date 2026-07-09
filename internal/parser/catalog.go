package parser

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Options controls schema parsing and resolution.
type Options struct {
	Catalogs []string
}

type catalogResolver struct {
	exact    map[string]string
	rewrites []catalogRewrite
}

type catalogRewrite struct {
	prefix      string
	replacement string
}

func loadCatalogs(paths []string) (catalogResolver, error) {
	resolver := catalogResolver{exact: make(map[string]string)}
	for _, path := range paths {
		if path == "" {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return resolver, fmt.Errorf("read catalog %s: %w", path, err)
		}
		baseDir := filepath.Dir(path)
		var catalog xmlCatalog
		if err := xml.Unmarshal(data, &catalog); err != nil {
			return resolver, fmt.Errorf("parse catalog %s: %w", path, err)
		}
		for _, entry := range catalog.Entries {
			switch entry.XMLName.Local {
			case "uri":
				if entry.Name != "" && entry.URI != "" {
					resolver.exact[entry.Name] = resolveCatalogPath(baseDir, entry.URI)
				}
			case "system":
				if entry.SystemID != "" && entry.URI != "" {
					resolver.exact[entry.SystemID] = resolveCatalogPath(baseDir, entry.URI)
				}
			case "rewriteURI":
				if entry.URIStart != "" && entry.RewritePrefix != "" {
					resolver.rewrites = append(resolver.rewrites, catalogRewrite{prefix: entry.URIStart, replacement: resolveCatalogPath(baseDir, entry.RewritePrefix)})
				}
			case "rewriteSystem":
				if entry.SystemIDStart != "" && entry.RewritePrefix != "" {
					resolver.rewrites = append(resolver.rewrites, catalogRewrite{prefix: entry.SystemIDStart, replacement: resolveCatalogPath(baseDir, entry.RewritePrefix)})
				}
			}
		}
	}
	return resolver, nil
}

type xmlCatalog struct {
	Entries []xmlCatalogEntry `xml:",any"`
}

type xmlCatalogEntry struct {
	XMLName       xml.Name
	Name          string `xml:"name,attr"`
	URI           string `xml:"uri,attr"`
	SystemID      string `xml:"systemId,attr"`
	URIStart      string `xml:"uriStartString,attr"`
	SystemIDStart string `xml:"systemIdStartString,attr"`
	RewritePrefix string `xml:"rewritePrefix,attr"`
}

func resolveCatalogPath(baseDir, value string) string {
	if value == "" || strings.Contains(value, "://") || filepath.IsAbs(value) {
		return value
	}
	return filepath.Clean(filepath.Join(baseDir, value))
}

func (r catalogResolver) resolve(location string) (string, bool) {
	if location == "" {
		return "", false
	}
	if target := r.exact[location]; target != "" {
		return target, true
	}
	for _, rewrite := range r.rewrites {
		if strings.HasPrefix(location, rewrite.prefix) {
			return filepath.Clean(filepath.Join(rewrite.replacement, strings.TrimPrefix(location, rewrite.prefix))), true
		}
	}
	return "", false
}
