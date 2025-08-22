// Manage Port Forwarding rules (API v8+): /fw/redir/
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
)

// Ensure interfaces
var (
	_ resource.Resource                = &portForwardResource{}
	_ resource.ResourceWithConfigure   = &portForwardResource{}
	_ resource.ResourceWithImportState = &portForwardResource{}
)

func NewPortForwardingResource() resource.Resource { return &portForwardResource{} }

// ---------- API models ----------

type apiPortForward struct {
	ID           int             `json:"id,omitempty"` // omitempty so create doesn't send 0
	Enabled      bool            `json:"enabled"`
	IpProto      string          `json:"ip_proto"`       // "tcp" | "udp"
	WanPortStart int             `json:"wan_port_start"` // required
	WanPortEnd   int             `json:"wan_port_end"`   // required
	LanIP        string          `json:"lan_ip"`         // required
	LanPort      int             `json:"lan_port"`       // required
	SrcIP        string          `json:"src_ip"`         // default "0.0.0.0"
	Comment      string          `json:"comment,omitempty"`
	Hostname     string          `json:"hostname,omitempty"` // read-only
	Host         json.RawMessage `json:"host,omitempty"`     // read-only
}

// NOTE: apiEnvelope[T] is defined elsewhere in this package.

// ---------- TF model ----------

type pfModel struct {
	ID           types.Int64  `tfsdk:"id"`
	Enabled      types.Bool   `tfsdk:"enabled"`
	IpProto      types.String `tfsdk:"ip_proto"`
	WanPortStart types.Int64  `tfsdk:"wan_port_start"`
	WanPortEnd   types.Int64  `tfsdk:"wan_port_end"`
	LanIP        types.String `tfsdk:"lan_ip"`
	LanPort      types.Int64  `tfsdk:"lan_port"`
	SrcIP        types.String `tfsdk:"src_ip"`
	Comment      types.String `tfsdk:"comment"`
	Hostname     types.String `tfsdk:"hostname"`
}

type portForwardResource struct{ client *Client }

// ---------- Resource wiring ----------

func (r *portForwardResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "freebox_port_forward"
}

func (r *portForwardResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = rschema.Schema{
		Description: "Manage Freebox Port Forwarding rules (fw/redir).",
		Attributes: map[string]rschema.Attribute{
			"id": rschema.Int64Attribute{
				Computed:    true,
				Description: "Port forwarding rule ID (assigned by Freebox).",
				// We keep Update robust by reading ID from state; no plan modifier required.
			},
			"enabled": rschema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Enable/disable this forwarding rule.",
			},
			"ip_proto": rschema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("tcp"),
				Description: `IP protocol ("tcp" or "udp").`,
			},
			"wan_port_start": rschema.Int64Attribute{
				Required:    true,
				Description: "External (WAN) start port.",
			},
			"wan_port_end": rschema.Int64Attribute{
				Required:    true,
				Description: "External (WAN) end port.",
			},
			"lan_ip": rschema.StringAttribute{
				Required:    true,
				Description: "Target LAN IP.",
			},
			"lan_port": rschema.Int64Attribute{
				Required:    true,
				Description: "Target LAN start port (end is lan_port + wan_port_end - wan_port_start).",
			},
			"src_ip": rschema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("0.0.0.0"),
				Description: "Source IP filter. Use 0.0.0.0 for any source.",
			},
			"comment": rschema.StringAttribute{
				Optional:    true,
				Description: "Optional comment.",
			},
			"hostname": rschema.StringAttribute{
				Computed:    true,
				Description: "Resolved target hostname (read-only).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *portForwardResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData != nil {
		r.client = req.ProviderData.(*Client)
	}
}

// ---------- CRUD ----------

func (r *portForwardResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Client not configured", "Provider client is nil")
		return
	}

	var plan pfModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload := apiPortForward{
		Enabled:      plan.Enabled.ValueBool(),
		IpProto:      plan.IpProto.ValueString(),
		WanPortStart: int(plan.WanPortStart.ValueInt64()),
		WanPortEnd:   int(plan.WanPortEnd.ValueInt64()),
		LanIP:        plan.LanIP.ValueString(),
		LanPort:      int(plan.LanPort.ValueInt64()),
		SrcIP:        plan.SrcIP.ValueString(),
		Comment:      plan.Comment.ValueString(),
	}
	b, _ := json.Marshal(payload)

	hreq, _ := r.client.newRequest(ctx, http.MethodPost, "/fw/redir/", bytes.NewBuffer(b))
	hres, err := r.client.http.Do(hreq)
	if err != nil {
		resp.Diagnostics.AddError("API error", err.Error())
		return
	}
	defer hres.Body.Close()

	var env apiEnvelope[apiPortForward]
	_ = json.NewDecoder(hres.Body).Decode(&env)
	if hres.StatusCode != http.StatusOK && hres.StatusCode != http.StatusCreated {
		resp.Diagnostics.AddError("API error", fmt.Sprintf("status %d (msg=%s, code=%s)", hres.StatusCode, env.Msg, env.ErrorCode))
		return
	}
	if !env.Success {
		resp.Diagnostics.AddError("API error", fmt.Sprintf("success=false (msg=%s, code=%s)", env.Msg, env.ErrorCode))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, toPFState(env.Result))...)
}

func (r *portForwardResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Client not configured", "Provider client is nil")
		return
	}

	var state pfModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id := state.ID.ValueInt64()
	if id == 0 {
		// nothing to read
		resp.State.RemoveResource(ctx)
		return
	}

	path := fmt.Sprintf("/fw/redir/%d", id)
	hreq, _ := r.client.newRequest(ctx, http.MethodGet, path, nil)
	hres, err := r.client.http.Do(hreq)
	if err != nil {
		resp.Diagnostics.AddError("API error", err.Error())
		return
	}
	defer hres.Body.Close()

	if hres.StatusCode == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	}
	if hres.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(hres.Body)
		resp.Diagnostics.AddError("API error", fmt.Sprintf("status %d: %s", hres.StatusCode, string(body)))
		return
	}

	var env apiEnvelope[apiPortForward]
	if err := json.NewDecoder(hres.Body).Decode(&env); err != nil {
		resp.Diagnostics.AddError("Decode error", err.Error())
		return
	}
	if !env.Success {
		resp.Diagnostics.AddError("API error", env.Msg)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, toPFState(env.Result))...)
}

func (r *portForwardResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Client not configured", "Provider client is nil")
		return
	}

	var plan pfModel
	var state pfModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Prefer ID from current state; fall back to plan if needed
	id := state.ID.ValueInt64()
	if id == 0 {
		id = plan.ID.ValueInt64()
	}
	if id == 0 {
		resp.Diagnostics.AddError("Invalid state", "Missing ID in state")
		return
	}

	// Include ID in payload so it matches the URL (the API validates this)
	payload := apiPortForward{
		ID:           int(id),
		Enabled:      plan.Enabled.ValueBool(),
		IpProto:      plan.IpProto.ValueString(),
		WanPortStart: int(plan.WanPortStart.ValueInt64()),
		WanPortEnd:   int(plan.WanPortEnd.ValueInt64()),
		LanIP:        plan.LanIP.ValueString(),
		LanPort:      int(plan.LanPort.ValueInt64()),
		SrcIP:        plan.SrcIP.ValueString(),
		Comment:      plan.Comment.ValueString(),
	}
	b, _ := json.Marshal(payload)

	path := fmt.Sprintf("/fw/redir/%d", id)
	hreq, _ := r.client.newRequest(ctx, http.MethodPut, path, bytes.NewBuffer(b))
	hres, err := r.client.http.Do(hreq)
	if err != nil {
		resp.Diagnostics.AddError("API error", err.Error())
		return
	}
	defer hres.Body.Close()

	var env apiEnvelope[apiPortForward]
	_ = json.NewDecoder(hres.Body).Decode(&env)
	if hres.StatusCode != http.StatusOK || !env.Success {
		resp.Diagnostics.AddError("API error", fmt.Sprintf("update failed: status %d, msg=%s, code=%s", hres.StatusCode, env.Msg, env.ErrorCode))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, toPFState(env.Result))...)
}

func (r *portForwardResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Client not configured", "Provider client is nil")
		return
	}

	var id types.Int64
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &id)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if id.IsNull() || id.ValueInt64() == 0 {
		return
	}

	path := fmt.Sprintf("/fw/redir/%d", id.ValueInt64())
	hreq, _ := r.client.newRequest(ctx, http.MethodDelete, path, nil)
	hres, err := r.client.http.Do(hreq)
	if err != nil {
		resp.Diagnostics.AddError("API error", err.Error())
		return
	}
	defer hres.Body.Close()

	// Freebox typically returns 200 OK; accept 204 too
	if hres.StatusCode != http.StatusOK && hres.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(hres.Body)
		resp.Diagnostics.AddError("API error", fmt.Sprintf("status %d: %s", hres.StatusCode, string(body)))
		return
	}
}

// Import by id
func (r *portForwardResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

// ---------- helpers ----------

func toPFState(p apiPortForward) *pfModel {
	return &pfModel{
		ID:           types.Int64Value(int64(p.ID)),
		Enabled:      types.BoolValue(p.Enabled),
		IpProto:      types.StringValue(p.IpProto),
		WanPortStart: types.Int64Value(int64(p.WanPortStart)),
		WanPortEnd:   types.Int64Value(int64(p.WanPortEnd)),
		LanIP:        types.StringValue(p.LanIP),
		LanPort:      types.Int64Value(int64(p.LanPort)),
		SrcIP:        stringOrNull(p.SrcIP),
		Comment:      stringOrNull(p.Comment),
		Hostname:     stringOrNull(p.Hostname),
	}
}
