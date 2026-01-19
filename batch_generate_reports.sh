#!/bin/bash

# Configuration
API_URL="http://localhost:8080/api/super/chat"
GNN_DIR="gnn_results"
OUTPUT_DIR="reports"

# Create output directory
mkdir -p "$OUTPUT_DIR"

echo "Starting Batch Report Generation..."

# Iterate over png files to identify threats
for file in "$GNN_DIR"/*.png; do
    # check if file exists (in case of no match)
    [ -e "$file" ] || continue

    filename=$(basename "$file")
    threat_name="${filename%.*}"

    echo "--------------------------------------------------"
    echo "Processing Threat: $threat_name"

    # Define Query
    QUERY="Analyze the security threat: $threat_name. Provide a comprehensive assessment."

    # Call API
    # We use jq to construct the JSON payload to ensure safe escaping
    PAYLOAD=$(jq -n --arg q "$QUERY" --arg img "" '{query: $q, image: $img}')
    
    RESPONSE=$(curl -s -X POST "$API_URL" -d "$PAYLOAD")

    # Check validity (simple check if 'final_report' key exists)
    if echo "$RESPONSE" | jq -e '.final_report' > /dev/null; then
        OUT_FILE="$OUTPUT_DIR/${threat_name}.md"
        
        # Format JSON to Markdown using jq
        echo "$RESPONSE" | jq -r '
            "# Report: " + "'"$threat_name"'" + "\n\n" +
            "## 🛡️ Super Agent Synthesis\n" + .final_report + "\n\n" +
            "## 👁️ Visual Analysis (VLM)\n" + .vlm_analysis + "\n\n" +
            "## 📈 GNN Trend Data\n" + .gnn_data + "\n\n" +
            "## 🕵️ Individual Agent Research\n" + 
            (.agent_reports | to_entries | map("- **" + .key + "**: " + .value) | join("\n\n"))
        ' > "$OUT_FILE"

        echo "✅ Report saved to: $OUT_FILE"
    else
        echo "❌ Failed to generate report for $threat_name"
        echo "Response Snippet: $(echo "$RESPONSE" | head -n 5)"
    fi
done

echo "--------------------------------------------------"
echo "Batch Processing Complete. Check '$OUTPUT_DIR/' for results."
