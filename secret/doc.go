// Package secret provides a small, dependency-light secret resolution layer.
//
// It supports:
//   - Strict environment expansion (see ExpandEnvStrict)
//   - Pluggable secret providers (see Provider + Registry)
//   - Resolving secret references in configuration values (see Resolver)
//
// References use the prefix "secretref:":
//   - Full value:  secretref:bws:project/dotenv/key/OPENAI_API_KEY
//   - Inline use:  Bearer secretref:bws:project/dotenv/key/OPENAI_API_KEY
//
// The format is compatible with mcp-gateway's secretref approach.
package secret
