package notionapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"
)

// Config structure for reading JSON file
type Config struct {
	NotionToken string `json:"notion_token"`
	DatabaseID  string `json:"notion_database_id"`
}

var notionToken string
var databaseID string
var notionAPIURL string

var client = &http.Client{Timeout: 10 * time.Second}

// Load config.json
func LoadConfig() error {
	// ✅ Clear `api_response.json` at the start
	if err := os.WriteFile("api_response.json", []byte("{}"), 0644); err != nil {
		fmt.Println("❌ Failed to clear `api_response.json`:", err)
	} else {
		fmt.Println("🗑 Cleared `api_response.json` before fetching new data")
	}

	file, err := os.ReadFile("config.json")
	if err != nil {
		return fmt.Errorf("failed to read config.json: %w", err)
	}

	var config Config
	if err := json.Unmarshal(file, &config); err != nil {
		return fmt.Errorf("failed to parse config.json: %w", err)
	}

	// Debugging: Print loaded values
	fmt.Println("🔍 DEBUG: Loaded Notion Token:", config.NotionToken)
	fmt.Println("🔍 DEBUG: Loaded Database ID:", config.DatabaseID)

	// Ensure values are not empty
	if config.NotionToken == "" {
		return fmt.Errorf("notion_token is missing in config.json")
	}
	if config.DatabaseID == "" {
		return fmt.Errorf("notion_database_id is missing in config.json")
	}

	// Set global variables
	notionToken = config.NotionToken
	databaseID = config.DatabaseID
	notionAPIURL = "https://api.notion.com/v1/databases/" + databaseID + "/query"

	fmt.Println("🔍 DEBUG: Notion API URL:", notionAPIURL)
	fmt.Println("✅ Config loaded successfully!")
	return nil
}

// Fetch Notion Data (Supports Pagination)
func FetchNotionData() ([]map[string]interface{}, error) {
	var allData []map[string]interface{}
	hasMore := true
	startCursor := ""

	for hasMore {
		payload := map[string]interface{}{
			"page_size": 100,
		}
		if startCursor != "" {
			payload["start_cursor"] = startCursor
		}

		payloadBytes, _ := json.Marshal(payload)
		req, err := http.NewRequest("POST", notionAPIURL, bytes.NewReader(payloadBytes))
		if err != nil {
			return nil, err
		}

		req.Header.Set("Authorization", "Bearer "+notionToken)
		req.Header.Set("Notion-Version", "2022-06-28")
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		// ✅ Format JSON before saving
		var formattedJSON bytes.Buffer
		if err := json.Indent(&formattedJSON, body, "", "    "); err != nil {
			fmt.Println("❌ Failed to format JSON:", err)
			return nil, err
		}

		// ✅ Write formatted API response to `api_response.json`
		if err := os.WriteFile("api_response.json", formattedJSON.Bytes(), 0644); err != nil {
			fmt.Println("❌ Failed to write API response to file:", err)
		} else {
			fmt.Println("📁 API response saved and formatted in `api_response.json`")
		}

		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			return nil, err
		}

		if results, ok := result["results"].([]interface{}); ok {
			for _, r := range results {
				if page, ok := r.(map[string]interface{}); ok {
					allData = append(allData, page)
				}
			}
		}

		// ✅ Safe type assertion for "has_more"
		if hasMoreVal, ok := result["has_more"].(bool); ok {
			hasMore = hasMoreVal
		} else {
			hasMore = false
		}

		// ✅ Safe check for "next_cursor"
		if nextCursor, ok := result["next_cursor"].(string); ok {
			startCursor = nextCursor
		} else {
			startCursor = ""
		}
	}

	return allData, nil
}

// Fetch Name (Page Title)
func GetName(props map[string]interface{}, key string) string {
	name := "No Name"
	if nameProp, exists := props[key]; exists {
		if titleList, ok := nameProp.(map[string]interface{})["title"].([]interface{}); ok && len(titleList) > 0 {
			if firstTitle, ok := titleList[0].(map[string]interface{}); ok {
				if text, exists := firstTitle["text"].(map[string]interface{}); exists {
					if content, exists := text["content"].(string); exists {
						name = content
					}
				}
			}
		}
	}
	return name
}

// Fetch Status
func GetStatus(props map[string]interface{}, key string) string {
	status := "No Status"

	// Check if "Asset Status" exists
	if assetStatus, exists := props[key]; exists {
		if assetStatusMap, ok := assetStatus.(map[string]interface{}); ok {
			// Check if "status" exists within "Asset Status"
			if statusMap, exists := assetStatusMap["status"]; exists {
				if statusDetails, ok := statusMap.(map[string]interface{}); ok {
					// Extract the "name" field from "status"
					if name, exists := statusDetails["name"].(string); exists {
						status = name
					}
				}
			}
		}
	}

	return status
}

// Fetch Float Value
func GetFloatValue(props map[string]interface{}, key string) string {
	if numProp, exists := props[key]; exists {
		if num, ok := numProp.(map[string]interface{})["number"].(float64); ok {
			return fmt.Sprintf("%.2f", num)
		}
	}
	return ""
}

// Fetch Integer Value
func GetIntValue(props map[string]interface{}, key string) string {
	if numProp, exists := props[key]; exists {
		if num, ok := numProp.(map[string]interface{})["number"].(float64); ok {
			return fmt.Sprintf("%d", int(num)) // Convert float64 to int and format as a string
		}
	}
	return ""
}

// Fetch Plain Text Value
func GetPlainTextValue(props map[string]interface{}, key string) string {
	if field, exists := props[key]; exists {
		if fieldMap, ok := field.(map[string]interface{}); ok {
			if richTextArray, exists := fieldMap["rich_text"]; exists {
				if texts, ok := richTextArray.([]interface{}); ok && len(texts) > 0 {
					if textMap, ok := texts[0].(map[string]interface{}); ok {
						if plainText, exists := textMap["plain_text"].(string); exists {
							return plainText
						}
					}
				}
			}
		}
	}
	return ""
}

// Fetch Select Value
func GetSelectValue(props map[string]interface{}, key string) string {
	if field, exists := props[key]; exists {
		if fieldMap, ok := field.(map[string]interface{}); ok {
			if selectField, exists := fieldMap["select"]; exists && selectField != nil {
				if selectMap, ok := selectField.(map[string]interface{}); ok {
					if name, exists := selectMap["name"].(string); exists {
						return name
					}
				}
			}
		}
	}
	return ""
}

// GetMultiSelectStrings extracts all string values from a multi-select rollup
func GetMultiSelectStrings(props map[string]interface{}, key string) []string {
	var values []string

	// Check if key exists in properties
	if field, exists := props[key]; exists {
		if fieldMap, ok := field.(map[string]interface{}); ok {
			if multiSelectField, exists := fieldMap["multi_select"]; exists {
				if multiSelectList, ok := multiSelectField.([]interface{}); ok {
					for _, item := range multiSelectList {
						if itemMap, ok := item.(map[string]interface{}); ok {
							if name, exists := itemMap["name"].(string); exists {
								values = append(values, name)
							}
						}
					}
				}
			}
		}
	}

	return values
}

// Fetch Date Value (Formatted MM/DD/YYYY)
func GetDateValue(props map[string]interface{}, key string) string {
	if dateProp, exists := props[key]; exists {
		if dateObj, ok := dateProp.(map[string]interface{})["date"].(map[string]interface{}); ok {
			if dateStr, ok := dateObj["start"].(string); ok && dateStr != "" {
				parsedDate, err := time.Parse("2006-01-02", dateStr)
				if err == nil {
					return parsedDate.Format("01/02/2006")
				}
				return dateStr // fallback if parsing fails
			}
		}
	}
	return ""
}

// Fetch URL Value
func GetURLValue(props map[string]interface{}, key string) string {
	if field, exists := props[key]; exists {
		if fieldMap, ok := field.(map[string]interface{}); ok {
			if url, exists := fieldMap["url"].(string); exists {
				return url
			}
		}
	}
	return ""
}

// Fetch Clean Plain Text Value (Removes Leading/Trailing Newlines and Spaces)
func GetCleanPlainTextValue(props map[string]interface{}, key string) string {
	// Check if the key exists
	if field, exists := props[key]; exists {
		if fieldMap, ok := field.(map[string]interface{}); ok {
			// Extract "rich_text" field
			if richTextArray, exists := fieldMap["rich_text"]; exists {
				if texts, ok := richTextArray.([]interface{}); ok && len(texts) > 0 {
					// Extract "plain_text" field from the first rich_text item
					if textMap, ok := texts[0].(map[string]interface{}); ok {
						if plainText, exists := textMap["plain_text"].(string); exists {
							// Trim whitespace and newlines, then return the cleaned plain text
							return strings.TrimSpace(plainText)
						}
					}
				}
			}
		}
	}
	return ""
}

// Fetch URL Value (Removes Leading/Trailing Newlines and Spaces)
func GetCleanURL(props map[string]interface{}, key string) string {
	// Check if the key exists
	if field, exists := props[key]; exists {
		if fieldMap, ok := field.(map[string]interface{}); ok {
			// Extract "url" field
			if url, exists := fieldMap["url"].(string); exists {
				// Trim whitespace and newlines, then return the cleaned URL
				return strings.TrimSpace(url)
			}
		}
	}
	return ""
}

// Fetch Email Value (Removes Leading/Trailing Newlines and Spaces)
func GetCleanEmailValue(props map[string]interface{}, key string) string {
	// Check if the key exists
	if field, exists := props[key]; exists {
		if fieldMap, ok := field.(map[string]interface{}); ok {
			// Extract "email" field
			if email, exists := fieldMap["email"].(string); exists {
				// Trim whitespace and newlines, then return the cleaned email
				return strings.TrimSpace(email)
			}
		}
	}
	return ""
}

// Fetch Phone Number Value (Removes Leading/Trailing Whitespace)
func GetPhoneNumberValue(props map[string]interface{}, key string) string {
	// Check if the key exists
	if field, exists := props[key]; exists {
		if fieldMap, ok := field.(map[string]interface{}); ok {
			// Extract "phone_number" field
			if phoneNumber, exists := fieldMap["phone_number"].(string); exists {
				// Trim whitespace and return the cleaned phone number
				return strings.TrimSpace(phoneNumber)
			}
		}
	}
	return ""
}

// Fetch Formula Text Value
func GetFormulaTextValue(props map[string]interface{}, key string) string {
	// Append " (As Text)" to the key
	lookupKey := key + " (As Text)"

	// Check if the property exists
	if field, exists := props[lookupKey]; exists {
		if fieldMap, ok := field.(map[string]interface{}); ok {
			// Ensure it has a "formula" field
			if formulaField, exists := fieldMap["formula"]; exists {
				if formulaMap, ok := formulaField.(map[string]interface{}); ok {
					// Extract the "string" value from the formula field
					if textValue, exists := formulaMap["string"].(string); exists {
						return textValue
					}
				}
			}
		}
	}
	return ""
}

// Fetch Formula Number Value
func GetFormulaNumberValue(props map[string]interface{}, key string) float64 {
	// Ensure the key exists in properties
	if field, exists := props[key]; exists {
		if fieldMap, ok := field.(map[string]interface{}); ok {
			// Ensure the field contains a "formula"
			if formulaField, exists := fieldMap["formula"]; exists {
				if formulaMap, ok := formulaField.(map[string]interface{}); ok {
					// Extract the "number" field
					if numberValue, exists := formulaMap["number"].(float64); exists {
						return numberValue
					}
				}
			}
		}
	}

	// If no valid number was found, return 0.0
	return 0.0
}

// Fetch Rollup Plain Text Value
func GetRollupPlainText(props map[string]interface{}, key string) []string {
	var values []string

	// Check if the key exists
	if field, exists := props[key]; exists {
		if fieldMap, ok := field.(map[string]interface{}); ok {
			// Check if it contains a "rollup" field
			if rollupField, exists := fieldMap["rollup"]; exists {
				if rollupMap, ok := rollupField.(map[string]interface{}); ok {
					// Check if "array" exists inside the rollup
					if arrayField, exists := rollupMap["array"]; exists {
						if arrayItems, ok := arrayField.([]interface{}); ok {
							// If the array is empty, append an empty string
							if len(arrayItems) == 0 {
								return []string{""}
							}

							// Iterate over array items
							for _, item := range arrayItems {
								if itemMap, ok := item.(map[string]interface{}); ok {
									// Check if "rich_text" exists
									if richTextArray, exists := itemMap["rich_text"]; exists {
										if richTextItems, ok := richTextArray.([]interface{}); ok && len(richTextItems) > 0 {
											// Extract the "plain_text" field from the first rich_text item
											if richTextMap, ok := richTextItems[0].(map[string]interface{}); ok {
												if plainText, exists := richTextMap["plain_text"].(string); exists {
													values = append(values, plainText)
												}
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}

	// If no values were extracted, return a slice containing an empty string
	if len(values) == 0 {
		return []string{""}
	}

	return values
}

// Fetch Rollup Formula String Value
func GetRollupFormulaString(props map[string]interface{}, key string) string {
	// Ensure the key exists in properties
	if field, exists := props[key]; exists {
		if fieldMap, ok := field.(map[string]interface{}); ok {
			// Ensure the field contains a "rollup"
			if rollupField, exists := fieldMap["rollup"]; exists {
				if rollupMap, ok := rollupField.(map[string]interface{}); ok {
					// Ensure "array" exists and is a list
					if arrayField, exists := rollupMap["array"]; exists {
						if arrayItems, ok := arrayField.([]interface{}); ok {
							// If the array is empty, return an empty string
							if len(arrayItems) == 0 {
								return ""
							}

							// Iterate over array items
							for _, item := range arrayItems {
								if itemMap, ok := item.(map[string]interface{}); ok {
									// Ensure the item contains a "formula"
									if formulaField, exists := itemMap["formula"]; exists {
										if formulaMap, ok := formulaField.(map[string]interface{}); ok {
											// Extract the "string" field from formula
											if textValue, exists := formulaMap["string"].(string); exists {
												return textValue
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}

	// If no valid string was found, return an empty string
	return ""
}
