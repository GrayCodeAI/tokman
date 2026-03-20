package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/filter"
)

var (
	apiProxyPort     int
	apiProxyUpstream string
)

var apiProxyCmd = &cobra.Command{
	Use:   "api-proxy",
	Short: "HTTP reverse proxy that compresses LLM API responses",
	Long: `Start a reverse proxy that sits between your app and LLM API.
Compresses tool output in API responses before they reach the model.

Example:
  tokman api-proxy --port 7878 --upstream https://api.anthropic.com
  export ANTHROPIC_BASE_URL=http://localhost:7878`,
	RunE: runAPIProxy,
}

func init() {
	apiProxyCmd.Flags().IntVar(&apiProxyPort, "port", 7878, "proxy listen port")
	apiProxyCmd.Flags().StringVar(&apiProxyUpstream, "upstream", "", "upstream API URL")
	rootCmd.AddCommand(apiProxyCmd)
}

func runAPIProxy(cmd *cobra.Command, args []string) error {
	if apiProxyUpstream == "" {
		return fmt.Errorf("--upstream required")
	}

	upstream, err := url.Parse(apiProxyUpstream)
	if err != nil {
		return fmt.Errorf("invalid upstream URL: %w", err)
	}

	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = upstream.Scheme
			req.URL.Host = upstream.Host
			req.Host = upstream.Host
		},
		ModifyResponse: func(resp *http.Response) error {
			ct := resp.Header.Get("Content-Type")
			if !strings.Contains(ct, "application/json") {
				return nil
			}
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return err
			}
			resp.Body.Close()

			var data interface{}
			if json.Unmarshal(body, &data) == nil {
				data = compressAPIData(data)
				newBody, err := json.Marshal(data)
				if err != nil {
					resp.Body = io.NopCloser(strings.NewReader(string(body)))
					return nil
				}
				resp.Body = io.NopCloser(strings.NewReader(string(newBody)))
				resp.ContentLength = int64(len(newBody))
				resp.Header.Set("Content-Length", fmt.Sprintf("%d", len(newBody)))
			} else {
				resp.Body = io.NopCloser(strings.NewReader(string(body)))
			}
			return nil
		},
	}

	addr := fmt.Sprintf(":%d", apiProxyPort)
	fmt.Fprintf(os.Stderr, "tokman api-proxy on %s → %s\n", addr, apiProxyUpstream)
	return http.ListenAndServe(addr, proxy)
}

func compressAPIData(data interface{}) interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{})
		for k, val := range v {
			if s, ok := val.(string); ok && len(s) > 500 {
				p := filter.NewPipelineCoordinator(filter.PipelineConfig{
					Mode: filter.ModeMinimal, NgramEnabled: true,
					EnableCompaction: true, EnableAttribution: true,
				})
				c, _ := p.Process(s)
				result[k] = c
			} else {
				result[k] = compressAPIData(val)
			}
		}
		return result
	case []interface{}:
		for i, val := range v {
			v[i] = compressAPIData(val)
		}
		return v
	default:
		return v
	}
}
