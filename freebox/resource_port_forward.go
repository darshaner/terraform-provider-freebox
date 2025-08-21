package freebox

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ----------------------------
// API Model
// ----------------------------
type PortForwardingConfig struct {
	ID           int    `json:"id"`
	Enabled      bool   `json:"enabled"`
	IpProto      string `json:"ip_proto"`
	WanPortStart int    `json:"wan_port_start"`
	WanPortEnd   int    `json:"wan_port_end"`
	LanIP        string `json:"lan_ip"`
	LanPort      int    `json:"lan_port"`
	SrcIP        string `json:"src_ip"`
	Comment      string `json:"comment"`
	Hostname     string `json:"hostname"`
}

// ----------------------------
// Terraform Resource Model
// ----------------------------
type PortForwardingModel struct {
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

// ----------------------------
// Resource Definition
// ----------------------------
type PortForwardingResource struct{}

func NewPortForwardingResource() resource.Resource {
	return &PortForwardingResource{}
}

func (r *PortForwardingResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_port_forward"
}

func (r *PortForwardingResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Computed: true,
			},
			"enabled": schema.BoolAttribute{
				Required: true,
				Default:  booldefault.StaticBool(true),
			},
			"ip_proto": schema.StringAttribute{
				Required: true,
			},
			"wan_port_start": schema.Int64Attribute{
				Required: true,
			},
			"wan_port_end": schema.Int64Attribute{
				Required: true,
			},
			"lan_ip": schema.StringAttribute{
				Required: true,
			},
			"lan_port": schema.Int64Attribute{
				Required: true,
			},
			"src_ip": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"comment": schema.StringAttribute{
				Optional: true,
			},
			"hostname": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

// ----------------------------
// CRUD Operations
// ----------------------------
func (r *PortForwardingResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan PortForwardingModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload := map[string]interface{}{
		"enabled":        plan.Enabled.ValueBool(),
		"ip_proto":       plan.IpProto.ValueString(),
		"wan_port_start": plan.WanPortStart.ValueInt64(),
		"wan_port_end":   plan.WanPortEnd.ValueInt64(),
		"lan_ip":         plan.LanIP.ValueString(),
		"lan_port":       plan.LanPort.ValueInt64(),
		"src_ip":         plan.SrcIP.ValueString(),
		"comment":        plan.Comment.ValueString(),
	}

	body, _ := json.Marshal(payload)
	url := "http://mafreebox.freebox.fr/api/v8/fw/redir/"
	res, err := fbClient.DoRequest(http.MethodPost, url, body)
	if err != nil {
		resp.Diagnostics.AddError("API error", fmt.Sprintf("Failed creating port forward: %s", err))
		return
	}
	defer res.Body.Close()
	raw, _ := ioutil.ReadAll(res.Body)

	var result struct {
		Success bool                 `json:"success"`
		Result  PortForwardingConfig `json:"result"`
	}
	_ = json.Unmarshal(raw, &result)
	if !result.Success {
		resp.Diagnostics.AddError("API error", fmt.Sprintf("Create failed: %s", string(raw)))
		return
	}

	plan.ID = types.Int64Value(int64(result.Result.ID))
	plan.Hostname = types.StringValue(result.Result.Hostname)

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *PortForwardingResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state PortForwardingModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	url := fmt.Sprintf("http://mafreebox.freebox.fr/api/v8/fw/redir/%d", state.ID.ValueInt64())
	res, err := fbClient.DoRequest(http.MethodGet, url, nil)
	if err != nil {
		resp.Diagnostics.AddError("API error", fmt.Sprintf("Failed reading port forward: %s", err))
		return
	}
	defer res.Body.Close()
	raw, _ := ioutil.ReadAll(res.Body)

	var result struct {
		Success bool                 `json:"success"`
		Result  PortForwardingConfig `json:"result"`
	}
	_ = json.Unmarshal(raw, &result)
	if !result.Success {
		// If rule not found, Terraform should drop it from state
		resp.State.RemoveResource(ctx)
		return
	}

	state.Enabled = types.BoolValue(result.Result.Enabled)
	state.IpProto = types.StringValue(result.Result.IpProto)
	state.WanPortStart = types.Int64Value(int64(result.Result.WanPortStart))
	state.WanPortEnd = types.Int64Value(int64(result.Result.WanPortEnd))
	state.LanIP = types.StringValue(result.Result.LanIP)
	state.LanPort = types.Int64Value(int64(result.Result.LanPort))
	state.SrcIP = types.StringValue(result.Result.SrcIP)
	state.Comment = types.StringValue(result.Result.Comment)
	state.Hostname = types.StringValue(result.Result.Hostname)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *PortForwardingResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan PortForwardingModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := plan.ID.ValueInt64()
	url := fmt.Sprintf("http://mafreebox.freebox.fr/api/v8/fw/redir/%d", id)

	payload := map[string]interface{}{
		"enabled":        plan.Enabled.ValueBool(),
		"ip_proto":       plan.IpProto.ValueString(),
		"wan_port_start": plan.WanPortStart.ValueInt64(),
		"wan_port_end":   plan.WanPortEnd.ValueInt64(),
		"lan_ip":         plan.LanIP.ValueString(),
		"lan_port":       plan.LanPort.ValueInt64(),
		"src_ip":         plan.SrcIP.ValueString(),
		"comment":        plan.Comment.ValueString(),
	}

	body, _ := json.Marshal(payload)
	res, err := fbClient.DoRequest(http.MethodPut, url, body)
	if err != nil {
		resp.Diagnostics.AddError("API error", fmt.Sprintf("Failed updating port forward: %s", err))
		return
	}
	defer res.Body.Close()
	raw, _ := ioutil.ReadAll(res.Body)

	var result struct {
		Success bool                 `json:"success"`
		Result  PortForwardingConfig `json:"result"`
	}
	_ = json.Unmarshal(raw, &result)
	if !result.Success {
		resp.Diagnostics.AddError("API error", fmt.Sprintf("Update failed: %s", string(raw)))
		return
	}

	plan.Hostname = types.StringValue(result.Result.Hostname)

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *PortForwardingResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state PortForwardingModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	url := fmt.Sprintf("http://mafreebox.freebox.fr/api/v8/fw/redir/%d", state.ID.ValueInt64())
	res, err := fbClient.DoRequest(http.MethodDelete, url, nil)
	if err != nil {
		resp.Diagnostics.AddError("API error", fmt.Sprintf("Failed deleting port forward: %s", err))
		return
	}
	defer res.Body.Close()

	// Remove from state
	resp.State.RemoveResource(ctx)
}
