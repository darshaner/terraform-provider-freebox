// Manage the DHCP server configuration (singleton) — API v8
package freebox

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &dhcpConfigResource{}
	_ resource.ResourceWithConfigure   = &dhcpConfigResource{}
	_ resource.ResourceWithImportState = &dhcpConfigResource{}
)

func NewDhcpConfigResource() resource.Resource { return &dhcpConfigResource{} }

type dhcpConfigResource struct{ client *Client }

type apiDhcpConfig struct {
	Enabled              bool     `json:"enabled"`
	StickyAssign         bool     `json:"sticky_assign"`
	Gateway              string   `json:"gateway"` // read-only
	Netmask              string   `json:"netmask"` // read-only
	IPRangeStart         string   `json:"ip_range_start"`
	IPRangeEnd           string   `json:"ip_range_end"`
	AlwaysBroadcast      bool     `json:"always_broadcast"`
	IgnoreOutOfRangeHint bool     `json:"ignore_out_of_range_hint"`
	DNS                  []string `json:"dns"`
}

type envCfg[T any] struct {
	Success   bool   `json:"success"`
	Result    T      `json:"result"`
	Msg       string `json:"msg"`
	ErrorCode string `json:"error_code"`
}

type dhcpConfigModel struct {
	Id                   types.String   `tfsdk:"id"`
	Enabled              types.Bool     `tfsdk:"enabled"`
	StickyAssign         types.Bool     `tfsdk:"sticky_assign"`
	Gateway              types.String   `tfsdk:"gateway"`
	Netmask              types.String   `tfsdk:"netmask"`
	IpRangeStart         types.String   `tfsdk:"ip_range_start"`
	IpRangeEnd           types.String   `tfsdk:"ip_range_end"`
	AlwaysBroadcast      types.Bool     `tfsdk:"always_broadcast"`
	IgnoreOutOfRangeHint types.Bool     `tfsdk:"ignore_out_of_range_hint"`
	Dns                  []types.String `tfsdk:"dns"`
}

func (r *dhcpConfigResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "freebox_dhcp_config"
}

func (r *dhcpConfigResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = rschema.Schema{
		Description: "Manage Freebox DHCP server configuration (API v8). Singleton resource.",
		Attributes: map[string]rschema.Attribute{
			"id": rschema.StringAttribute{Computed: true, Description: "Synthetic ID.", PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},

			// Writable
			"enabled":                  rschema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false), Description: "Enable/Disable DHCP server."},
			"sticky_assign":            rschema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(true), Description: "Always assign same IP to a host."},
			"ip_range_start":           rschema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString(""), Description: "DHCP range start IP."},
			"ip_range_end":             rschema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString(""), Description: "DHCP range end IP."},
			"always_broadcast":         rschema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false), Description: "Always broadcast DHCP responses."},
			"ignore_out_of_range_hint": rschema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false), Description: "Ignore requested address if outside DHCP range."},
			"dns":                      rschema.ListAttribute{Optional: true, Computed: true, ElementType: types.StringType, Description: "DNS servers to include in replies (Freebox returns 5 items)."},

			// Read-only
			"gateway": rschema.StringAttribute{Computed: true, Description: "Gateway IP (read-only).", PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"netmask": rschema.StringAttribute{Computed: true, Description: "Gateway netmask (read-only).", PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
		},
	}
}

func (r *dhcpConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData != nil {
		r.client = req.ProviderData.(*Client)
	}
}

func (r *dhcpConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Client not configured", "Provider client is nil")
		return
	}
	var plan dhcpConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload := modelToPayload(plan)
	b, _ := json.Marshal(payload)
	hreq, _ := r.client.newRequest(ctx, http.MethodPut, "/dhcp/config/", bytes.NewBuffer(b))
	hres, err := r.client.http.Do(hreq)
	if err != nil {
		resp.Diagnostics.AddError("API error", err.Error())
		return
	}
	defer hres.Body.Close()

	var env envCfg[apiDhcpConfig]
	_ = json.NewDecoder(hres.Body).Decode(&env)
	if hres.StatusCode != http.StatusOK || !env.Success {
		resp.Diagnostics.AddError("API error", dhcpErrDetail(hres.StatusCode, env))
		return
	}

	state := cfgToModel(env.Result)
	state.Id = types.StringValue("dhcp_config")
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	tflog.Info(ctx, "Applied DHCP config (create)")
}

func (r *dhcpConfigResource) Read(ctx context.Context, _ resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Client not configured", "Provider client is nil")
		return
	}
	hreq, _ := r.client.newRequest(ctx, http.MethodGet, "/dhcp/config/", nil)
	hres, err := r.client.http.Do(hreq)
	if err != nil {
		resp.Diagnostics.AddError("API error", err.Error())
		return
	}
	defer hres.Body.Close()
	if hres.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(hres.Body)
		resp.Diagnostics.AddError("API error", fmt.Sprintf("status %d: %s", hres.StatusCode, string(b)))
		return
	}

	var env envCfg[apiDhcpConfig]
	if err := json.NewDecoder(hres.Body).Decode(&env); err != nil {
		resp.Diagnostics.AddError("Decode error", err.Error())
		return
	}
	if !env.Success {
		resp.Diagnostics.AddError("API error", env.Msg)
		return
	}
	state := cfgToModel(env.Result)
	state.Id = types.StringValue("dhcp_config")
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *dhcpConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Client not configured", "Provider client is nil")
		return
	}
	var plan dhcpConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload := modelToPayload(plan)
	b, _ := json.Marshal(payload)
	preq, _ := r.client.newRequest(ctx, http.MethodPut, "/dhcp/config/", bytes.NewBuffer(b))
	pres, err := r.client.http.Do(preq)
	if err != nil {
		resp.Diagnostics.AddError("API error", err.Error())
		return
	}
	defer pres.Body.Close()

	var env envCfg[apiDhcpConfig]
	_ = json.NewDecoder(pres.Body).Decode(&env)
	if pres.StatusCode != http.StatusOK || !env.Success {
		resp.Diagnostics.AddError("API error", dhcpErrDetail(pres.StatusCode, env))
		return
	}
	state := cfgToModel(env.Result)
	state.Id = types.StringValue("dhcp_config")
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	tflog.Info(ctx, "Applied DHCP config (update)")
}

func (r *dhcpConfigResource) Delete(context.Context, resource.DeleteRequest, *resource.DeleteResponse) {
}

func (r *dhcpConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// helpers
func modelToPayload(m dhcpConfigModel) apiDhcpConfig {
	return apiDhcpConfig{
		Enabled:              m.Enabled.ValueBool(),
		StickyAssign:         m.StickyAssign.ValueBool(),
		IPRangeStart:         m.IpRangeStart.ValueString(),
		IPRangeEnd:           m.IpRangeEnd.ValueString(),
		AlwaysBroadcast:      m.AlwaysBroadcast.ValueBool(),
		IgnoreOutOfRangeHint: m.IgnoreOutOfRangeHint.ValueBool(),
		DNS:                  expandStringList(m.Dns),
	}
}

func cfgToModel(c apiDhcpConfig) dhcpConfigModel {
	return dhcpConfigModel{
		Enabled:              types.BoolValue(c.Enabled),
		StickyAssign:         types.BoolValue(c.StickyAssign),
		Gateway:              stringOrNull(c.Gateway),
		Netmask:              stringOrNull(c.Netmask),
		IpRangeStart:         stringOrNull(c.IPRangeStart),
		IpRangeEnd:           stringOrNull(c.IPRangeEnd),
		AlwaysBroadcast:      types.BoolValue(c.AlwaysBroadcast),
		IgnoreOutOfRangeHint: types.BoolValue(c.IgnoreOutOfRangeHint),
		Dns:                  flattenStringList(c.DNS),
	}
}

func dhcpErrDetail(status int, env any) string {
	type slim struct{ Msg, ErrorCode string }
	b, _ := json.Marshal(env)
	var s slim
	_ = json.Unmarshal(b, &s)
	pretty := map[string]string{
		"inval":              "invalid argument",
		"inval_netmask":      "invalid netmask",
		"inval_ip_range":     "invalid IP range",
		"inval_ip_range_net": "IP range & netmask mismatch",
		"inval_gw_net":       "gateway & netmask mismatch",
		"exist":              "already exists",
		"nodev":              "no such device",
		"noent":              "no such entry",
		"netdown":            "network is down",
		"busy":               "device or resource busy",
	}
	if human, ok := pretty[s.ErrorCode]; ok {
		return fmt.Sprintf("status %d: %s (error_code=%s: %s)", status, s.Msg, s.ErrorCode, human)
	}
	return fmt.Sprintf("status %d: %s (error_code=%s)", status, s.Msg, s.ErrorCode)
}
