package freebox

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type PortForwardingsDataSource struct{}

type PortForwardingData struct {
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

type PortForwardingsModel struct {
	Forwards []PortForwardingData `tfsdk:"forwards"`
}

func NewPortForwardingsDataSource() datasource.DataSource {
	return &PortForwardingsDataSource{}
}

func (d *PortForwardingsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_port_forwardings"
}

func (d *PortForwardingsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"forwards": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":             schema.Int64Attribute{Computed: true},
						"enabled":        schema.BoolAttribute{Computed: true},
						"ip_proto":       schema.StringAttribute{Computed: true},
						"wan_port_start": schema.Int64Attribute{Computed: true},
						"wan_port_end":   schema.Int64Attribute{Computed: true},
						"lan_ip":         schema.StringAttribute{Computed: true},
						"lan_port":       schema.Int64Attribute{Computed: true},
						"src_ip":         schema.StringAttribute{Computed: true},
						"comment":        schema.StringAttribute{Computed: true},
						"hostname":       schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *PortForwardingsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	url := "http://mafreebox.freebox.fr/api/v8/fw/redir/"
	res, err := fbClient.DoRequest(http.MethodGet, url, nil)
	if err != nil {
		resp.Diagnostics.AddError("API error", fmt.Sprintf("Failed to fetch port forwardings: %s", err))
		return
	}
	defer res.Body.Close()
	raw, _ := ioutil.ReadAll(res.Body)

	var result struct {
		Success bool                   `json:"success"`
		Result  []PortForwardingConfig `json:"result"`
	}
	_ = json.Unmarshal(raw, &result)
	if !result.Success {
		resp.Diagnostics.AddError("API error", fmt.Sprintf("Failed: %s", string(raw)))
		return
	}

	var forwards []PortForwardingData
	for _, pf := range result.Result {
		forwards = append(forwards, PortForwardingData{
			ID:           types.Int64Value(int64(pf.ID)),
			Enabled:      types.BoolValue(pf.Enabled),
			IpProto:      types.StringValue(pf.IpProto),
			WanPortStart: types.Int64Value(int64(pf.WanPortStart)),
			WanPortEnd:   types.Int64Value(int64(pf.WanPortEnd)),
			LanIP:        types.StringValue(pf.LanIP),
			LanPort:      types.Int64Value(int64(pf.LanPort)),
			SrcIP:        types.StringValue(pf.SrcIP),
			Comment:      types.StringValue(pf.Comment),
			Hostname:     types.StringValue(pf.Hostname),
		})
	}

	state := PortForwardingsModel{Forwards: forwards}
	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
