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

// Ensure interface compliance
var (
	_ datasource.DataSource              = &portForwardsDataSource{}
	_ datasource.DataSourceWithConfigure = &portForwardsDataSource{}
)

// Constructor
func NewPortForwardingsDataSource() datasource.DataSource { return &portForwardsDataSource{} }

// Data source
type portForwardsDataSource struct{ client *Client }

// TF models
type pfItemOut struct {
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

type pfDSModel struct {
	Id       types.String `tfsdk:"id"`
	Forwards []pfItemOut  `tfsdk:"forwards"`
}

func (d *portForwardsDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "freebox_port_forwardings"
}

func (d *portForwardsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = dschema.Schema{
		Description: "List all Freebox port forwarding rules (fw/redir).",
		Attributes: map[string]dschema.Attribute{
			"id": dschema.StringAttribute{
				Computed:    true,
				Description: "Synthetic ID for this data source.",
			},
			"forwards": dschema.ListNestedAttribute{
				Computed:    true,
				Description: "All port forwarding rules.",
				NestedObject: dschema.NestedAttributeObject{
					Attributes: map[string]dschema.Attribute{
						"id":             dschema.Int64Attribute{Computed: true, Description: "Rule ID."},
						"enabled":        dschema.BoolAttribute{Computed: true},
						"ip_proto":       dschema.StringAttribute{Computed: true},
						"wan_port_start": dschema.Int64Attribute{Computed: true},
						"wan_port_end":   dschema.Int64Attribute{Computed: true},
						"lan_ip":         dschema.StringAttribute{Computed: true},
						"lan_port":       dschema.Int64Attribute{Computed: true},
						"src_ip":         dschema.StringAttribute{Computed: true},
						"comment":        dschema.StringAttribute{Computed: true},
						"hostname":       dschema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *portForwardsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData != nil {
		d.client = req.ProviderData.(*Client)
	}
}

func (d *portForwardsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		resp.Diagnostics.AddError("Client not configured", "Provider client is nil")
		return
	}

	// GET /fw/redir/
	hreq, _ := d.client.newRequest(ctx, http.MethodGet, "/fw/redir/", nil)
	hres, err := d.client.http.Do(hreq)
	if err != nil {
		resp.Diagnostics.AddError("API error", err.Error())
		return
	}
	defer hres.Body.Close()

	if hres.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(hres.Body)
		resp.Diagnostics.AddError("API error", fmt.Sprintf("status %d: %s", hres.StatusCode, string(body)))
		return
	}

	var env apiEnvelope[[]apiPortForward]
	if err := json.NewDecoder(hres.Body).Decode(&env); err != nil {
		resp.Diagnostics.AddError("Decode error", err.Error())
		return
	}
	if !env.Success {
		resp.Diagnostics.AddError("API error", env.Msg)
		return
	}

	out := pfDSModel{Id: types.StringValue("port_forwardings")}
	out.Forwards = make([]pfItemOut, 0, len(env.Result))
	for _, pf := range env.Result {
		out.Forwards = append(out.Forwards, pfItemOut{
			ID:           types.Int64Value(int64(pf.ID)),
			Enabled:      types.BoolValue(pf.Enabled),
			IpProto:      types.StringValue(pf.IpProto),
			WanPortStart: types.Int64Value(int64(pf.WanPortStart)),
			WanPortEnd:   types.Int64Value(int64(pf.WanPortEnd)),
			LanIP:        types.StringValue(pf.LanIP),
			LanPort:      types.Int64Value(int64(pf.LanPort)),
			SrcIP:        stringOrNull(pf.SrcIP),
			Comment:      stringOrNull(pf.Comment),
			Hostname:     stringOrNull(pf.Hostname),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &out)...)
}
