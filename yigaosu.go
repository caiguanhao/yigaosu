package yigaosu

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	AccessToken string
}

type loginResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    *struct {
		AccessToken string `json:"access_token"`
	} `json:"data"`
}

func Login(ctx context.Context, phone, encryptedPassword string) (*Client, error) {
	data := url.Values{
		"appVersion":     {"5.2.7"},
		"client_id":      {"00000000-0000-0000-0000-000000000000"},
		"login_name":     {phone},
		"login_password": {encryptedPassword},
		"model":          {"iPhone 13"},
		"os":             {"ios"},
		"osVersion":      {"16.6"},
		"state":          {"1"},
		"type":           {"cipher"},
	}
	req, err := http.NewRequestWithContext(ctx, "POST", "https://eapi.yigaosu.com/login/loginByPassWord", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	debugRequest(ctx, req)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	debugResponse(ctx, resp, body)
	if err != nil {
		return nil, err
	}
	var ret loginResponse
	err = json.Unmarshal(body, &ret)
	if err != nil {
		return nil, err
	}
	if ret.Code != 200 {
		return nil, fmt.Errorf("failed to login (error status: %d, message: %s)", ret.Code, ret.Message)
	}
	if ret.Data == nil {
		return nil, fmt.Errorf("no login data found")
	}
	return &Client{
		AccessToken: ret.Data.AccessToken,
	}, nil
}

type ETCCard struct {
	CardCode string `json:"cardCode"` // 1
	CardNo   string `json:"cardNo"`   // XXXXXXXXXXXXXXXXXXXX
	CardType string `json:"cardType"` // 记账卡
	PlateNo  string `json:"plateNo"`  // 粤XX
}

type etcCardListResponse struct {
	Code    int       `json:"code"`
	Message string    `json:"message"`
	Data    []ETCCard `json:"data"`
}

// Get list of ETC cards of current user.
func (c Client) GetETCCards(ctx context.Context) ([]ETCCard, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", "https://eapi.yigaosu.com/etcCard/plateNo", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("access_token", c.AccessToken)
	debugRequest(ctx, req)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	debugResponse(ctx, resp, body)
	if err != nil {
		return nil, err
	}
	var data etcCardListResponse
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}
	if data.Code != 200 {
		return nil, fmt.Errorf("failed to get ETC cards (error status: %d, message: %s)", data.Code, data.Message)
	}
	return data.Data, nil
}

type ETCCardBill struct {
	Amount           string `json:"amount"`           // 19.0
	Bank             string `json:"bank"`             // XX银行(XX)
	BillId           string `json:"billid"`           // T20230901XXXXXXXXXXXXXXXXXXXX
	BillName         string `json:"billName"`         // ETC通行费
	BillTitle        string `json:"billTitle"`        // XX站驶入-XX站驶出
	BillType         string `json:"billType"`         // etc
	CardType         string `json:"cardType"`         // 记账卡
	EndStation       string `json:"endStation"`       // XX站驶出
	EndTime          int64  `json:"endTime"`          // 1693480225000
	PayTime          int64  `json:"payTime"`          // 1693480225000
	PlateNo          string `json:"plateNo"`          // 粤XX
	RefundFailReason string `json:"refundFailReason"` // 已受理,支付中
	RefundState      int    `json:"refundState"`      // 3
	StartStation     string `json:"startStation"`     // XX站驶入
	StartTime        int64  `json:"startTime"`        // 1693479229000
	TotalAmount      string `json:"totalAmount"`      // 19
	WasteId          string `json:"wasteId"`          // G00944400XXXXXXXXXXXXXXXXXXXXXXXXXXXX
	WasteType        int    `json:"wasteType"`        // 1
}

type etcCardBillListResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		BillList []ETCCardBill `json:"billList"`
	} `json:"data"`
}

// Get bills of ETC card. Limit should be a number from 1 to 200.
func (c Client) GetETCCardBillsPage(ctx context.Context, card ETCCard, limit, page int) ([]ETCCardBill, error) {
	now := time.Now()
	y, m, _ := now.Date()
	endDate := fmt.Sprintf("%d000", time.Date(y, m+1, 1, 0, 0, 0, 0, now.Location()).Add(-1*time.Second).Unix())

	baseURL, _ := url.Parse("https://eapi.yigaosu.com/etcCard/etcBillList")
	params := url.Values{}
	params.Add("plateNo", card.PlateNo)
	params.Add("cardNo", card.CardNo)
	params.Add("cardType", card.CardCode)
	params.Add("startDate", "0")
	params.Add("endDate", endDate)
	params.Add("limit", strconv.Itoa(limit))
	params.Add("page", strconv.Itoa(page))
	baseURL.RawQuery = params.Encode()
	req, err := http.NewRequestWithContext(ctx, "GET", baseURL.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("access_token", c.AccessToken)
	debugRequest(ctx, req)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	debugResponse(ctx, resp, body)
	if err != nil {
		return nil, err
	}
	var data etcCardBillListResponse
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}
	if data.Code != 200 {
		return nil, fmt.Errorf("failed to get ETC card bills (error status: %d, message: %s)", data.Code, data.Message)
	}
	return data.Data.BillList, nil
}

func debugRequest(ctx context.Context, req *http.Request) {
	if !isDebug(ctx) {
		return
	}
	dump, err := httputil.DumpRequestOut(req, true)
	if err != nil {
		return
	}
	scanner := bufio.NewScanner(bytes.NewReader(dump))
	for scanner.Scan() {
		fmt.Fprintln(os.Stderr, "[DEBUG]", scanner.Text())
	}
}

func debugResponse(ctx context.Context, resp *http.Response, body []byte) {
	if !isDebug(ctx) {
		return
	}
	dump, err := httputil.DumpResponse(resp, false)
	if err != nil {
		return
	}
	scanner := bufio.NewScanner(bytes.NewReader(dump))
	for scanner.Scan() {
		fmt.Fprintln(os.Stderr, "[DEBUG]", scanner.Text())
	}
	if len(body) > 1024 {
		fmt.Fprintln(os.Stderr, "[DEBUG]", string(body[:1024]), "...")
	} else {
		fmt.Fprintln(os.Stderr, "[DEBUG]", string(body))
	}
}

func isDebug(ctx context.Context) bool {
	debug, ok := ctx.Value("DEBUG").(bool)
	return ok && debug
}
