# GraphQL To JSON Schema

Convert graphql to json schema in Go. This library is basically a Go port of the TypeScript library: [graphql-to-json-schema](https://github.com/charlypoly/graphql-to-json-schema). The main difference, apart from the language, is that this library also allows you to specify a specific query or mutation to generate the schema from.

The main usecase that precipitated this library is to generate JSON schemas from GraphQL queries to be used in specifying the shape of the data for an LLM with structured outputs (e.g. OpenAI API). This is so it can generate a query that has a much better chance of being valid. Primary end consumer will be [guac-ai-mole](https://github.com/sozercan/guac-ai-mole)

## Usage

```bash
❯ go run . -e http://localhost:8080/query --method packagesList
{
  "$schema": "http://json-schema.org/draft-06/schema#",
  "properties": {
    "Query": {
      "$schema": "",
      "type": "object",
      "properties": {
        "packagesList": {
          "$schema": "",
          "type": "object",
          "properties": {
            "arguments": {
              "$schema": "",
              "type": "object",
              "properties": {
                "after": {
                  "$schema": "",
                  "type": "string",
                  "title": "ID"
                },
                "first": {
                  "$schema": "",
                  "type": "number",
                  "title": "Int"
                },
                "pkgSpec": {
                  "$schema": "",
                  "$ref": "#/definitions/PkgSpec"
                }
              },
              "required": [
                "pkgSpec"
              ]
            },
            "return": {
              "$schema": "",
              "$ref": "#/definitions/PackageConnection"
            }
          },
          "description": "Returns a paginated results via PackageConnection"
        }
      }
    }
  },
  "definitions": {
    "PackageConnection": {
      "$schema": "",
      "type": "object",
      "properties": {
        "edges": {
          "$schema": "",
          "type": "object",
          "properties": {
            "arguments": {
              "$schema": "",
              "type": "object"
            },
            "return": {
              "$schema": "",
              "type": "array",
              "items": {
                "$schema": "",
                "$ref": "#/definitions/PackageEdge"
              }
            }
          }
        },
        "pageInfo": {
          "$schema": "",
          "type": "object",
          "properties": {
            "arguments": {
              "$schema": "",
              "type": "object"
            },
            "return": {
              "$schema": "",
              "$ref": "#/definitions/PageInfo"
            }
          }
        },
        "totalCount": {
          "$schema": "",
          "type": "object",
          "properties": {
            "arguments": {
              "$schema": "",
              "type": "object"
            },
            "return": {
              "$schema": "",
              "type": "number",
              "title": "Int"
            }
          }
        }
      },
      "required": [
        "totalCount",
        "pageInfo",
        "edges"
      ],
      "description": "PackageConnection returns the paginated results for Package.\n\ntotalCount is the total number of results returned.\n\npageInfo provides information to the client if there is\na next page of results and the starting and\nending cursor for the current set.\n\nedges contains the PackageEdge which contains the current cursor\nand the Package node itself"
    },
    "PkgSpec": {
      "$schema": "",
      "type": "object",
      "properties": {
        "id": {
          "$schema": "",
          "type": "string",
          "title": "ID"
        },
        "matchOnlyEmptyQualifiers": {
          "$schema": "",
          "type": "boolean",
          "title": "Boolean",
          "default": false
        },
        "name": {
          "$schema": "",
          "type": "string",
          "title": "String"
        },
        "namespace": {
          "$schema": "",
          "type": "string",
          "title": "String"
        },
        "qualifiers": {
          "$schema": "",
          "type": "array",
          "items": {
            "$schema": "",
            "$ref": "#/definitions/PackageQualifierSpec"
          },
          "default": []
        },
        "subpath": {
          "$schema": "",
          "type": "string",
          "title": "String"
        },
        "type": {
          "$schema": "",
          "type": "string",
          "title": "String"
        },
        "version": {
          "$schema": "",
          "type": "string",
          "title": "String"
        }
      },
      "description": "PkgSpec allows filtering the list of sources to return in a query.\n\nEach field matches a qualifier from pURL. Use null to match on all values at\nthat level. For example, to get all packages in GUAC backend, use a PkgSpec\nwhere every field is null.\n\nThe id field can be used to match on a specific node in the trie to match packageTypeID, \npackageNamespaceID, packageNameID, or packageVersionID.\n\nEmpty string at a field means matching with the empty string. If passing in\nqualifiers, all of the values in the list must match. Since we want to return\nnodes with any number of qualifiers if no qualifiers are passed in the input,\nwe must also return the same set of nodes it the qualifiers list is empty. To\nmatch on nodes that don't contain any qualifier, set matchOnlyEmptyQualifiers\nto true. If this field is true, then the qualifiers argument is ignored."
    }
  }
}
```
