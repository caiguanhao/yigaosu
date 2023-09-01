package yigaosu

import (
	"context"
	"os"
	"testing"
)

func Test(t *testing.T) {
	token := os.Getenv("TOKEN")
	if token == "" {
		t.Fatal("Set TOKEN env first")
	}
	client := Client{
		Token: token,
	}
	ctx := context.Background()
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
