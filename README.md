# 🔑 ETH Auth for PocketBase

Hey! This is a simple way to let users log into your PocketBase app using their Ethereum wallet (MetaMask).

⚠️ **Heads up**: This is experimental stuff - don't use it in production without proper testing!

## What it does

- Sign in with MetaMask
- Verifies Ethereum signatures
- Lets users update their username
- Handles auth tokens automatically

## Quick setup

1. Get the code and dependencies:
```bash
git clone <your-repo>
cd pb-eth-auth
go mod init pb-eth-auth
go mod tidy
```

2. Set up PocketBase:
- Create a "users" collection
- Add an "address" field (text, required, unique)
- Set these collection rules:
  ```
  View rule:   id = @request.auth.id
  Update rule: id = @request.auth.id
  ```

3. Run it:
```bash
go run *.go serve
python3 -m http.server 8000  # In another terminal
```

4. Try it out: http://localhost:8000/test.html

## License

MIT License