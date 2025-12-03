#!/usr/bin/env python3
"""
Simple MCP client test script.
Sends initialize, list_tools, and call_tool requests to the local MCP server.
"""

import json
import sys
import requests

def test_mcp_server(url="http://localhost:8080/mcp/stream"):
    """Send a sequence of MCP requests and print responses."""
    
    requests_to_send = [
        {
            "jsonrpc": "2.0",
            "id": 1,
            "method": "initialize",
            "params": {
                "client_name": "test-client",
                "client_version": "0.1.0"
            }
        },
        {
            "jsonrpc": "2.0",
            "id": 2,
            "method": "tools/list",
            "params": {}
        },
        {
            "jsonrpc": "2.0",
            "id": 3,
            "method": "tools/call",
            "params": {
                "name": "job_search",
                "arguments": {
                    "query": "ai engineer"
                }
            }
        }
    ]
    
    # Build NDJSON payload (one JSON object per line)
    payload = "\n".join(json.dumps(req) for req in requests_to_send) + "\n"
    
    print(f"Sending {len(requests_to_send)} requests to {url}...")
    print(f"Payload:\n{payload}\n")
    
    try:
        response = requests.post(
            url,
            data=payload,
            headers={
                "Content-Type": "application/x-ndjson"
            },
            stream=True,
            timeout=10
        )
        
        print(f"Status Code: {response.status_code}\n")
        
        if response.status_code != 200:
            print(f"Error: {response.text}")
            return
        
        print("Responses:")
        print("-" * 80)
        
        # Read NDJSON responses line by line
        for line in response.iter_lines():
            if line:
                try:
                    resp_json = json.loads(line)
                    print(json.dumps(resp_json, indent=2))
                    print("-" * 80)
                except json.JSONDecodeError as e:
                    print(f"Failed to parse response line: {line}")
                    print(f"Error: {e}")
        
        print("\nTest completed successfully!")
        
    except requests.exceptions.RequestException as e:
        print(f"Request failed: {e}", file=sys.stderr)
        sys.exit(1)

if __name__ == "__main__":
    url = sys.argv[1] if len(sys.argv) > 1 else "http://localhost:8080/mcp/stream"
    test_mcp_server(url)

