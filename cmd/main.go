package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/robert-cronin/gql2jsonschema-go/pkg"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile            string
	inputFile          string
	outputFile         string
	endpoint           string
	headers            []string
	timeout            int
	ignoreInternals    bool
	nullableArrayItems bool
	idTypeMapping      string
)

// Define the introspection query
const introspectionQuery = `
query IntrospectionQuery {
  __schema {
    queryType { name }
    mutationType { name }
    types {
      kind
      name
      description
      fields {
        name
        description
        args {
          name
          description
          type {
            kind
            name
            ofType {
              kind
              name
              ofType {
                kind
                name
                ofType {
                  kind
                  name
                }
              }
            }
          }
          defaultValue
        }
        type {
          kind
          name
          ofType {
            kind
            name
            ofType {
              kind
              name
              ofType {
                kind
                name
              }
            }
          }
        }
      }
      inputFields {
        name
        description
        type {
          kind
          name
          ofType {
            kind
            name
            ofType {
              kind
              name
              ofType {
                kind
                name
              }
            }
          }
        }
        defaultValue
      }
      interfaces {
        kind
        name
        ofType {
          kind
          name
          ofType {
            kind
            name
            ofType {
              kind
              name
            }
          }
        }
      }
      enumValues {
        name
        description
      }
      possibleTypes {
        kind
        name
        ofType {
          kind
          name
          ofType {
            kind
            name
            ofType {
              kind
              name
            }
          }
        }
      }
    }
  }
}
`

var rootCmd = &cobra.Command{
	Use:   "gql2jsonschema",
	Short: "Convert GraphQL Schema to JSON Schema",
	Long: `A command line tool to convert GraphQL Schema to JSON Schema.
Supports three input methods:
1. GraphQL endpoint URL (--endpoint)
2. Input file with introspection query result (--input)
3. Stdin (pipe or redirect introspection query result)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runConversion()
	},
}

func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".gql2jsonschema" (without extension)
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".gql2jsonschema")
	}

	// Read in environment variables that match
	viper.SetEnvPrefix("GRAPHQL2JSON")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	// If a config file is found, read it in
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.gql2jsonschema.yaml)")

	// Local flags
	rootCmd.Flags().StringVarP(&inputFile, "input", "i", "", "input file containing GraphQL introspection query result")
	rootCmd.Flags().StringVarP(&outputFile, "output", "o", "", "output file for JSON Schema (default is stdout)")
	rootCmd.Flags().StringVarP(&endpoint, "endpoint", "e", "", "GraphQL endpoint URL")
	rootCmd.Flags().StringArrayVarP(&headers, "header", "H", []string{}, "HTTP headers for endpoint (format: 'Key: Value')")
	rootCmd.Flags().IntVarP(&timeout, "timeout", "t", 30, "timeout in seconds for HTTP requests")
	rootCmd.Flags().BoolVar(&ignoreInternals, "ignore-internals", true, "ignore GraphQL internal types")
	rootCmd.Flags().BoolVar(&nullableArrayItems, "nullable-array-items", false, "properly represent nullable items in arrays")
	rootCmd.Flags().StringVar(&idTypeMapping, "id-type", "string", "how to represent ID type (string, number, or both)")

	// Bind flags to viper
	viper.BindPFlag("input", rootCmd.Flags().Lookup("input"))
	viper.BindPFlag("output", rootCmd.Flags().Lookup("output"))
	viper.BindPFlag("endpoint", rootCmd.Flags().Lookup("endpoint"))
	viper.BindPFlag("headers", rootCmd.Flags().Lookup("header"))
	viper.BindPFlag("timeout", rootCmd.Flags().Lookup("timeout"))
	viper.BindPFlag("ignore-internals", rootCmd.Flags().Lookup("ignore-internals"))
	viper.BindPFlag("nullable-array-items", rootCmd.Flags().Lookup("nullable-array-items"))
	viper.BindPFlag("id-type", rootCmd.Flags().Lookup("id-type"))
}

type GraphQLResponse struct {
	Data   *pkg.IntrospectionQuery `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors,omitempty"`
}

func getIntrospectionFromEndpoint(endpoint string, headers []string) (*pkg.IntrospectionQuery, error) {
	// Prepare the request payload
	payload := map[string]interface{}{
		"query": introspectionQuery,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("error marshaling query: %w", err)
	}

	// Create request
	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	for _, header := range headers {
		parts := strings.SplitN(header, ":", 2)
		if len(parts) == 2 {
			req.Header.Set(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
		}
	}

	// Create client with timeout
	client := &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
	}

	// Make request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %w", err)
	}

	// Parse response
	var graphqlResp GraphQLResponse
	if err := json.Unmarshal(body, &graphqlResp); err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}

	// Check for GraphQL errors
	if len(graphqlResp.Errors) > 0 {
		return nil, fmt.Errorf("GraphQL error: %s", graphqlResp.Errors[0].Message)
	}

	if graphqlResp.Data == nil {
		return nil, fmt.Errorf("no data in response")
	}

	return graphqlResp.Data, nil
}

func getIntrospectionFromStdin() (*pkg.IntrospectionQuery, error) {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return nil, fmt.Errorf("error checking stdin: %w", err)
	}

	if (stat.Mode() & os.ModeCharDevice) != 0 {
		return nil, nil // stdin is not piped/redirected
	}

	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return nil, fmt.Errorf("error reading from stdin: %w", err)
	}

	var introspection pkg.IntrospectionQuery
	if err := json.Unmarshal(data, &introspection); err != nil {
		// Try unwrapping from GraphQL response
		var graphqlResp GraphQLResponse
		if err2 := json.Unmarshal(data, &graphqlResp); err2 == nil && graphqlResp.Data != nil {
			return graphqlResp.Data, nil
		}
		return nil, fmt.Errorf("error parsing stdin data: %w", err)
	}

	return &introspection, nil
}

func runConversion() error {
	var introspection *pkg.IntrospectionQuery
	var err error

	// Try getting data from endpoint first
	if endpoint := viper.GetString("endpoint"); endpoint != "" {
		fmt.Fprintf(os.Stderr, "Fetching schema from endpoint: %s\n", endpoint)
		introspection, err = getIntrospectionFromEndpoint(endpoint, viper.GetStringSlice("headers"))
		if err != nil {
			return err
		}
	} else if inputFile := viper.GetString("input"); inputFile != "" {
		// Try input file
		data, err := os.ReadFile(inputFile)
		if err != nil {
			return fmt.Errorf("error reading input file: %w", err)
		}

		if err := json.Unmarshal(data, &introspection); err != nil {
			return fmt.Errorf("error parsing input file: %w", err)
		}
	} else {
		// Try stdin
		introspection, err = getIntrospectionFromStdin()
		if err != nil {
			return err
		}
		if introspection == nil {
			return fmt.Errorf("no input provided: use --endpoint, --input, or pipe data to stdin")
		}
	}

	// Create conversion options
	idMapping := pkg.IDTypeMapping(viper.GetString("id-type"))
	if !pkg.IsValidIDTypeMapping(idMapping) {
		return fmt.Errorf("invalid id-type mapping: %s", idMapping)
	}

	opts := pkg.Options{
		IgnoreInternals:    viper.GetBool("ignore-internals"),
		NullableArrayItems: viper.GetBool("nullable-array-items"),
		IDTypeMapping:      idMapping,
	}

	// Convert to JSON Schema
	schema, err := pkg.FromIntrospectionQuery(*introspection, &opts)
	if err != nil {
		return fmt.Errorf("error converting to JSON Schema: %w", err)
	}

	// Marshal the result
	output, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling JSON Schema: %w", err)
	}

	// Write output
	outputFile := viper.GetString("output")
	if outputFile == "" {
		// Write to stdout
		fmt.Println(string(output))
		return nil
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(outputFile), 0755); err != nil {
		return fmt.Errorf("error creating output directory: %w", err)
	}

	// Write to file
	if err := os.WriteFile(outputFile, output, 0644); err != nil {
		return fmt.Errorf("error writing output file: %w", err)
	}

	return nil
}

func Execute() error {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return err
	}
	return nil
}
