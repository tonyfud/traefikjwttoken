package traefikjwttoken

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"
	// "github.com/go-redis/redis/v8"
)

var ctx = context.Background()

type Config struct {
	Secret          string `json:"secret,omitempty"`
	ProxyHeaderName string `json:"proxyHeaderName,omitempty"`
	AuthHeader      string `json:"authHeader,omitempty"`
	HeaderPrefix    string `json:"headerPrefix,omitempty"`
}

func CreateConfig() *Config {
	return &Config{}
}

type JWT struct {
	next            http.Handler
	name            string
	secret          string
	proxyHeaderName string
	authHeader      string
	headerPrefix    string
}

func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {

	// rdb := redis.NewClient(&redis.Options{
	// 	Addr:     "localhost:6379",
	// 	Password: "", // no password set
	// 	DB:       0,  // use default DB
	// })

	// err := rdb.Set(ctx, "key", "value", 0).Err()
	// if err != nil {
	// 	panic(err)
	// }

	// val, err := rdb.Get(ctx, "key").Result()
	// if err != nil {
	// 	panic(err)
	// }
	// fmt.Println("key", val)

	if len(config.Secret) == 0 {
		config.Secret = "SECRET"
	}
	if len(config.ProxyHeaderName) == 0 {
		config.ProxyHeaderName = "injectedPayload"
	}
	if len(config.AuthHeader) == 0 {
		config.AuthHeader = "Authorization"
	}
	if len(config.HeaderPrefix) == 0 {
		config.HeaderPrefix = "Bearer"
	}

	return &JWT{
		next:            next,
		name:            name,
		secret:          config.Secret,
		proxyHeaderName: config.ProxyHeaderName,
		authHeader:      config.AuthHeader,
		headerPrefix:    config.HeaderPrefix,
	}, nil
}

func (j *JWT) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	headerToken := req.Header.Get(j.authHeader)

	if len(headerToken) == 0 {
		http.Error(res, "Request error", http.StatusBadRequest)
		return
	}

	token, preprocessError := preprocessJWT(headerToken, j.headerPrefix)
	if preprocessError != nil {
		http.Error(res, "Request error", http.StatusBadRequest)
		return
	}

	verified, verificationError := verifyJWT(token, j.secret)
	if verificationError != nil {
		http.Error(res, "Not allowed", http.StatusUnauthorized)
		return
	}

	if verified {
		// If true decode payload
		payload, decodeErr := decodeBase64(token.payload)
		if decodeErr != nil {
			http.Error(res, "Request error", http.StatusBadRequest)
			return
		}

		// TODO Check for outside of ASCII range characters
		Data := []byte(payload)

		fmt.Println("str payload : ----------> ", payload)
		var v map[string]interface{}
		err := json.Unmarshal(Data, &v)
		if err != nil {
			fmt.Println(err)
			panic(err)
		}

		fmt.Printf("str expiredate : ----------> %f , unixtime : %d \n", v["exp"], time.Now().UnixNano())
		fmt.Printf("str expiredate : ----------> %d , unixtime : %d \n", int(math.Floor(v["exp"].(float64))), time.Now().UnixNano()/1000000000)

		expiredate := int64(math.Floor(v["exp"].(float64)))

		if isExpire(expiredate) && err == nil {

			xType := fmt.Sprintf("expire Type : %T \n", v["exp"])
			fmt.Printf(xType)

			http.Error(res, "Token Expired", http.StatusBadRequest)
			return
		}

		// Inject header as proxypayload or configured name
		req.Header.Add(j.proxyHeaderName, payload)
		fmt.Println(req.Header)
		j.next.ServeHTTP(res, req)
	} else {
		http.Error(res, "Not allowed", http.StatusUnauthorized)
	}
}

func isExpire(ctime int64) bool {

	if ctime < (time.Now().UnixNano() / 1000000000) {
		return true
	}
	return false
}

// Token Deconstructed header token
type Token struct {
	header       string
	payload      string
	verification string
}

// verifyJWT Verifies jwt token with secret
func verifyJWT(token Token, secret string) (bool, error) {
	mac := hmac.New(sha256.New, []byte(secret))
	message := token.header + "." + token.payload
	mac.Write([]byte(message))
	expectedMAC := mac.Sum(nil)

	decodedVerification, errDecode := base64.RawURLEncoding.DecodeString(token.verification)
	if errDecode != nil {
		return false, errDecode
	}

	if hmac.Equal(decodedVerification, expectedMAC) {
		return true, nil
	}
	return false, nil
	// TODO Add time check to jwt verification
}

// preprocessJWT Takes the request header string, strips prefix and whitespaces and returns a Token
func preprocessJWT(reqHeader string, prefix string) (Token, error) {
	// fmt.Println("==> [processHeader] SplitAfter")
	// structuredHeader := strings.SplitAfter(reqHeader, "Bearer ")[1]
	cleanedString := strings.TrimPrefix(reqHeader, prefix)
	cleanedString = strings.TrimSpace(cleanedString)
	// fmt.Println("<== [processHeader] SplitAfter", cleanedString)

	var token Token

	tokenSplit := strings.Split(cleanedString, ".")

	if len(tokenSplit) != 3 {
		return token, fmt.Errorf("Invalid token")
	}

	token.header = tokenSplit[0]
	token.payload = tokenSplit[1]
	token.verification = tokenSplit[2]

	return token, nil
}

// decodeBase64 Decode base64 to string
func decodeBase64(baseString string) (string, error) {
	byte, decodeErr := base64.RawURLEncoding.DecodeString(baseString)
	if decodeErr != nil {
		return baseString, fmt.Errorf("Error decoding")
	}
	return string(byte), nil
}
