package yigaosu

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type Client struct {
	Token string // JWT
}

type ETCCard struct {
	CardCode         string `json:"cardCode"`         // 1
	CardNo           string `json:"cardNo"`           // XXXXXXXXXXXXXXXXXXXX
	CardNoFormat     string `json:"cardNoFormat"`     // XXXX********XXXXXXXX
	CardType         string `json:"cardType"`         // 记账卡
	ETCCardStatus    string `json:"etcCardStatus"`    // 1
	ETCCardStatusMsg string `json:"etcCardStatusMsg"` // 正常使用
	PlateNo          string `json:"plateNo"`          // 粤XX
	ValidTime        int64  `json:"validTime"`        // 1973692800000
}

type etcCardListResponse struct {
	Code    int       `json:"code"`
	Message string    `json:"message"`
	Data    []ETCCard `json:"data"`
}

// Get list of ETC cards of current user.
func (c Client) GetETCCards(ctx context.Context) ([]ETCCard, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://weapi.yigaosu.com/etcFree/findEtcCardList", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", c.Token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var data etcCardListResponse
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}
	if data.Code != 200 {
		return nil, fmt.Errorf("failed to get ETC cards (error status %d)", data.Code)
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

	baseURL, _ := url.Parse("https://weapi.yigaosu.com/etcCard/etcBillList")
	params := url.Values{}
	params.Add("plateNo", card.PlateNo)
	params.Add("cardNo", card.CardNo)
	params.Add("cardType", card.CardType)
	params.Add("startDate", "0")
	params.Add("endDate", endDate)
	params.Add("limit", strconv.Itoa(limit))
	params.Add("page", strconv.Itoa(page))
	baseURL.RawQuery = params.Encode()
	req, err := http.NewRequestWithContext(ctx, "GET", baseURL.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", c.Token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var data etcCardBillListResponse
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}
	if data.Code != 200 {
		return nil, fmt.Errorf("failed to get ETC card bills (error status %d)", data.Code)
	}
	return data.Data.BillList, nil
}
