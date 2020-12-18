package helpers

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func OauthSignature(method ,baseUrl , appSecret, userSecret string,params url.Values, authParams url.Values)(string,string){

	var encodedParams []string

	encodedParams = encodeMap(encodedParams,params)
	encodedParams = encodeMap(encodedParams,authParams)

	encodedParamsString := ""

	for i, param := range encodedParams {
		if i > 0{
			encodedParamsString += "&"
		}
		encodedParamsString +=param
	}

	signatureBaseString := method+"&"+ percentEncode(baseUrl)+"&"+ percentEncode(encodedParamsString)

	signingKey := percentEncode(appSecret) + "&" + percentEncode(userSecret)

	hmacHash := hmac.New(sha1.New,[]byte(signingKey))

	_, _ = hmacHash.Write([]byte(signatureBaseString))

	b64code := base64.StdEncoding.EncodeToString(hmacHash.Sum(nil))

	/*log.Printf("final String : %v\n",encodedParamsString)
	log.Printf("signature String String : %v\n",signatureBaseString)
	log.Printf("signing key : %v\n",signingKey)

	log.Printf("signature : %v",b64code)
*/

	authParams["oauth_signature"] = []string{b64code}

	return b64code, GetAuthString(authParams)

}


func percentEncode(x string)string {
	return strings.Replace(url.QueryEscape(x),"+","%20",-1)
}

func encodeMap(encodedParams []string,params url.Values) []string{
	for paramKey, paramValue := range params {

		v := url.Values{}
		v.Add(paramKey,paramValue[0])

		encodedParam := percentEncode(paramKey)+"="+ percentEncode(paramValue[0])
		i := -1
		for j, param := range encodedParams {
			if encodedParam < param{
				i = j
				break
			}

		}

		if i >= 0 {
			encodedParams = append(encodedParams, "")
			copy(encodedParams[i+1:], encodedParams[i:])
			encodedParams[i] = encodedParam
		}else{
			encodedParams = append(encodedParams, encodedParam)

		}

	}

	return  encodedParams

}

func GetAuthString(oauthParams url.Values)string{
	oauthHeader := "OAuth "

	for key, value := range oauthParams {
		oauthHeader += key+"=\""+ percentEncode(value[0])+"\","
	}

	return strings.TrimSuffix(oauthHeader,",")

}

func GetAuthParams(params map[string]string) url.Values{
	oauthParams := url.Values{}
	if params != nil {
		for key, value := range params {
			oauthParams.Add(key,value)
		}
	}

	oauthParams.Add("oauth_consumer_key", "auth2.TwitterApiKey")
	oauthParams.Add("oauth_nonce", strconv.FormatInt(time.Now().Unix(), 10))
	oauthParams.Add("oauth_version", "1.0")
	oauthParams.Add("oauth_signature_method", "HMAC-SHA1")
	oauthParams.Add("oauth_version", "1.0")
	oauthParams.Add("oauth_timestamp", strconv.FormatInt(time.Now().Unix(), 10))

	return oauthParams
}

