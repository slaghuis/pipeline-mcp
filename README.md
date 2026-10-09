# Pipeline MCP server

 ## One-Time Setup
```
# 1. Build pipeline-lib (nothing to build — it's a library)
cd ~/code/ai-factory/pipeline-lib
go mod tidy

# 2. Build pipeline-mcp
cd ~/code/ai-factory/pipeline-mcp
go mod tidy
go build -o ~/.local/bin/pipeline-mcp ./cmd/pipeline-mcp

# 3. Build per-service pipelines (one time, then rebuilt per service)
cd ~/code/services/myservice/pipeline
go mod tidy
go build -o ../bin/pipeline .

# 4. Create the services registry
mkdir -p ~/.config/ai-factory
cat > ~/.config/ai-factory/services.yaml <<EOF
services:
  - name: myservice
    path: ~/code/services/myservice
    pipeline_binary: ./bin/pipeline
    config: ./pipeline.yaml
    default_env: staging
EOF

# 5. Create pipeline-mcp config
mkdir -p ~/.config/ai-factory
cat > ~/.config/ai-factory/pipeline-mcp.yaml <<EOF
listen: ":8766"
services_file: ~/.config/ai-factory/services.yaml
reports_dir: ~/.local/share/ai-factory/reports
EOF
```

 ## Launch
```
# From anywhere:
pipeline-mcp -config ~/.config/ai-factory/pipeline-mcp.yaml
```

## Wire into agents
Add to `~/.config/opencode/opencode.json`:
```
{
  "mcp": {
    "pipeline": {
      "type": "remote",
      "url": "http://localhost:8766/sse",
      "enabled": true
    }
  }
}
```
For Cursor `~/.cursor/mcp.json`:
```
{
  "mcpServers": {
    "pipeline": { "url": "http://localhost:8766/sse" }
  }
}
```
For Claude Code:
```
claude mcp add --transport sse pipeline http://localhost:8766/sse
```

 ## Smoke Test 
```
# From the inspector:
npx @modelcontextprotocol/inspector --transport sse http://localhost:8766/sse

# Try:
# - pipeline_list_services       → shows myservice
# - pipeline_run service=myservice command=lint
# - pipeline_run service=myservice command=release env=staging
#   → triggers Telegram approval; tap "Deploy" on your phone
```

 ## Add Another Service Later
```
# 1. In the new service repo:
mkdir -p pipeline
cat > pipeline/go.mod <<EOF
module github.com/slaghuis/paymentservice/pipeline
go 1.23
require github.com/slaghuis/pipeline-lib v0.0.0
replace github.com/slaghuis/pipeline-lib => ../../../ai-factory/pipeline-lib
EOF

cat > pipeline/main.go <<'EOF'
package main
import (
    "os"
    "github.com/slaghuis/pipeline-lib/cli"
)
func main() { os.Exit(cli.Run(os.Args[1:])) }
EOF

cat > pipeline.yaml <<EOF
service: paymentservice
# ... (see B.4)
EOF

cd pipeline && go mod tidy && go build -o ../bin/pipeline . && cd ..

# 2. Register it:
# Edit ~/.config/ai-factory/services.yaml and add the entry.

# 3. Reload pipeline-mcp:
kill -HUP $(pgrep pipeline-mcp)
```

Agents immediately see the new service via `pipeline_list_services`.


 ## Example Registry

In file `~/.config/ai-factory/services.yaml`
```
services:
  - name: myservice
    path: ${HOME}/code/services/myservice
    pipeline_binary: ./bin/pipeline
    default_env: staging

  - name: another-service
    path: ${HOME}/code/services/another-service
    pipeline_binary: ./bin/pipeline
    default_env: staging

  - name: payments
    path: ${HOME}/code/services/payments
    pipeline_binary: ./bin/pipeline
    default_env: staging
```
