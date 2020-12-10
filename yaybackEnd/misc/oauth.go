package misc

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"net/url"
	"strings"
)

func OauthSignature(method ,baseUrl , appSecret, userSecret string,params map[string]string, authParams map[string]string)(string,string){

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

	signatureBaseString := method+"&"+percentEncode(baseUrl)+"&"+percentEncode(encodedParamsString)

	signingKey := percentEncode(appSecret) + "&" + percentEncode(userSecret)

	hmacHash := hmac.New(sha1.New,[]byte(signingKey))

	_, _ = hmacHash.Write([]byte(signatureBaseString))

	b64code := base64.StdEncoding.EncodeToString(hmacHash.Sum(nil))

	/*log.Printf("final String : %v\n",encodedParamsString)
	log.Printf("signature String String : %v\n",signatureBaseString)
	log.Printf("signing key : %v\n",signingKey)

	log.Printf("signature : %v",b64code)
*/

	authParams["oauth_signature"] = b64code

	return b64code,GetAuthString(authParams)

}


func percentEncode(x string)string {
	return strings.Replace(url.QueryEscape(x),"+","%20",-1)
}

func encodeMap(encodedParams []string,params map[string]string) []string{
	for paramKey, paramValue := range params {

		v := url.Values{}
		v.Add(paramKey,paramValue)

		encodedParam := percentEncode(paramKey)+"="+percentEncode(paramValue)
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

func GetAuthString(oauthParams map[string]string)string{
	oauthHeader := "Oauth "

	for key, value := range oauthParams {
		oauthHeader += key+"=\""+percentEncode(value)+"\","
	}

	return strings.TrimSuffix(oauthHeader,",")

}

