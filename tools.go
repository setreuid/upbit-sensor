package main

import (
    "bufio"
    "bytes"
    "crypto/sha512"
    "crypto/tls"
    "encoding/csv"
    "encoding/hex"
    "encoding/json"
    "errors"
    "fmt"
    "github.com/dgrijalva/jwt-go/v4"
    "github.com/google/uuid"
    "gitlab.com/c0b/go-ordered-json"
    "golang.org/x/text/encoding/korean"
    "golang.org/x/text/transform"
    "io/ioutil"
    "log"
    "net/http"
    "net/url"
    "os"
    "strconv"
    "strings"
    "time"
)

func Delete(url string, data interface{}) (int, []byte, error) {
    return Request("DELETE", url, data)
}

func Post(url string, data interface{}) (int, []byte, error) {
    return Request("POST", url, data)
}

func Request(method string, url string, data interface{}) (int, []byte, error) {
    jsonString, err := json.Marshal(data)
    if err != nil {
        return 0, nil, err
    }

    var om *ordered.OrderedMap = ordered.NewOrderedMap()
    err = json.Unmarshal(jsonString, om)
    if err != nil {
        return 0, nil, err
    }

    values := Map2UrlParams(om)

    tr := &http.Transport{
        TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
    }
    client := http.Client{
        Timeout:   time.Duration(2) * time.Second,
        Transport: tr,
    }

    req, err := http.NewRequest(method, url, bytes.NewBuffer(jsonString))
    if err != nil {
        return 0, nil, err
    }

    jToken, err := MakeToken(values)
    if err != nil {
        return 0, nil, err
    }

    req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", jToken))
    req.Header.Add("Content-type", "application/json")

    resp, err := client.Do(req)
    if err != nil {
        return 0, nil, err
    }

    defer func() {
        resp.Body.Close()
        client.CloseIdleConnections()
    }()

    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        return 0, nil, err
    }

    return resp.StatusCode, body, nil
}

func Get(url string, params string) (int, []byte, error) {
    tr := &http.Transport{
        TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
    }
    client := http.Client{
        Timeout:   time.Duration(2) * time.Second,
        Transport: tr,
    }

    req, err := http.NewRequest("GET", fmt.Sprintf("%s?%s", url, params), nil)
    if err != nil {
        return 0, nil, err
    }

    jToken, err := MakeToken(params)
    if err != nil {
        return 0, nil, err
    }
    req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", jToken))

    resp, err := client.Do(req)
    if err != nil {
        return 0, nil, err
    }

    defer func() {
        resp.Body.Close()
        client.CloseIdleConnections()
    }()

    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        return 0, nil, err
    }

    return resp.StatusCode, body, nil
}

func MakeToken(params string) (string, error) {
    return MakeTokenWithBinary([]byte(params))
}

// 여기서 API 키를 발급받으세요.
// https://upbit.com/mypage/open_api_management
func MakeTokenWithBinary(params []byte) (string, error) {
    at := AuthTokenClaims{
        AccessKey: "Access key", // 여기랑
        Nonce: uuid.NewString(),
    }

    if params != nil && len(params) > 0 {
        at.QueryHash = DigestSHA512Binary(params)
        at.QueryHashAlg = "SHA512"
    }

    aToken := jwt.NewWithClaims(jwt.SigningMethodHS256, &at)
    return aToken.SignedString([]byte("Secret key")) // 여기
}

func DigestSHA512Binary(s []byte) string {
    h := sha512.New()
    h.Write(s)
    return hex.EncodeToString(h.Sum(nil))
}

func Map2UrlParams(om *ordered.OrderedMap) string {
    var result []string
    iter := om.EntriesIter()
    for {
        pair, ok := iter()
        if !ok {
            break
        }

        key := pair.Key
        element := pair.Value

        if element == nil || element == "" {
            element = ""
        }
        result = append(result, fmt.Sprintf("%s=%s", key, url.QueryEscape(fmt.Sprintf("%s", element))))
    }
    return strings.Join(result, "&")
}

func GetTodayString() string {
    currentTime := time.Now()
    return currentTime.Format("2006-01-02")
}

func GetTimeString() string {
    currentTime := time.Now()
    return currentTime.Format("15:04:05")
}

func LogCsv(code string, isBuy bool, uses float64, percent int, percentLts int, assetEarn float64, assetEarnPercent float64, asset float64, earnPercent float64, remark string,) {
    src := fmt.Sprintf("logs/%s.csv", GetTodayString())
    exist := true

    if _, err := os.Stat(src); errors.Is(err, os.ErrNotExist) {
        exist = false
    }

    if file, err := os.OpenFile(src, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600); err == nil {
        defer file.Close()

        wr := csv.NewWriter(bufio.NewWriter(file))

        if !exist {
            wr.Write([]string{
                ToEucKr("체결일자"),
                ToEucKr("체결시간"),
                ToEucKr("코인"),
                ToEucKr("구분"),
                ToEucKr("체결금액"),
                ToEucKr("매수강도"),
                ToEucKr("매수강도(장기적인)"),
                ToEucKr("이익율"),
                ToEucKr("1시간 내 총 이익"),
                ToEucKr("1시간 내 총 이익율"),
                ToEucKr("총 보유자산"),
                ToEucKr("비고"),
            })
            wr.Flush()
        }

        buy := "매수"
        if !isBuy {
            buy = "매도"
        }

        wr.Write([]string{
            GetTodayString(),
            GetTimeString(),
            code,
            ToEucKr(buy),
            fmt.Sprintf("%.2f", uses),
            fmt.Sprintf("%d%%", percent),
            fmt.Sprintf("%d%%", percentLts),
            fmt.Sprintf("%.2f%%", earnPercent),
            fmt.Sprintf("%.2f", assetEarn),
            fmt.Sprintf("%.2f%%", assetEarnPercent),
            fmt.Sprintf("%.2f", asset),
            ToEucKr(remark),
        })
        wr.Flush()
    } else {
        log.Println(err)
    }
}

func ToEucKr(str string) string {
    var bufs bytes.Buffer

    wr := transform.NewWriter(&bufs, korean.EUCKR.NewEncoder())
    wr.Write([]byte(str))
    wr.Close()

    wr = nil

    return bufs.String()
}

func ToFloat64(str string) float64 {
    fc, err := strconv.ParseFloat(str, 64)
    if err != nil {
        return 0
    }
    return fc
}
