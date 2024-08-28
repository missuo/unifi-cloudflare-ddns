/*
 * @Author: Vincent Yang
 * @Date: 2024-08-28 14:46:32
 * @LastEditors: Vincent Yang
 * @LastEditTime: 2024-08-28 15:04:54
 * @FilePath: /unifi-cloudflare-ddns/ddns.go
 * @Telegram: https://t.me/missuo
 * @GitHub: https://github.com/missuo
 *
 * Copyright © 2024 by Vincent, All Rights Reserved.
 */

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type Cloudflare struct {
	Token string
}

type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Error   interface{} `json:"error,omitempty"`
}

func (cf *Cloudflare) apiRequest(method, endpoint string, body interface{}) (map[string]interface{}, error) {
	url := "https://api.cloudflare.com/client/v4/" + endpoint
	client := &http.Client{}

	var req *http.Request
	var err error

	if body != nil {
		jsonBody, _ := json.Marshal(body)
		req, err = http.NewRequest(method, url, strings.NewReader(string(jsonBody)))
	} else {
		req, err = http.NewRequest(method, url, nil)
	}

	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cf.Token)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if success, ok := result["success"].(bool); !ok || !success {
		return result, fmt.Errorf("API request failed")
	}

	return result, nil
}

func (cf *Cloudflare) updateDNS(hostname, ip string) (map[string]interface{}, error) {
	// Find zone
	domainParts := strings.Split(hostname, ".")
	domain := strings.Join(domainParts[len(domainParts)-2:], ".")

	zoneResult, err := cf.apiRequest("GET", "zones?name="+domain, nil)
	if err != nil {
		return zoneResult, err
	}

	zones := zoneResult["result"].([]interface{})
	if len(zones) == 0 {
		return zoneResult, fmt.Errorf("zone not found")
	}
	zoneID := zones[0].(map[string]interface{})["id"].(string)

	// Find DNS record
	recordResult, err := cf.apiRequest("GET", fmt.Sprintf("zones/%s/dns_records?name=%s", zoneID, hostname), nil)
	if err != nil {
		return recordResult, err
	}

	records := recordResult["result"].([]interface{})
	var recordID string
	var recordType string

	if len(records) > 0 {
		record := records[0].(map[string]interface{})
		recordID = record["id"].(string)
		recordType = record["type"].(string)
	} else {
		recordType = "A"
		if strings.Contains(ip, ":") {
			recordType = "AAAA"
		}
	}

	// Update or create DNS record
	updateData := map[string]interface{}{
		"type":    recordType,
		"name":    hostname,
		"content": ip,
		"ttl":     1,
		"proxied": false,
	}

	var result map[string]interface{}
	if recordID != "" {
		result, err = cf.apiRequest("PUT", fmt.Sprintf("zones/%s/dns_records/%s", zoneID, recordID), updateData)
	} else {
		result, err = cf.apiRequest("POST", fmt.Sprintf("zones/%s/dns_records", zoneID), updateData)
	}

	return result, err
}

func handleUpdate(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "Missing token",
		})
		return
	}

	hostname := c.Query("hostname")
	if hostname == "" {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "Missing hostname",
		})
		return
	}

	ip := c.Query("ip")
	if ip == "" {
		ip = c.ClientIP()
	}

	cf := &Cloudflare{Token: token}
	result, err := cf.updateDNS(hostname, ip)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if result != nil {
			// If we have a result, it's likely a Cloudflare API error response
			statusCode = http.StatusBadRequest
		}
		c.JSON(statusCode, APIResponse{
			Success: false,
			Message: "Update failed",
			Error:   result, // This will contain the full Cloudflare API error response
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Message: "DNS updated successfully",
	})
}

func main() {
	r := gin.Default()
	r.GET("/update", handleUpdate)
	r.Run(":9909")
}
