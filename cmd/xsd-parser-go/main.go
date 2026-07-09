package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/harald-mue/xsd-parser-go/internal/pipeline"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		usage()
		return nil
	}

	switch args[0] {
	case "inspect":
		return inspect(args[1:])
	case "generate":
		return generate(args[1:])
	case "convert":
		return convert(args[1:])
	case "help", "-h", "--help":
		usage()
		return nil
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func inspect(args []string) error {
	fs := flag.NewFlagSet("inspect", flag.ContinueOnError)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return fmt.Errorf("inspect requires at least one XSD file")
	}

	stats, err := pipeline.Inspect(fs.Args())
	if err != nil {
		return err
	}

	fmt.Printf("Schemas:       %d\n", stats.Files)
	fmt.Printf("Namespaces:    %d\n", len(stats.Namespaces))
	fmt.Printf("Elements:      %d\n", stats.Elements)
	fmt.Printf("Complex types: %d\n", stats.ComplexTypes)
	fmt.Printf("Simple types:  %d\n", stats.SimpleTypes)
	return nil
}

func generate(args []string) error {
	fs := flag.NewFlagSet("generate", flag.ContinueOnError)
	packageName := fs.String("package", "model", "default Go package name")
	outDir := fs.String("out", "generated/model", "output directory")
	modulePath := fs.String("module", "", "Go module import path for generated namespace packages")
	runtimePackage := fs.String("runtime-package", "", "optional generated shared runtime package name")
	outputFile := fs.String("output-file", "", "generated Go file name; defaults to models.go")
	validateOnUnmarshal := fs.Bool("validate-on-unmarshal", false, "call Validate after generated UnmarshalXML methods")
	optimizerSerde := fs.Bool("optimizer-serde", false, "enable serde-oriented optimizer passes (flatten unions, merge enum unions)")
	var namespacePackages keyValueFlags
	var namespacePrefixes keyValueFlags
	var typeMappings keyValueFlags
	var catalogs stringSliceFlags
	var bindings stringSliceFlags
	fs.Var(&namespacePackages, "namespace-package", "namespace-to-package mapping as URI=package; may be repeated")
	fs.Var(&namespacePrefixes, "namespace-prefix", "namespace-to-XML-prefix mapping as URI=prefix; may be repeated")
	fs.Var(&typeMappings, "type-map", "XSD type to Go type mapping as QName=GoType; examples: xs:dateTime=time.Time or urn#Type=example.com/pkg.Type")
	fs.Var(&catalogs, "catalog", "OASIS XML catalog file; may be repeated")
	fs.Var(&bindings, "binding", "JAXB .xjb binding file subset; may be repeated")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return fmt.Errorf("generate requires at least one XSD file")
	}
	return pipeline.GenerateWithOptions(fs.Args(), pipeline.GenerateOptions{
		Package:             *packageName,
		OutDir:              *outDir,
		ModulePath:          *modulePath,
		NamespacePackages:   namespacePackages.values,
		NamespacePrefixes:   namespacePrefixes.values,
		RuntimePackage:      *runtimePackage,
		OutputFile:          *outputFile,
		ValidateOnUnmarshal: *validateOnUnmarshal,
		OptimizerSerde:      *optimizerSerde,
		Catalogs:            catalogs.values,
		BindingFiles:        bindings.values,
		TypeMappings:        typeMappings.values,
	})
}

func convert(args []string) error {
	options := pipeline.GenerateOptions{OutputFile: "models.go", AutoNamespacePackages: true}
	var positionals []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "--") {
			positionals = append(positionals, arg)
			continue
		}
		name, value, hasValue := splitFlag(arg)
		requireValue := func() (string, error) {
			if hasValue {
				return value, nil
			}
			if i+1 >= len(args) {
				return "", fmt.Errorf("%s requires a value", name)
			}
			i++
			return args[i], nil
		}
		switch name {
		case "--namespace-package":
			v, err := requireValue()
			if err != nil {
				return err
			}
			if options.NamespacePackages == nil {
				options.NamespacePackages = make(map[string]string)
			}
			if err := setMapping(options.NamespacePackages, v); err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
		case "--namespace-prefix":
			v, err := requireValue()
			if err != nil {
				return err
			}
			if options.NamespacePrefixes == nil {
				options.NamespacePrefixes = make(map[string]string)
			}
			if err := setMapping(options.NamespacePrefixes, v); err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
		case "--type-map":
			v, err := requireValue()
			if err != nil {
				return err
			}
			if options.TypeMappings == nil {
				options.TypeMappings = make(map[string]string)
			}
			if err := setMapping(options.TypeMappings, v); err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
		case "--catalog":
			v, err := requireValue()
			if err != nil {
				return err
			}
			options.Catalogs = append(options.Catalogs, v)
		case "--binding":
			v, err := requireValue()
			if err != nil {
				return err
			}
			options.BindingFiles = append(options.BindingFiles, v)
		case "--package":
			v, err := requireValue()
			if err != nil {
				return err
			}
			options.Package = v
		case "--module":
			v, err := requireValue()
			if err != nil {
				return err
			}
			options.ModulePath = v
		case "--out":
			v, err := requireValue()
			if err != nil {
				return err
			}
			options.OutDir = v
		case "--runtime-package":
			v, err := requireValue()
			if err != nil {
				return err
			}
			options.RuntimePackage = v
		case "--output-file":
			v, err := requireValue()
			if err != nil {
				return err
			}
			options.OutputFile = v
		case "--validate-on-unmarshal":
			v := "true"
			if hasValue {
				v = value
			}
			options.ValidateOnUnmarshal = v != "false" && v != "0"
		case "--optimizer-serde":
			v := "true"
			if hasValue {
				v = value
			}
			options.OptimizerSerde = v != "false" && v != "0"
		default:
			return fmt.Errorf("unknown convert flag %s", name)
		}
	}
	if len(positionals) < 3 {
		return fmt.Errorf("convert requires at least one XSD file followed by module path and output directory")
	}
	paths := positionals[:len(positionals)-2]
	if options.ModulePath == "" {
		options.ModulePath = positionals[len(positionals)-2]
	}
	if options.OutDir == "" {
		options.OutDir = positionals[len(positionals)-1]
	}
	return pipeline.GenerateWithOptions(paths, options)
}

func splitFlag(arg string) (name string, value string, hasValue bool) {
	idx := strings.Index(arg, "=")
	if idx < 0 {
		return arg, "", false
	}
	return arg[:idx], arg[idx+1:], true
}

func setMapping(values map[string]string, value string) error {
	idx := strings.LastIndex(value, "=")
	if idx <= 0 || idx == len(value)-1 {
		return fmt.Errorf("mapping must be URI=value")
	}
	values[value[:idx]] = value[idx+1:]
	return nil
}

type stringSliceFlags struct {
	values []string
}

func (f *stringSliceFlags) String() string {
	if f == nil {
		return ""
	}
	return strings.Join(f.values, ",")
}

func (f *stringSliceFlags) Set(value string) error {
	if value == "" {
		return fmt.Errorf("value must not be empty")
	}
	f.values = append(f.values, value)
	return nil
}

type keyValueFlags struct {
	values map[string]string
}

func (f *keyValueFlags) String() string {
	if f == nil || len(f.values) == 0 {
		return ""
	}
	parts := make([]string, 0, len(f.values))
	for ns, pkg := range f.values {
		parts = append(parts, ns+"="+pkg)
	}
	return strings.Join(parts, ",")
}

func (f *keyValueFlags) Set(value string) error {
	idx := strings.LastIndex(value, "=")
	if idx <= 0 || idx == len(value)-1 {
		return fmt.Errorf("mapping must be URI=value")
	}
	if f.values == nil {
		f.values = make(map[string]string)
	}
	f.values[value[:idx]] = value[idx+1:]
	return nil
}

func usage() {
	fmt.Println(`xsd-parser-go

Usage:
  xsd-parser-go inspect <schema.xsd> [...]
  xsd-parser-go generate --package model --out generated/model <schema.xsd> [...]
  xsd-parser-go generate --module example.com/app --namespace-package urn:foo=foo --namespace-package urn:bar=bar <schema.xsd> [...]
  xsd-parser-go convert <schema.xsd> [...] module/path output/dir

Generate flags:
  --package name                 default Go package for generated code
  --out dir                      output directory
  --module path                  Go module path used for cross-package imports
  --namespace-package URI=pkg    map XML namespace URI to a Go package; repeatable
  --namespace-prefix URI=prefix  map XML namespace URI to a deterministic XML prefix; repeatable
  --runtime-package pkg          emit shared XML runtime helpers into a generated package
  --output-file file             generated Go file name; defaults to models.go
  --validate-on-unmarshal        call Validate from generated UnmarshalXML methods where custom code is generated
  --optimizer-serde              enable serde-oriented optimizer passes (flatten unions, merge enum unions)
  --catalog file                 OASIS XML catalog for schemaLocation resolution; repeatable
  --binding file                 JAXB .xjb binding subset for class/property names; repeatable
  --type-map QName=GoType        map an XSD type to a Go type; repeatable

The convert command accepts legacy positional arguments: XSD files, module path, output directory.
It writes models.go by default and auto-detects namespace packages from targetNamespace values.

This is a native Go XSD parser and Go code generator.`)
}
