package freebox

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	pschema "github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const hardcodedAppID = "fr.freebox.terraform"

// ---------- HTTP client & auth ----------

type Client struct {
	baseURL      string
	appToken     string
	http         *http.Client
	sessionToken string
}

func (c *Client) newRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.sessionToken != "" {
		req.Header.Set("X-Fbx-App-Auth", c.sessionToken)
	}
	return req, nil
}

func (c *Client) openSession(ctx context.Context) error {
	// Step 1: get challenge
	type loginResp struct {
		Success bool `json:"success"`
		Result  struct {
			Challenge string `json:"challenge"`
		} `json:"result"`
		Msg       string `json:"msg"`
		ErrorCode string `json:"error_code"`
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/login/", nil)
	if err != nil {
		return fmt.Errorf("build login request: %w", err)
	}
	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("get challenge: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(res.Body)
		return fmt.Errorf("get challenge status %d: %s", res.StatusCode, string(b))
	}

	var l loginResp
	if err := json.NewDecoder(res.Body).Decode(&l); err != nil {
		return fmt.Errorf("decode challenge: %w", err)
	}
	if l.Result.Challenge == "" {
		return fmt.Errorf("empty challenge from Freebox")
	}

	// Step 2: compute password = HMAC-SHA1(app_token, challenge)
	mac := hmac.New(sha1.New, []byte(c.appToken))
	mac.Write([]byte(l.Result.Challenge))
	password := hex.EncodeToString(mac.Sum(nil))

	// Step 3: open session
	payload := map[string]string{
		"app_id":   hardcodedAppID,
		"password": password,
	}
	body, _ := json.Marshal(payload)

	sessReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/login/session/", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("build session request: %w", err)
	}
	sessReq.Header.Set("Content-Type", "application/json")

	sessRes, err := c.http.Do(sessReq)
	if err != nil {
		return fmt.Errorf("open session: %w", err)
	}
	defer sessRes.Body.Close()

	if sessRes.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(sessRes.Body)
		return fmt.Errorf("open session status %d: %s", sessRes.StatusCode, string(b))
	}

	var env struct {
		Success bool `json:"success"`
		Result  struct {
			SessionToken string `json:"session_token"`
		} `json:"result"`
		Msg       string `json:"msg"`
		ErrorCode string `json:"error_code"`
	}
	if err := json.NewDecoder(sessRes.Body).Decode(&env); err != nil {
		return fmt.Errorf("decode session: %w", err)
	}
	if !env.Success || env.Result.SessionToken == "" {
		return fmt.Errorf("no session token returned (msg=%s, error_code=%s)", env.Msg, env.ErrorCode)
	}

	c.sessionToken = env.Result.SessionToken
	return nil
}

// ---------- Provider ----------

func New() provider.Provider { return &freeboxProvider{} }

type freeboxProvider struct{}

func (p *freeboxProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "freebox"
}

func (p *freeboxProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = pschema.Schema{
		Attributes: map[string]pschema.Attribute{
			"app_token": pschema.StringAttribute{
				Required:    true,
				Description: "Freebox application token (after approving the app on the Freebox).",
			},
			"base_url": pschema.StringAttribute{
				Optional:    true,
				Description: "Freebox API base URL. Default is http://mafreebox.freebox.fr/api/v8",
			},
		},
	}
}

func (p *freeboxProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewDhcpLeaseResource,
		NewDhcpConfigResource,
	}
}

func (p *freeboxProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewDhcpLeasesDataSource,
		NewDhcpConfigDataSource,
	}
}

func (p *freeboxProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var cfg struct {
		AppToken string `tfsdk:"app_token"`
		BaseURL  string `tfsdk:"base_url"`
	}
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if cfg.BaseURL == "" {
		cfg.BaseURL = "http://mafreebox.freebox.fr/api/v8"
	}

	c := &Client{
		baseURL:  cfg.BaseURL,
		appToken: cfg.AppToken,
		http:     &http.Client{Timeout: 15 * time.Second},
	}

	if err := c.openSession(ctx); err != nil {
		resp.Diagnostics.AddError("Failed to authenticate to Freebox", err.Error())
		return
	}

	tflog.Debug(ctx, "Freebox client configured", map[string]any{
		"base_url": cfg.BaseURL,
		"app_id":   hardcodedAppID,
	})

	resp.ResourceData = c
	resp.DataSourceData = c
}
