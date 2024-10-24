package main

import (
    "encoding/hex"
    "fmt"

    "strings"

    "github.com/ethereum/go-ethereum/common"
    "github.com/ethereum/go-ethereum/crypto"
    "github.com/labstack/echo/v5"
    "github.com/pocketbase/pocketbase"
    "github.com/pocketbase/pocketbase/apis"
    "github.com/pocketbase/pocketbase/core"
    "github.com/pocketbase/pocketbase/models"
    "github.com/pocketbase/pocketbase/tokens"
)

type EthAuthRequest struct {
    Address   string `json:"address"`
    Signature string `json:"signature"`
    Message   string `json:"message"`
}

type EthAuthResponse struct {
    Token string         `json:"token"`
    User  *models.Record `json:"user"`
}

func registerEthereumAuth(app *pocketbase.PocketBase) {
    app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
        e.Router.AddRoute(echo.Route{
            Method: "POST",
            Path:   "/api/eth-auth",
            Handler: func(c echo.Context) error {
                return handleEthAuth(app, c)
            },
            Middlewares: []echo.MiddlewareFunc{
                apis.ActivityLogger(app),
            },
        })
        return nil
    })
}

func handleEthAuth(app *pocketbase.PocketBase, c echo.Context) error {
    var req EthAuthRequest
    if err := c.Bind(&req); err != nil {
        return apis.NewBadRequestError("Invalid request data", err)
    }

    // Validate address format
    if !common.IsHexAddress(req.Address) {
        return apis.NewBadRequestError("Invalid Ethereum address", nil)
    }

    // Convert address to checksum format
    address := common.HexToAddress(req.Address).Hex()

    // Verify signature
    verified, err := verifySignature(req.Message, req.Signature, address)
    if err != nil {
        return apis.NewBadRequestError("Failed to verify signature", err)
    }
    if !verified {
        return apis.NewBadRequestError("Invalid signature", nil)
    }

    // Find or create user
    user, err := findOrCreateUser(app, address)
    if err != nil {
        return apis.NewBadRequestError("Failed to process user", err)
    }

    // Generate auth token
    token, err := tokens.NewRecordAuthToken(app, user)
    if err != nil {
        return apis.NewBadRequestError("Failed to create auth token", err)
    }

    return c.JSON(200, EthAuthResponse{
        Token: token,
        User:  user,
    })
}

func verifySignature(message, signature, address string) (bool, error) {
    // Remove "0x" prefix if present
    signature = strings.TrimPrefix(signature, "0x")

    // Decode signature
    sigBytes, err := hex.DecodeString(signature)
    if err != nil {
        return false, fmt.Errorf("failed to decode signature: %v", err)
    }

    // Add Ethereum message prefix
    prefixedMessage := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(message), message)
    hash := crypto.Keccak256Hash([]byte(prefixedMessage))

    // Fix v value in signature
    if sigBytes[64] >= 27 {
        sigBytes[64] -= 27
    }

    // Recover public key
    pubKey, err := crypto.SigToPub(hash.Bytes(), sigBytes)
    if err != nil {
        return false, fmt.Errorf("failed to recover public key: %v", err)
    }

    // Derive Ethereum address from public key
    recoveredAddr := crypto.PubkeyToAddress(*pubKey).Hex()

    return strings.EqualFold(recoveredAddr, address), nil
}

func findOrCreateUser(app *pocketbase.PocketBase, address string) (*models.Record, error) {
    collection, err := app.Dao().FindCollectionByNameOrId("users")
    if err != nil {
        return nil, fmt.Errorf("failed to find users collection: %v", err)
    }

    // Try to find existing user
    user, _ := app.Dao().FindFirstRecordByData("users", "address", strings.ToLower(address))
    if user != nil {
        return user, nil
    }

    // Create new user
    newUser := models.NewRecord(collection)
    newUser.Set("address", strings.ToLower(address))
    newUser.Set("username", strings.ToLower(address)) // Required for auth collection
    newUser.Set("emailVisibility", false)

    if err := app.Dao().SaveRecord(newUser); err != nil {
        return nil, fmt.Errorf("failed to create user: %v", err)
    }

    return newUser, nil
}