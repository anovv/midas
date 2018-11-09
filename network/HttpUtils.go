package network

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
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
	reqData map[string]string,
	useApiKey bool,
	useSignature bool) ([]byte, error) {

	transport := &http.Transport{}
	client := &http.Client{
		Transport: transport,
	}

	req, err := http.NewRequest(reqType, reqUrl, nil)
	if err != nil {
		return nil, err
	}

	if reqData == nil {
		reqData = make(map[string]string)
	}

	if useApiKey {
		req.Header.Add("X-MBX-APIKEY", apiKey)
	}

	if useSignature {
		// TODO add mutex on ts
		reqData["recvWindow"] = "600000"
		if _, hasTs := reqData["timestamp"]; !hasTs {
			tonce := strconv.FormatInt(time.Now().UnixNano(), 10)[0:13]
			reqData["timestamp"] = tonce
		}
	}

	q := req.URL.Query()
	for key, val := range reqData {
		q.Add(key, val)
	}

	if useSignature {
		payload := q.Encode()
		signature, err := getParamHmacSHA256Sign(apiSecret, payload)
		if err != nil {
			return nil, err
		}
		q.Add("signature", signature)
	}

	req.URL.RawQuery = q.Encode()

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

