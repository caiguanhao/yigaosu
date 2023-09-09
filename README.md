# yigaosu

```go
ctx := context.Background()
client, err := yigaosu.Login(ctx, "12345678901", "encrypted-password")
cards, err := client.GetETCCards(ctx)
bills, err := client.GetETCCardBillsPage(ctx, cards[0], 1, 1)
```

You can use Mitmproxy to obtain the encrypted password when using the Yigaosu
(e高速) app.
