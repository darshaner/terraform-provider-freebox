// Manage DHCP static leases (API v8): id == mac
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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &dhcpLeaseResource{}
	_ resource.ResourceWithConfigure   = &dhcpLeaseResource{}
	_ resource.ResourceWithImportState = &dhcpLeaseResource{}
)

func NewDhcpLeaseResource() resource.Resource { return &dhcpLeaseResource{} }

type dhcpLeaseResource struct{ client *Client }

type leaseModel struct {
	Id       types.String `tfsdk:"id"`
	Mac      types.String `tfsdk:"mac"`
	Ip       types.String `tfsdk:"ip"`
	Comment  types.String `tfsdk:"comment"`
	Hostname types.String `tfsdk:"hostname"`
	Host     types.String `tfsdk:"host"`
}

type apiLease struct {
	Id       string          `json:"id"`
	Mac      string          `json:"mac"`
	Ip       string          `json:"ip"`
	Comment  string          `json:"comment,omitempty"`
	Hostname string          `json:"hostname,omitempty"`
	Host     json.RawMessage `json:"host,omitempty"`
}

type apiEnvelope[T any] struct {
	Success   bool   `json:"success"`
	Result    T      `json:"result"`
	Msg       string `json:"msg"`
	ErrorCode string `json:"error_code"`
}

func (r *dhcpLeaseResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "freebox_dhcp_lease"
}

func (r *dhcpLeaseResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = rschema.Schema{
		Description: "Manage Freebox DHCP static leases (API v8).",
		Attributes: map[string]rschema.Attribute{
			"id":       rschema.StringAttribute{Computed: true, Description: "Lease id (equals MAC).", PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"mac":      rschema.StringAttribute{Required: true, Description: "Host MAC address.", PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
			"ip":       rschema.StringAttribute{Required: true, Description: "IPv4 to assign to the host."},
			"comment":  rschema.StringAttribute{Optional: true, Computed: true, Description: "Optional comment.", PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"hostname": rschema.StringAttribute{Computed: true, Description: "Read-only hostname matching the MAC.", PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"host":     rschema.StringAttribute{Computed: true, Description: "Raw JSON of LanHost (read-only).", PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
		},
	}
}

func (r *dhcpLeaseResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData != nil {
		r.client = req.ProviderData.(*Client)
	}
}

func (r *dhcpLeaseResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Client not configured", "Provider client is nil")
		return
	}
	var plan leaseModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload := map[string]string{"mac": plan.Mac.ValueString(), "ip": plan.Ip.ValueString()}
	if !plan.Comment.IsNull() {
		payload["comment"] = plan.Comment.ValueString()
	}

	b, _ := json.Marshal(payload)
	hreq, _ := r.client.newRequest(ctx, http.MethodPost, "/dhcp/static_lease/", bytes.NewBuffer(b))
	hres, err := r.client.http.Do(hreq)
	if err != nil {
		resp.Diagnostics.AddError("API error", err.Error())
		return
	}
	defer hres.Body.Close()

	var env apiEnvelope[apiLease]
	_ = json.NewDecoder(hres.Body).Decode(&env)
	if hres.StatusCode != http.StatusOK && hres.StatusCode != http.StatusCreated {
		resp.Diagnostics.AddError("API error", fmt.Sprintf("status %d: %s (code=%s)", hres.StatusCode, env.Msg, env.ErrorCode))
		return
	}
	if !env.Success {
		// Attempt adopt if already exists
		if env.ErrorCode == "already_exists" || env.ErrorCode == "exist" || env.ErrorCode == "conflict" {
			found, diag := r.findLeaseByMacOrIP(ctx, plan.Mac.ValueString(), plan.Ip.ValueString())
			if diag != "" {
				resp.Diagnostics.AddError("API error (list)", diag)
				return
			}
			if found != nil {
				resp.Diagnostics.Append(resp.State.Set(ctx, toState(*found))...)
				tflog.Info(ctx, "Adopted existing DHCP lease", map[string]any{"id": found.Id})
				return
			}
		}
		resp.Diagnostics.AddError("API error", fmt.Sprintf("success=false: %s (code=%s)", env.Msg, env.ErrorCode))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, toState(env.Result))...)
}

func (r *dhcpLeaseResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Client not configured", "Provider client is nil")
		return
	}
	var state leaseModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.Id.ValueString()
	if id == "" {
		id = state.Mac.ValueString()
	}
	if id == "" {
		resp.State.RemoveResource(ctx)
		return
	}
	path := fmt.Sprintf("/dhcp/static_lease/%s", id)
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
		b, _ := io.ReadAll(hres.Body)
		resp.Diagnostics.AddError("API error", fmt.Sprintf("status %d: %s", hres.StatusCode, string(b)))
		return
	}

	var env apiEnvelope[apiLease]
	if err := json.NewDecoder(hres.Body).Decode(&env); err != nil {
		resp.Diagnostics.AddError("Decode error", err.Error())
		return
	}
	if !env.Success {
		resp.Diagnostics.AddError("API error", env.Msg)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, toState(env.Result))...)
}

func (r *dhcpLeaseResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Client not configured", "Provider client is nil")
		return
	}
	var plan, state leaseModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	patch := map[string]string{}
	if !plan.Comment.IsNull() && plan.Comment.ValueString() != state.Comment.ValueString() {
		patch["comment"] = plan.Comment.ValueString()
	}
	if !plan.Ip.IsNull() && plan.Ip.ValueString() != state.Ip.ValueString() {
		patch["ip"] = plan.Ip.ValueString()
	}
	if len(patch) == 0 {
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		return
	}

	b, _ := json.Marshal(patch)
	id := state.Id.ValueString()
	if id == "" {
		id = state.Mac.ValueString()
	}
	path := fmt.Sprintf("/dhcp/static_lease/%s", id)
	preq, _ := r.client.newRequest(ctx, http.MethodPut, path, bytes.NewBuffer(b))
	pres, err := r.client.http.Do(preq)
	if err != nil {
		resp.Diagnostics.AddError("API error", err.Error())
		return
	}
	defer pres.Body.Close()

	var env apiEnvelope[apiLease]
	if err := json.NewDecoder(pres.Body).Decode(&env); err != nil {
		resp.Diagnostics.AddError("Decode error", err.Error())
		return
	}
	if pres.StatusCode != http.StatusOK || !env.Success {
		resp.Diagnostics.AddError("API error", fmt.Sprintf("update failed: status %d, msg=%s, code=%s", pres.StatusCode, env.Msg, env.ErrorCode))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, toState(env.Result))...)
}

func (r *dhcpLeaseResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Client not configured", "Provider client is nil")
		return
	}
	var id types.String
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &id)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if id.IsNull() || id.ValueString() == "" {
		var mac types.String
		_ = req.State.GetAttribute(ctx, path.Root("mac"), &mac)
		id = mac
	}
	if id.IsNull() || id.ValueString() == "" {
		return
	}

	path := fmt.Sprintf("/dhcp/static_lease/%s", id.ValueString())
	dreq, _ := r.client.newRequest(ctx, http.MethodDelete, path, nil)
	dres, err := r.client.http.Do(dreq)
	if err != nil {
		resp.Diagnostics.AddError("API error", err.Error())
		return
	}
	defer dres.Body.Close()
	if dres.StatusCode != http.StatusOK && dres.StatusCode != http.StatusNoContent {
		b, _ := io.ReadAll(dres.Body)
		resp.Diagnostics.AddError("API error", fmt.Sprintf("status %d: %s", dres.StatusCode, string(b)))
		return
	}
}

func (r *dhcpLeaseResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("mac"), req.ID)...) // id == mac
}

// helpers
func (r *dhcpLeaseResource) findLeaseByMacOrIP(ctx context.Context, mac, ip string) (*apiLease, string) {
	req, _ := r.client.newRequest(ctx, http.MethodGet, "/dhcp/static_lease/", nil)
	res, err := r.client.http.Do(req)
	if err != nil {
		return nil, err.Error()
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(res.Body)
		return nil, fmt.Sprintf("status %d: %s", res.StatusCode, string(b))
	}
	var env apiEnvelope[[]apiLease]
	if err := json.NewDecoder(res.Body).Decode(&env); err != nil {
		return nil, err.Error()
	}
	if !env.Success {
		return nil, env.Msg
	}
	for i := range env.Result {
		if (mac != "" && env.Result[i].Mac == mac) || (ip != "" && env.Result[i].Ip == ip) {
			return &env.Result[i], ""
		}
	}
	return nil, ""
}

func toState(l apiLease) *leaseModel {
	host := ""
	if len(l.Host) > 0 {
		host = string(l.Host)
	}
	id := l.Id
	if id == "" {
		id = l.Mac
	}
	return &leaseModel{Id: types.StringValue(id), Mac: types.StringValue(l.Mac), Ip: types.StringValue(l.Ip), Comment: stringOrNull(l.Comment), Hostname: stringOrNull(l.Hostname), Host: stringOrNull(host)}
}
