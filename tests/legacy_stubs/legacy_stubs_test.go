package legacy_stubs

import (
	"encoding/xml"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/harald-mue/xsd-parser-go/internal/pipeline"
)

func oldRoot() string {
	if root := os.Getenv("XSD_LEGACY_STUB_ROOT"); root != "" {
		return root
	}
	return filepath.Join("..", "..", "old")
}

func stubPath(t *testing.T, parts ...string) string {
	t.Helper()
	p := filepath.Join(append([]string{oldRoot()}, parts...)...)
	if _, err := os.Stat(p); err != nil {
		t.Skipf("legacy stub not available at %s: %v", p, err)
	}
	return p
}

func targetNamespace(t *testing.T, path string) string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open schema %s: %v", path, err)
	}
	defer f.Close()
	dec := xml.NewDecoder(f)
	for {
		tok, err := dec.Token()
		if err != nil {
			t.Fatalf("read schema %s: %v", path, err)
		}
		start, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		for _, attr := range start.Attr {
			if attr.Name.Local == "targetNamespace" {
				return attr.Value
			}
		}
		t.Fatalf("schema %s has no targetNamespace", path)
	}
}

func TestDeviceMonitorStubGeneratesWithAutoNamespacePackages(t *testing.T) {
	root := stubPath(t, "ap-dcm-device-monitor-control-stub", "target")
	dir := t.TempDir()
	module := "tmp/legacydevicemonitor"
	if err := pipeline.GenerateWithOptions([]string{
		filepath.Join(root, "DeviceCommonTypes.xsd"),
		filepath.Join(root, "DeviceControlMessages.xsd"),
		filepath.Join(root, "DeviceMonitorMessages.xsd"),
	}, pipeline.GenerateOptions{OutDir: dir, ModulePath: module, AutoNamespacePackages: true}); err != nil {
		t.Fatalf("GenerateWithOptions returned error: %v", err)
	}
	checks := map[string][]string{
		filepath.Join("data", "models.go"): {
			"`xml:\"name,attr\"`",
			"type IncidentType string",
			"ErrorFlagsReinstrequired ErrorFlags = \"ReinstRequired\"",
		},
		filepath.Join("monitor", "models.go"): {
			"data.Incident",
			"data.Configuration",
		},
		filepath.Join("control", "models.go"): {
			"data.IncidentCollection",
			"data.String512",
			"type SoftwareVersionChanged = SkidataSoftwareVersionChanged",
		},
	}
	assertGeneratedContains(t, dir, checks)
	assertBuilds(t, dir, module)
}

func TestWebRTCStubGeneratesWithAutoNamespacePackages(t *testing.T) {
	root := stubPath(t, "ap-scon-webrtc-stub", "target", "xsd", "scon", "webrtc", "v1")
	dir := t.TempDir()
	module := "tmp/legacywebrtc"
	exceptionSchema := filepath.Join(root, "Exception.xsd")
	exceptionNS := targetNamespace(t, exceptionSchema)
	if err := pipeline.GenerateWithOptions([]string{
		filepath.Join(root, "Command.xsd"),
		filepath.Join(root, "Announcement.xsd"),
		filepath.Join(root, "Reply.xsd"),
	}, pipeline.GenerateOptions{OutDir: dir, ModulePath: module, AutoNamespacePackages: true}); err != nil {
		t.Fatalf("GenerateWithOptions returned error: %v", err)
	}
	announcement, err := os.ReadFile(filepath.Join(dir, "announcement", "models.go"))
	if err != nil {
		t.Fatalf("read announcement model: %v", err)
	}
	text := string(announcement)
	wantWireQName := "`xml:\"" + exceptionNS + " webRTCException\"`"
	if !strings.Contains(text, wantWireQName) {
		t.Fatalf("generated announcement model missing wire QName tag %s", wantWireQName)
	}
	for _, want := range []string{
		"type WebRtccommunicationRequested = WebRTCCommunicationRequested",
		"type AdditionalIcecandidate = AdditionalICECandidate",
		"type WebRtccommunicationFailure = WebRTCCommunicationFailure",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("generated announcement model missing legacy alias %q", want)
		}
	}
	assertBuilds(t, dir, module)
}

func TestVTPMessagingStubGeneratesWithAutoNamespacePackages(t *testing.T) {
	root := stubPath(t, "ap-vtp-messaging-stub", "target", "api", "common", "messaging", "v1")
	dir := t.TempDir()
	module := "tmp/legacyvtp"
	if err := pipeline.GenerateWithOptions([]string{
		filepath.Join(root, "Command.xsd"),
		filepath.Join(root, "Reply.xsd"),
		filepath.Join(root, "Notification.xsd"),
	}, pipeline.GenerateOptions{OutDir: dir, ModulePath: module, AutoNamespacePackages: true}); err != nil {
		t.Fatalf("GenerateWithOptions returned error: %v", err)
	}
	assertGeneratedContains(t, dir, map[string][]string{
		filepath.Join("datatypes", "models.go"): {"type Rgbcolor = RGBColor"},
	})
	assertBuilds(t, dir, module)
}

func TestDiagnosticStubGeneratesWithAutoNamespacePackages(t *testing.T) {
	root := stubPath(t, "ap-dcm-diagnostic-monitor-control-stub", "target", "api")
	dir := t.TempDir()
	module := "tmp/legacydiagnostic"
	commandSchema := filepath.Join(root, "DiagnosticCommand.xsd")
	notificationSchema := filepath.Join(root, "DiagnosticNotification.xsd")
	if err := pipeline.GenerateWithOptions([]string{
		commandSchema,
		notificationSchema,
	}, pipeline.GenerateOptions{OutDir: dir, ModulePath: module, AutoNamespacePackages: true}); err != nil {
		t.Fatalf("GenerateWithOptions returned error: %v", err)
	}
	checks := map[string][]string{
		filepath.Join("command", "models.go"): {
			"type LogSpaceConfig interface",
			"data.LogFileSizeConfig",
			"type GetTraceDevices GetTraceDevicesType",
		},
		filepath.Join("notification", "models.go"): {
			"data.NonEmptyString256",
			"type TraceFilesUploaded TraceFilesUploadedType",
		},
		filepath.Join("data", "models.go"): {
			"type Device struct",
			"type NonEmptyString256 NonEmptyString",
		},
	}
	assertGeneratedContains(t, dir, checks)
	if out, err := run(dir, "go", "mod", "init", module); err != nil {
		t.Fatalf("go mod init: %v\n%s", err, out)
	}
	if out, err := run(dir, "go", "build", "./..."); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
}

func TestMultiProtocolStubGeneratesWithAutoNamespacePackages(t *testing.T) {
	apiRoot := stubPath(t, "ap-dcm-interface-stub", "target", "api")
	discoveryDir := t.TempDir()
	upgradeDir := t.TempDir()
	discoveryModule := "tmp/legacymultidiscovery"
	upgradeModule := "tmp/legacymultiupgrade"

	if err := pipeline.GenerateWithOptions([]string{
		filepath.Join(apiRoot, "discovery", "v6", "DiscoveryDataMessages.xsd"),
	}, pipeline.GenerateOptions{OutDir: discoveryDir, ModulePath: discoveryModule, AutoNamespacePackages: true}); err != nil {
		t.Fatalf("discovery GenerateWithOptions returned error: %v", err)
	}
	if err := pipeline.GenerateWithOptions([]string{
		filepath.Join(apiRoot, "softwareupgrade", "v1", "SoftwareUpgradeDefinition.xsd"),
		filepath.Join(apiRoot, "softwareupgrade", "v1", "SoftwareUpgradeCommand.xsd"),
		filepath.Join(apiRoot, "softwareupgrade", "v1", "SoftwareUpgradeServiceCommand.xsd"),
		filepath.Join(apiRoot, "softwareupgrade", "v1", "SoftwareUpgradeNotification.xsd"),
	}, pipeline.GenerateOptions{OutDir: upgradeDir, ModulePath: upgradeModule, AutoNamespacePackages: true}); err != nil {
		t.Fatalf("upgrade GenerateWithOptions returned error: %v", err)
	}

	assertGeneratedContains(t, discoveryDir, map[string][]string{
		filepath.Join("data", "models.go"): {
			"TimeZoneEnumEtcGMT12",
			`TimeZoneEnum = "Etc/GMT+12"`,
		},
	})
	assertGeneratedContains(t, upgradeDir, map[string][]string{
		filepath.Join("data", "models.go"): {
			"type Checksum interface",
			"type TargetSystems interface",
		},
		filepath.Join("definition", "models.go"): {
			"type Artifact =",
			"v := data.",
		},
		filepath.Join("command", "models.go"): {
			"Artifact          data.",
		},
		filepath.Join("notification", "models.go"): {
			"type DeploymentIdentifier interface",
			"DeploymentIdentifierRegistry",
			"xsi:type",
			"IsDeploymentIdentifier()",
		},
	})
	assertBuilds(t, discoveryDir, discoveryModule)
	assertBuilds(t, upgradeDir, upgradeModule)
}

func TestGenericIntercomStubGeneratesWithAutoNamespacePackages(t *testing.T) {
	root := stubPath(t, "ap-dcm-generic-intercom-monitor-control-stub", "target", "wsdl")
	dir := t.TempDir()
	module := "tmp/legacyintercom"
	dataSchema := filepath.Join(root, "GenericIntercomCommonTypes.xsd")
	controlSchema := filepath.Join(root, "GenericIntercomControlMessages.xsd")
	monitorSchema := filepath.Join(root, "GenericIntercomMonitorMessages.xsd")
	if err := pipeline.GenerateWithOptions([]string{
		dataSchema,
		controlSchema,
		monitorSchema,
	}, pipeline.GenerateOptions{OutDir: dir, ModulePath: module, AutoNamespacePackages: true}); err != nil {
		t.Fatalf("GenerateWithOptions returned error: %v", err)
	}
	checks := map[string][]string{
		filepath.Join("control", "models.go"): {
			"data.IntercomException",
			"type CallCenterCreateCallRequest CallCenterCreateCallRequestType",
		},
		filepath.Join("monitor", "models.go"): {
			"data.Device",
			"type NotifyProviderListenerDeviceStateChanged NotifyProviderListenerDeviceStateChangedType",
		},
		filepath.Join("data", "models.go"): {
			"type Device struct",
			"type IntercomException struct",
			"DeviceTypeDesktopstation DeviceType = \"DesktopStation\"",
		},
	}
	assertGeneratedContains(t, dir, checks)
	if out, err := run(dir, "go", "mod", "init", module); err != nil {
		t.Fatalf("go mod init: %v\n%s", err, out)
	}
	if out, err := run(dir, "go", "build", "./..."); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
}

func TestReducedFixturesMatchLegacyPatterns(t *testing.T) {
	cases := []struct {
		name        string
		schema      string
		module      string
		packageName string
		pkgs        map[string]string
		contain     []string
	}{
		{
			name:        "imported_type_element",
			schema:      filepath.Join("..", "fixtures", "imported_type_element", "schema.xsd"),
			module:      "tmp/legacy/imported",
			packageName: "monitor",
			pkgs: map[string]string{
				"urn:legacy:data":    "data",
				"urn:legacy:monitor": "monitor",
			},
			contain: []string{"data.Incident"},
		},
		{
			name:        "global_element_type_qname",
			schema:      filepath.Join("..", "fixtures", "global_element_type_qname", "schema.xsd"),
			module:      "tmp/legacy/qname",
			packageName: "announcement",
			pkgs: map[string]string{
				"urn:legacy:announcement": "announcement",
				"urn:legacy:exception":    "exception",
			},
			contain: []string{"exception.WebRTCException", "`xml:\"urn:legacy:exception webRTCException\"`"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := pipeline.GenerateWithOptions([]string{tc.schema}, pipeline.GenerateOptions{
				Package:           tc.packageName,
				OutDir:            dir,
				ModulePath:        tc.module,
				NamespacePackages: tc.pkgs,
			}); err != nil {
				t.Fatalf("GenerateWithOptions returned error: %v", err)
			}
			src, err := os.ReadFile(filepath.Join(dir, tc.packageName, "models.go"))
			if err != nil {
				t.Fatalf("read generated model: %v", err)
			}
			text := string(src)
			for _, want := range tc.contain {
				if !strings.Contains(text, want) {
					t.Fatalf("generated model missing %q", want)
				}
			}
		})
	}
}

func assertGeneratedContains(t *testing.T, dir string, checks map[string][]string) {
	t.Helper()
	for rel, wants := range checks {
		data, err := os.ReadFile(filepath.Join(dir, rel))
		if err != nil {
			t.Fatalf("read generated model %s: %v", rel, err)
		}
		text := string(data)
		for _, want := range wants {
			if !strings.Contains(text, want) {
				t.Fatalf("generated model %s missing %q", rel, want)
			}
		}
	}
}

func assertBuilds(t *testing.T, dir string, module string) {
	t.Helper()
	if out, err := run(dir, "go", "mod", "init", module); err != nil {
		t.Fatalf("go mod init: %v\n%s", err, out)
	}
	if out, err := run(dir, "go", "build", "./..."); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
}

func run(dir string, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return string(out), err
}
