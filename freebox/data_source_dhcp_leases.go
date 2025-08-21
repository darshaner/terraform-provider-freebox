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
	_ datasource.DataSource              = &dhcpLeasesDataSource{}
	_ datasource.DataSourceWithConfigure = &dhcpLeasesDataSource{}
)

func NewDhcpLeasesDataSource() datasource.DataSource { return &dhcpLeasesDataSource{} }

type dhcpLeasesDataSource struct{ client *Client }

type leasesDSModel struct {
	Id     types.String   `tfsdk:"id"`
	Leases []leaseItemOut `tfsdk:"leases"`
}

type leaseItemOut struct {
	Id       types.String `tfsdk:"id"`
	Mac      types.String `tfsdk:"mac"`
	Ip       types.String `tfsdk:"ip"`
	Comment  types.String `tfsdk:"comment"`
	Hostname types.String `tfsdk:"hostname"`
	Host     types.String `tfsdk:"host"`
}

func (d *dhcpLeasesDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "freebox_dhcp_leases"
}

func (d *dhcpLeasesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = dschema.Schema{
		Description: "List DHCP static leases (API v8).",
		Attributes: map[string]dschema.Attribute{
			"id": dschema.StringAttribute{Computed: true, Description: "Synthetic ID."},
			"leases": dschema.ListNestedAttribute{
				Computed:    true,
				Description: "All DHCP static leases.",
				NestedObject: dschema.NestedAttributeObject{Attributes: map[string]dschema.Attribute{
					"id":       dschema.StringAttribute{Computed: true},
					"mac":      dschema.StringAttribute{Computed: true},
					"ip":       dschema.StringAttribute{Computed: true},
					"comment":  dschema.StringAttribute{Computed: true},
					"hostname": dschema.StringAttribute{Computed: true},
					"host":     dschema.StringAttribute{Computed: true},
				}},
			},
		},
	}
}

func (d *dhcpLeasesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData != nil {
		d.client = req.ProviderData.(*Client)
	}
}

func (d *dhcpLeasesDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		resp.Diagnostics.AddError("Client not configured", "Provider client is nil")
		return
	}
	req, _ := d.client.newRequest(ctx, http.MethodGet, "/dhcp/static_lease/", nil)
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

	var env envCfg[[]apiLease]
	if err := json.NewDecoder(res.Body).Decode(&env); err != nil {
		resp.Diagnostics.AddError("Decode error", err.Error())
		return
	}
	if !env.Success {
		resp.Diagnostics.AddError("API error", env.Msg)
		return
	}

	out := leasesDSModel{Id: types.StringValue("dhcp_leases")}
	out.Leases = make([]leaseItemOut, 0, len(env.Result))
	for _, l := range env.Result {
		host := ""
		if len(l.Host) > 0 {
			host = string(l.Host)
		}
		id := l.Id
		if id == "" {
			id = l.Mac
		}
		out.Leases = append(out.Leases, leaseItemOut{
			Id: types.StringValue(id), Mac: types.StringValue(l.Mac), Ip: types.StringValue(l.Ip), Comment: stringOrNull(l.Comment), Hostname: stringOrNull(l.Hostname), Host: stringOrNull(host),
		})
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &out)...)
}
