package network

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"net/url"
	"midas/configuration"
	"strconv"
	"time"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

var apiKey = configuration.ReadBrainConfig().API_KEY
var apiSecret = configuration.ReadBrainConfig().API_SECRET

func NewHttpRequest(
	reqType string,
	reqUrl string,
	reqData url.Values,
	useApiKey bool,
	useSignature bool) ([]byte, error) {

	if reqData == nil {
		reqData = url.Values{}
	}

	if useSignature {
		reqData.Set("recvWindow", "6000000")
		tonce := strconv.FormatInt(time.Now().UnixNano(), 10)[0:13]
		reqData.Set("timestamp", tonce)
		payload := reqData.Encode()
		signature, err := getParamHmacSHA256Sign(apiSecret, payload)
		if err != nil {
			return nil, err
		}
		reqData.Set("signature", signature)
	}

	req, _ := http.NewRequest(reqType, reqUrl, strings.NewReader(reqData.Encode()))

	//req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// TODO is this needed?
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 5.1) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/31.0.1650.63 Safari/537.36")

	if useApiKey {
		req.Header.Add("X-MBX-APIKEY", apiKey)
	}

	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	bodyData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, errors.New(fmt.Sprintf("HttpStatusCode:%d ,Desc:%s", resp.StatusCode, string(bodyData)))
	}

	return bodyData, nil
}

func getParamHmacSHA256Sign(secret, params string) (string, error) {
	mac := hmac.New(sha256.New, []byte(secret))
	_, err := mac.Write([]byte(params))
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(mac.Sum(nil)), nil
}

