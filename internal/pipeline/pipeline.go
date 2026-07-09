package pipeline

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/harald-mue/xsd-parser-go/internal/binding"
	"github.com/harald-mue/xsd-parser-go/internal/generator"
	"github.com/harald-mue/xsd-parser-go/internal/interpreter"
	"github.com/harald-mue/xsd-parser-go/internal/optimizer"
	"github.com/harald-mue/xsd-parser-go/internal/parser"
	"github.com/harald-mue/xsd-parser-go/internal/renderer"
	"github.com/harald-mue/xsd-parser-go/internal/schema"
)

// Inspect parses schemas and returns aggregate statistics.
func Inspect(paths []string) (schema.Stats, error) {
	schemas, err := parser.ParseFiles(paths)
	if err != nil {
		return schema.Stats{}, err
	}
	return schemas.Stats(), nil
}

// GenerateOptions controls pipeline generation.
type GenerateOptions struct {
	Package               string
	OutDir                string
	ModulePath            string
	NamespacePackages     map[string]string
	AutoNamespacePackages bool
	NamespacePrefixes     map[string]string
	RuntimePackage        string
	OutputFile            string
	ValidateOnUnmarshal   bool
	OptimizerSerde        bool
	Catalogs              []string
	BindingFiles          []string
	TypeMappings          map[string]string
}

// Generate parses, interprets, optimizes, lowers, renders, and writes Go files.
func Generate(paths []string, packageName string, outDir string) error {
	return GenerateWithOptions(paths, GenerateOptions{Package: packageName, OutDir: outDir})
}

// GenerateWithOptions parses, interprets, optimizes, lowers, renders, and writes Go files.
func GenerateWithOptions(paths []string, options GenerateOptions) error {
	genOptions, err := generatorOptionsFromGenerate(options)
	if err != nil {
		return err
	}
	files, err := RenderWithOptions(paths, genOptions)
	if err != nil {
		return err
	}
	outDir := options.OutDir
	if outDir == "" {
		outDir = "generated/model"
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("create output directory %s: %w", outDir, err)
	}
	for name, src := range files {
		path := filepath.Join(outDir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return fmt.Errorf("create output directory for %s: %w", path, err)
		}
		if err := os.WriteFile(path, src, 0o644); err != nil {
			return fmt.Errorf("write generated file %s: %w", path, err)
		}
	}
	return nil
}

// Render runs the full pipeline and returns rendered file contents keyed by
// file name, without writing to disk.
func Render(paths []string, packageName string) (map[string][]byte, error) {
	return RenderWithOptions(paths, generator.Options{Package: packageName})
}

// RenderWithGenerateOptions runs the full pipeline using generation options.
func RenderWithGenerateOptions(paths []string, options GenerateOptions) (map[string][]byte, error) {
	genOptions, err := generatorOptionsFromGenerate(options)
	if err != nil {
		return nil, err
	}
	return RenderWithOptions(paths, genOptions)
}

// RenderWithOptions runs the full pipeline and returns rendered file contents keyed by
// file name, without writing to disk.
func RenderWithOptions(paths []string, options generator.Options) (map[string][]byte, error) {
	schemas, err := parser.ParseFilesWithOptions(paths, parser.Options{Catalogs: options.Catalogs})
	if err != nil {
		return nil, err
	}
	if options.AutoNamespacePackages {
		options.NamespacePackages = autoNamespacePackages(schemas, options.NamespacePackages)
	}
	meta, err := interpreter.Interpret(schemas)
	if err != nil {
		return nil, err
	}
	meta, err = optimizer.OptimizeWithFlags(meta, optimizerFlags(options))
	if err != nil {
		return nil, err
	}
	pkg, err := generator.GenerateWithOptions(meta, options)
	if err != nil {
		return nil, err
	}
	return renderer.RenderPackage(pkg)
}

func generatorOptionsFromGenerate(options GenerateOptions) (generator.Options, error) {
	bindings, err := binding.LoadFiles(options.BindingFiles)
	if err != nil {
		return generator.Options{}, err
	}
	return generator.Options{
		Package:               options.Package,
		ModulePath:            options.ModulePath,
		NamespacePackages:     options.NamespacePackages,
		AutoNamespacePackages: options.AutoNamespacePackages,
		NamespacePrefixes:     options.NamespacePrefixes,
		RuntimePackage:        options.RuntimePackage,
		OutputFile:            options.OutputFile,
		ValidateOnUnmarshal:   options.ValidateOnUnmarshal,
		OptimizerSerde:        options.OptimizerSerde,
		Catalogs:              options.Catalogs,
		TypeMappings:          options.TypeMappings,
		TypeNameOverrides:     bindings.TypeNameOverrides,
		FieldNameOverrides:    bindings.FieldNameOverrides,
	}, nil
}

func optimizerFlags(options generator.Options) optimizer.Flags {
	flags := optimizer.DefaultFlags()
	if options.OptimizerSerde {
		flags |= optimizer.FlagSerde
	}
	return flags
}

func autoNamespacePackages(schemas schema.Schemas, explicit map[string]string) map[string]string {
	result := make(map[string]string, len(explicit)+len(schemas.Files))
	for ns, pkg := range explicit {
		if ns != "" && pkg != "" {
			result[ns] = pkg
		}
	}
	for _, file := range schemas.Files {
		ns := file.TargetNamespace
		if ns == "" || result[ns] != "" {
			continue
		}
		result[ns] = packageNameFromNamespace(ns)
	}
	return result
}

func packageNameFromNamespace(namespace string) string {
	trimmed := strings.TrimRight(namespace, "/#:")
	idx := strings.LastIndexAny(trimmed, "/#:")
	last := trimmed
	if idx >= 0 && idx+1 < len(trimmed) {
		last = trimmed[idx+1:]
	}
	name := sanitizePackageName(last)
	if name == "" {
		name = "model"
	}
	if goKeywords[name] {
		name += "pkg"
	}
	return name
}

func sanitizePackageName(value string) string {
	var b strings.Builder
	for i := 0; i < len(value); i++ {
		ch := value[i]
		if ch >= 'A' && ch <= 'Z' {
			ch += 'a' - 'A'
		}
		if (ch >= 'a' && ch <= 'z') || ch == '_' || (ch >= '0' && ch <= '9' && b.Len() > 0) {
			b.WriteByte(ch)
		}
	}
	return b.String()
}

var goKeywords = map[string]bool{
	"break": true, "default": true, "func": true, "interface": true, "select": true,
	"case": true, "defer": true, "go": true, "map": true, "struct": true,
	"chan": true, "else": true, "goto": true, "package": true, "switch": true,
	"const": true, "fallthrough": true, "if": true, "range": true, "type": true,
	"continue": true, "for": true, "import": true, "return": true, "var": true,
}
