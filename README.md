# yigaosu

```go
client := Client{
	Token: "ey...",
}
ctx := context.Background()
cards, err := client.GetETCCards(ctx)
bills, err := client.GetETCCardBillsPage(ctx, cards[0], 1, 1)
```

You can use Mitmproxy to obtain the JWT token when visiting the Yigaosu (e高速)
WeChat mini-program.
