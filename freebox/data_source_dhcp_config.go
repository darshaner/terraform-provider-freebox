package freebox

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &dhcpConfigDataSource{}
	_ datasource.DataSourceWithConfigure = &dhcpConfigDataSource{}
)

func NewDhcpConfigDataSource() datasource.DataSource { return &dhcpConfigDataSource{} }

type dhcpConfigDataSource struct{ client *Client }

type dhcpConfigDSModel struct {
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

func (d *dhcpConfigDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "freebox_dhcp_config"
}

func (d *dhcpConfigDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = dschema.Schema{
		Description: "Read Freebox DHCP server configuration (API v8).",
		Attributes: map[string]dschema.Attribute{
			"id":                       dschema.StringAttribute{Computed: true, Description: "Synthetic ID."},
			"enabled":                  dschema.BoolAttribute{Computed: true},
			"sticky_assign":            dschema.BoolAttribute{Computed: true},
			"gateway":                  dschema.StringAttribute{Computed: true},
			"netmask":                  dschema.StringAttribute{Computed: true},
			"ip_range_start":           dschema.StringAttribute{Computed: true},
			"ip_range_end":             dschema.StringAttribute{Computed: true},
			"always_broadcast":         dschema.BoolAttribute{Computed: true},
			"ignore_out_of_range_hint": dschema.BoolAttribute{Computed: true},
			"dns":                      dschema.ListAttribute{Computed: true, ElementType: types.StringType},
		},
	}
}

func (d *dhcpConfigDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData != nil {
		d.client = req.ProviderData.(*Client)
	}
}

func (d *dhcpConfigDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		resp.Diagnostics.AddError("Client not configured", "Provider client is nil")
		return
	}
	req, _ := d.client.newRequest(ctx, http.MethodGet, "/dhcp/config/", nil)
	res, err := d.client.http.Do(req)
	if err != nil {
		resp.Diagnostics.AddError("API error", err.Error())
		return
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(res.Body)
		resp.Diagnostics.AddError("API error", fmt.Sprintf("status %d: %s", res.StatusCode, string(b)))
		return
	}

	var env envCfg[apiDhcpConfig]
	if err := json.NewDecoder(res.Body).Decode(&env); err != nil {
		resp.Diagnostics.AddError("Decode error", err.Error())
		return
	}
	if !env.Success {
		resp.Diagnostics.AddError("API error", env.Msg)
		return
	}

	state := dhcpConfigDSModel{
		Id:                   types.StringValue("dhcp_config"),
		Enabled:              types.BoolValue(env.Result.Enabled),
		StickyAssign:         types.BoolValue(env.Result.StickyAssign),
		Gateway:              stringOrNull(env.Result.Gateway),
		Netmask:              stringOrNull(env.Result.Netmask),
		IpRangeStart:         stringOrNull(env.Result.IPRangeStart),
		IpRangeEnd:           stringOrNull(env.Result.IPRangeEnd),
		AlwaysBroadcast:      types.BoolValue(env.Result.AlwaysBroadcast),
		IgnoreOutOfRangeHint: types.BoolValue(env.Result.IgnoreOutOfRangeHint),
		Dns:                  flattenStringList(env.Result.DNS),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
