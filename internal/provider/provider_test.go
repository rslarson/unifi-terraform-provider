package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestProviderMetadata(t *testing.T) {
	p := New("1.0.0")()
	resp := &provider.MetadataResponse{}
	p.Metadata(context.Background(), provider.MetadataRequest{}, resp)

	if resp.TypeName != "unifi" {
		t.Errorf("expected type name 'unifi', got %q", resp.TypeName)
	}
	if resp.Version != "1.0.0" {
		t.Errorf("expected version '1.0.0', got %q", resp.Version)
	}
}

func TestProviderSchema(t *testing.T) {
	p := New("test")()
	resp := &provider.SchemaResponse{}
	p.Schema(context.Background(), provider.SchemaRequest{}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected schema errors: %v", resp.Diagnostics)
	}

	schema := resp.Schema

	// Verify api_key attribute exists and is sensitive
	apiKeyAttr, ok := schema.Attributes["api_key"]
	if !ok {
		t.Fatal("expected 'api_key' attribute in schema")
	}
	if !apiKeyAttr.IsSensitive() {
		t.Error("expected 'api_key' to be sensitive")
	}
	if !apiKeyAttr.IsOptional() {
		t.Error("expected 'api_key' to be optional")
	}

	// Verify host_id attribute exists
	hostIDAttr, ok := schema.Attributes["host_id"]
	if !ok {
		t.Fatal("expected 'host_id' attribute in schema")
	}
	if !hostIDAttr.IsOptional() {
		t.Error("expected 'host_id' to be optional")
	}
}

func testProviderConfig(t *testing.T) tfsdk.Config {
	t.Helper()
	p := New("test")()
	schemaResp := &provider.SchemaResponse{}
	p.Schema(context.Background(), provider.SchemaRequest{}, schemaResp)

	rawVal, err := schemaResp.Schema.Type().ValueFromTerraform(context.Background(),
		tftypes.NewValue(schemaResp.Schema.Type().TerraformType(context.Background()), map[string]tftypes.Value{
			"api_key": tftypes.NewValue(tftypes.String, nil),
			"host_id": tftypes.NewValue(tftypes.String, nil),
		}),
	)
	if err != nil {
		t.Fatalf("failed to create config value: %v", err)
	}

	raw, err2 := rawVal.ToTerraformValue(context.Background())
	if err2 != nil {
		t.Fatalf("failed to convert to terraform value: %v", err2)
	}

	return tfsdk.Config{
		Schema: schemaResp.Schema,
		Raw:    raw,
	}
}

func TestProviderConfigureMissingAPIKey(t *testing.T) {
	t.Setenv("UNIFI_API_KEY", "")
	t.Setenv("UNIFI_HOST_ID", "")

	p := &UnifiProvider{version: "test"}
	resp := &provider.ConfigureResponse{}
	p.Configure(context.Background(), provider.ConfigureRequest{
		Config: testProviderConfig(t),
	}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for missing API key")
	}

	found := false
	for _, d := range resp.Diagnostics.Errors() {
		if d.Summary() == "Missing API Key" {
			found = true
		}
	}
	if !found {
		t.Error("expected 'Missing API Key' error")
	}
}

func TestProviderConfigureMissingHostID(t *testing.T) {
	t.Setenv("UNIFI_API_KEY", "test-key")
	t.Setenv("UNIFI_HOST_ID", "")

	p := &UnifiProvider{version: "test"}
	resp := &provider.ConfigureResponse{}
	p.Configure(context.Background(), provider.ConfigureRequest{
		Config: testProviderConfig(t),
	}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for missing host ID")
	}

	found := false
	for _, d := range resp.Diagnostics.Errors() {
		if d.Summary() == "Missing Host ID" {
			found = true
		}
	}
	if !found {
		t.Error("expected 'Missing Host ID' error")
	}
}

func TestProviderConfigureSuccess(t *testing.T) {
	t.Setenv("UNIFI_API_KEY", "test-key")
	t.Setenv("UNIFI_HOST_ID", "test-host")

	p := &UnifiProvider{version: "test"}
	resp := &provider.ConfigureResponse{}
	p.Configure(context.Background(), provider.ConfigureRequest{
		Config: testProviderConfig(t),
	}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
	if resp.DataSourceData == nil {
		t.Error("expected DataSourceData to be set")
	}
	if resp.ResourceData == nil {
		t.Error("expected ResourceData to be set")
	}
}

func TestProviderResources(t *testing.T) {
	p := New("test")()
	resources := p.(*UnifiProvider).Resources(context.Background())

	expectedCount := 8 // network, wifi_broadcast, firewall_zone, acl_rule, firewall_policy, traffic_matching_list, dns_policy, hotspot_voucher
	if len(resources) != expectedCount {
		t.Errorf("expected %d resources, got %d", expectedCount, len(resources))
	}
}

func TestProviderDataSources(t *testing.T) {
	p := New("test")()
	dataSources := p.(*UnifiProvider).DataSources(context.Background())

	expectedCount := 18 // network, wifi_broadcast, firewall_zone, device, sites, acl_rule, firewall_policy, traffic_matching_list, dns_policy, clients, wans, vpn_servers, vpn_tunnels, radius_profiles, device_tags, pending_devices, dpi_categories, dpi_applications
	if len(dataSources) != expectedCount {
		t.Errorf("expected %d data sources, got %d", expectedCount, len(dataSources))
	}
}
