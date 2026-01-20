package agent

import (
	"swu-cyber-security-agent-go/internal/db"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

// CVE Details Tool
type CveInput struct {
	CveID string `json:"cve_id" doc:"The CVE ID to look up (e.g., CVE-2023-1234)"`
}

func CveDetailsHandler(ctx tool.Context, input CveInput) (map[string]interface{}, error) {
	data, err := db.GetCveContext(input.CveID)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return map[string]interface{}{"message": "No data found for this CVE"}, nil
	}
	return data, nil
}

func NewCveTool() (tool.Tool, error) {
	return functiontool.New(functiontool.Config{
		Name:        "check_cve_details",
		Description: "Retrieves structured metadata for a CVE, including EPSS scores and CISA KEV status.",
	}, CveDetailsHandler)
}

// Indicator Search Tool
type IocInput struct {
	Indicator string `json:"indicator" doc:"The IP address or Domain to search for"`
}

func IocSearchHandler(ctx tool.Context, input IocInput) (interface{}, error) {
	results, err := db.SearchIoc(input.Indicator)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return map[string]interface{}{"message": "No indicators found"}, nil
	}
	return results, nil
}

func NewIocTool() (tool.Tool, error) {
	return functiontool.New(functiontool.Config{
		Name:        "search_indicators",
		Description: "Searches for an IP or Domain in the OTX threat intelligence database.",
	}, IocSearchHandler)
}
