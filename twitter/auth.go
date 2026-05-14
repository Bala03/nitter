package twitter

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"
)

const (
	twitterLoginURL    = "https://twitter.com"
	twitterAPIBase     = "https://api.twitter.com"
	loginFlowURL       = twitterAPIBase + "/1.1/onboarding/task.json"
	loginBearerToken   = "AAAAAAAAAAAAAAAAAAAAANRILgAAAAAAnNwIzUejRCOuH5E6I8xnZz4puTs%3D1Zv7ttfk8LF81IUq16cHjhLTvJu4FA33AGWWjCpTnA"
)

// TwitterAuth handles authenticating with Twitter and extracting session cookies
type TwitterAuth struct {
	httpClient *http.Client
}

func NewTwitterAuth() *TwitterAuth {
	jar, _ := cookiejar.New(nil)
	return &TwitterAuth{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Jar:     jar,
		},
	}
}

// Login attempts to authenticate with Twitter using username/password
// and returns a Session with auth_token and ct0 cookies
func (a *TwitterAuth) Login(username, password string) (*Session, error) {
	// Step 1: Initialize the login flow
	flowToken, err := a.initFlow()
	if err != nil {
		return nil, fmt.Errorf("init flow: %v", err)
	}

	// Step 2: Submit username (instrumentation)
	flowToken, err = a.submitInstrumentation(flowToken)
	if err != nil {
		return nil, fmt.Errorf("instrumentation: %v", err)
	}

	// Step 3: Submit username
	flowToken, err = a.submitUsername(flowToken, username)
	if err != nil {
		return nil, fmt.Errorf("submit username: %v", err)
	}

	// Step 4: Submit password
	flowToken, err = a.submitPassword(flowToken, password)
	if err != nil {
		return nil, fmt.Errorf("submit password: %v", err)
	}

	// Step 5: Handle potential extra steps (e.g., account duplication check)
	flowToken, _ = a.handleDuplicationCheck(flowToken)

	// Extract session cookies
	session, err := a.extractSession(username)
	if err != nil {
		return nil, fmt.Errorf("extract session: %v", err)
	}

	return session, nil
}

func (a *TwitterAuth) doFlowRequest(payload map[string]interface{}, queryParams ...string) (map[string]interface{}, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	reqURL := loginFlowURL
	if len(queryParams) > 0 {
		reqURL += "?" + strings.Join(queryParams, "&")
	}

	req, err := http.NewRequest("POST", reqURL, strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+loginBearerToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("x-twitter-active-user", "yes")
	req.Header.Set("x-twitter-client-language", "en")

	// Set guest token from cookies if available
	twitterURL, _ := url.Parse(twitterAPIBase)
	for _, cookie := range a.httpClient.Jar.Cookies(twitterURL) {
		if cookie.Name == "gt" {
			req.Header.Set("x-guest-token", cookie.Value)
		}
		if cookie.Name == "ct0" {
			req.Header.Set("x-csrf-token", cookie.Value)
		}
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("flow request failed (%d): %s", resp.StatusCode, string(respBody))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parse flow response: %v", err)
	}

	return result, nil
}

func getFlowToken(result map[string]interface{}) (string, error) {
	ft, ok := result["flow_token"].(string)
	if !ok {
		return "", fmt.Errorf("flow_token not found in response")
	}
	return ft, nil
}

func (a *TwitterAuth) initFlow() (string, error) {
	// First, get guest token
	req, err := http.NewRequest("POST", activateURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+loginBearerToken)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("guest token request: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var gtResult struct {
		GuestToken string `json:"guest_token"`
	}
	json.Unmarshal(body, &gtResult)

	if gtResult.GuestToken != "" {
		twitterURL, _ := url.Parse(twitterAPIBase)
		a.httpClient.Jar.SetCookies(twitterURL, []*http.Cookie{
			{Name: "gt", Value: gtResult.GuestToken},
		})
	}

	payload := map[string]interface{}{
		"input_flow_data": map[string]interface{}{
			"flow_context": map[string]interface{}{
				"debug_overrides": map[string]interface{}{},
				"start_location":  map[string]interface{}{"location": "splash_screen"},
			},
		},
		"subtask_versions": map[string]interface{}{
			"action_list":                         2,
			"alert_dialog":                        1,
			"app_download_cta":                    1,
			"check_logged_in_account":             1,
			"choice_selection":                    3,
			"contacts_live_sync_permission_prompt": 0,
			"cta":                                 7,
			"email_verification":                  2,
			"end_flow":                            1,
			"enter_date":                          1,
			"enter_email":                         2,
			"enter_password":                      5,
			"enter_phone":                         2,
			"enter_recaptcha":                     1,
			"enter_text":                          5,
			"enter_username":                      2,
			"generic_urt":                         3,
			"in_app_notification":                 1,
			"interest_picker":                     3,
			"js_instrumentation":                  1,
			"menu_dialog":                         1,
			"notifications_permission_prompt":     2,
			"open_account":                        2,
			"open_home_timeline":                  1,
			"open_link":                           1,
			"phone_verification":                  4,
			"privacy_options":                     1,
			"security_key":                        3,
			"select_avatar":                       4,
			"select_banner":                       2,
			"settings_list":                       7,
			"show_code":                           1,
			"sign_up":                             2,
			"sign_up_review":                      4,
			"tweet_selection_urt":                 1,
			"update_users":                        1,
			"upload_media":                        1,
			"user_recommendations_list":           4,
			"user_recommendations_urt":            1,
			"wait_spinner":                        3,
			"web_modal":                           1,
		},
	}

	result, err := a.doFlowRequest(payload, "flow_name=login")
	if err != nil {
		return "", err
	}

	return getFlowToken(result)
}

func (a *TwitterAuth) submitInstrumentation(flowToken string) (string, error) {
	payload := map[string]interface{}{
		"flow_token": flowToken,
		"subtask_inputs": []map[string]interface{}{
			{
				"subtask_id":     "LoginJsInstrumentationSubtask",
				"js_instrumentation": map[string]interface{}{
					"response": "{}",
					"link":     "next_link",
				},
			},
		},
	}

	result, err := a.doFlowRequest(payload)
	if err != nil {
		return flowToken, nil // Non-critical, proceed
	}

	ft, err := getFlowToken(result)
	if err != nil {
		return flowToken, nil
	}
	return ft, nil
}

func (a *TwitterAuth) submitUsername(flowToken, username string) (string, error) {
	payload := map[string]interface{}{
		"flow_token": flowToken,
		"subtask_inputs": []map[string]interface{}{
			{
				"subtask_id": "LoginEnterUserIdentifierSSO",
				"settings_list": map[string]interface{}{
					"setting_responses": []map[string]interface{}{
						{
							"key":           "user_identifier",
							"response_data": map[string]interface{}{"text_data": map[string]interface{}{"result": username}},
						},
					},
					"link": "next_link",
				},
			},
		},
	}

	result, err := a.doFlowRequest(payload)
	if err != nil {
		return "", fmt.Errorf("username submission failed: %v", err)
	}

	return getFlowToken(result)
}

func (a *TwitterAuth) submitPassword(flowToken, password string) (string, error) {
	payload := map[string]interface{}{
		"flow_token": flowToken,
		"subtask_inputs": []map[string]interface{}{
			{
				"subtask_id": "LoginEnterPassword",
				"enter_password": map[string]interface{}{
					"password": password,
					"link":     "next_link",
				},
			},
		},
	}

	result, err := a.doFlowRequest(payload)
	if err != nil {
		return "", fmt.Errorf("password submission failed: %v", err)
	}

	return getFlowToken(result)
}

func (a *TwitterAuth) handleDuplicationCheck(flowToken string) (string, error) {
	payload := map[string]interface{}{
		"flow_token": flowToken,
		"subtask_inputs": []map[string]interface{}{
			{
				"subtask_id":   "AccountDuplicationCheck",
				"check_logged_in_account": map[string]interface{}{
					"link": "AccountDuplicationCheck_false",
				},
			},
		},
	}

	result, err := a.doFlowRequest(payload)
	if err != nil {
		return flowToken, nil
	}

	ft, err := getFlowToken(result)
	if err != nil {
		return flowToken, nil
	}
	return ft, nil
}

func (a *TwitterAuth) extractSession(username string) (*Session, error) {
	twitterURL, _ := url.Parse(twitterAPIBase)
	cookies := a.httpClient.Jar.Cookies(twitterURL)

	// Also check twitter.com domain
	twitterWebURL, _ := url.Parse(twitterLoginURL)
	cookies = append(cookies, a.httpClient.Jar.Cookies(twitterWebURL)...)

	session := &Session{
		Username:  username,
		CreatedAt: time.Now(),
		LastUsed:  time.Now(),
		Active:    true,
	}

	for _, cookie := range cookies {
		switch cookie.Name {
		case "auth_token":
			session.AuthToken = cookie.Value
		case "ct0":
			session.CT0 = cookie.Value
		}
	}

	if session.AuthToken == "" {
		return nil, fmt.Errorf("login failed: no auth_token cookie received. Check credentials or if account needs verification")
	}

	return session, nil
}

// ValidateSession checks if a session is still valid by making a test API call
func ValidateSession(session *Session) (bool, error) {
	req, err := http.NewRequest("GET", twitterAPIBase+"/1.1/account/verify_credentials.json", nil)
	if err != nil {
		return false, err
	}

	req.Header.Set("Authorization", "Bearer "+loginBearerToken)
	req.Header.Set("x-csrf-token", session.CT0)
	req.AddCookie(&http.Cookie{Name: "auth_token", Value: session.AuthToken})
	req.AddCookie(&http.Cookie{Name: "ct0", Value: session.CT0})

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	return resp.StatusCode == 200, nil
}
