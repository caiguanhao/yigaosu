package yigaosu

import (
	"context"
	"os"
	"strings"
	"testing"
)

func Test(t *testing.T) {
	login := os.Getenv("YIGAOSU_LOGIN")
	parts := strings.Split(login, ",")
	if len(parts) < 2 {
		t.Fatal("Set env first: YIGAOSU_LOGIN=PHONE,ENCRYPTED_PASSWORD")
	}
	ctx := context.Background()
	client, err := Login(ctx, parts[0], parts[1])
	if err != nil {
		t.Fatal(err)
	}
	t.Log("using access token:", client.AccessToken)
	cards, err := client.GetETCCards(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(cards) == 0 {
		t.Fatal("should have etc cards")
	}
	card := cards[0]
	t.Log(card)
	bills, err := client.GetETCCardBillsPage(ctx, card, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(bills) == 0 {
		t.Fatal("should have etc card bills")
	}
	t.Log(bills)
}
