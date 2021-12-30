package main

import (
    "encoding/json"
    "fmt"
    "github.com/cornelk/hashmap"
    "log"
    "math"
    "strconv"
    "time"
)

var (
    wallet = &hashmap.HashMap{}
)

var (
    Info = Teal
    Warn = Yellow
    Fata = Red
)

var (
    Black   = Color("\033[1;30m%s\033[0m")
    Red     = Color("\033[1;31m%s\033[0m")
    Green   = Color("\033[1;32m%s\033[0m")
    Yellow  = Color("\033[1;33m%s\033[0m")
    Purple  = Color("\033[1;34m%s\033[0m")
    Magenta = Color("\033[1;35m%s\033[0m")
    Teal    = Color("\033[1;36m%s\033[0m")
    White   = Color("\033[1;37m%s\033[0m")
)


func buyProcess() {
    for RUNNING {
        cs := <- cBuy
        code := cs.Code
        //strong := cs.Percent

        priceWon := cs.Price
        if priceWon < 1 {
            continue
        }

        var v interface{}
        var ok bool
        if v, ok = wallet.Get(code); !ok {
            v = &TradeWallet{
                Code: code,
                Locked: false,
            }
            wallet.Set(code, v)
        }

        f := 1.0
        // f := (float64(cs.Percent) - 45) / 15.0 // 매수 강도에 따른 코인 갯수 배율 (이 코드로 대체하면 강도가 쎌때 많이 매수합니다)

        count := minCount(code) * f * 1.1
        pw := count * priceWon * (1 + 0.0005)
        remain := getRemain("KRW")
        if remain < pw {
            continue
        }

        w := v.(*TradeWallet)
        if w.Locked {
            continue
        }

        w.Locked = true
        tradeLock = true
        tradeMtx.Lock()

        otype := ORD_TYPE // ORD_TYPE

        params := OrderBody{
            Market: code,
            Side: "bid",
            Volume: nil,
            Price: nil,
            OrdType: map[bool]string{true: "limit", false: "price"}[otype == "limit"] ,
        }

        if otype == "limit" {
            vol := fmt.Sprintf("%.8g", count)
            pri := fmt.Sprintf("%.8g", priceWon)
            params.Volume = &vol
            params.Price = &pri
        } else {
            pri := fmt.Sprintf("%.8g", count * priceWon)
            params.Price = &pri
        }

        status, body, err := Post("https://api.upbit.com/v1/orders", params)

        if status == 201 && err == nil {
            response := &OrderResponse{}
            if err := json.Unmarshal(body, response); err != nil {
                w.Locked = false
                tradeLock = false
                tradeMtx.Unlock()
                continue
            }

            orderBuy.Set(response.Uuid, OrderWait{
                Code: code,
                Count: count,
                UsedAmount: pw,
                Detail: response,
                Strong: cs,
            })

            w.Count += count
            w.UsedAmount -= pw
            w.Price = priceWon

            log.Println(Red(fmt.Sprintf("%s | 매수 %s %s개 (%s원)", printWallet(), code,  Format(int64(count)), Format(int64(priceWon)))))
        } else {
            w.Locked = false
            log.Printf("오류: 매수 - %v+", string(body))
        }

        lastTrade = time.Now()
        tradeLock = false
        tradeMtx.Unlock()
    }
}

func sellProcess() {
    for RUNNING {
        cs := <- cSell
        code := cs.Code

        priceWon := cs.Price
        if priceWon < 1 {
            continue
        }

        var v interface{}
        var ok bool

        if v, ok = wallet.Get(code); !ok {
            v = &TradeWallet{
                Code: code,
                Locked: false,
            }
            wallet.Set(code, v)
        }

        w := v.(*TradeWallet)
        if w.Locked {
            continue
        }

        count := minCount(code)

        if w.Count < count || w.Count < count * 1.95 {
            count = w.Count
        }

        if count * priceWon < minTotal(code) {
            continue
        }

        w.Locked = true
        tradeLock = true
        tradeMtx.Lock()

        otype := ORD_TYPE // ORD_TYPE

        params := OrderBody{
            Market: code,
            Side: "ask",
            Volume: nil,
            Price: nil,
            OrdType: map[bool]string{true: "limit", false: "market"}[otype == "limit"] ,
        }

        var vol string
        if otype == "limit" {
            vol = fmt.Sprintf("%.8f", count)
            pri := fmt.Sprintf("%.8f", priceWon)
            params.Volume = &vol
            params.Price = &pri
        } else {
            vol = fmt.Sprintf("%.8f", count)
            params.Volume = &vol
        }

        if f, err := strconv.ParseFloat(vol, 64); f == 0.0 || err != nil {
            w.Locked = false
            tradeLock = false
            tradeMtx.Unlock()
            continue
        }

        status, body, err := Post("https://api.upbit.com/v1/orders", params)

        if status == 201 && err == nil {
            response := &OrderResponse{}
            if err := json.Unmarshal(body, response); err != nil {
                w.Locked = false
                tradeLock = false
                tradeMtx.Unlock()
                continue
            }

            pw := count * priceWon * (1 - 0.0005)
            orderSell.Set(response.Uuid, OrderWait{
                Code:       code,
                Count:      count,
                UsedAmount: pw,
                Detail:     response,
                Strong:     cs,
            })

            w.Count -= count
            w.UsedAmount += pw
            log.Println(Green(fmt.Sprintf("%s | 매도 %s %s개 (%s원)", printWallet(), code, Format(int64(count)), Format(int64(priceWon)))))
        } else {
            w.Locked = false
            log.Printf("오류: 매도 %.8f개 - %v+", count, string(body))
        }

        lastTrade = time.Now()
        tradeLock = false
        tradeMtx.Unlock()
    }
}

func getPrice(code string) float64 {
    if v, ok := price.Get(code); ok {
        w := v.(float64)
        return w
    }
    return 0
}

func getAskPrice(code string) float64 {
    if v, ok := price.Get(code); ok {
        w := v.(float64)
        return w
    }
    return 0
}

func getWalletRemain(code string) float64 {
    accMtx.Lock()
    defer accMtx.Unlock()
    if v, ok := wallet.Get(code); ok {
        if v.(*TradeWallet).Locked {
            return 0
        }
        return v.(*TradeWallet).Count
    }
    return 0
}

func getEarnPercent(code string) float64 {
    accMtx.Lock()
    defer accMtx.Unlock()
    if v, ok := wallet.Get(code); ok {
        t := v.(*TradeWallet)
        if t.Locked {
            return 0
        }
        if t.Count == 0 {
            return 0
        }
        return (1 - (t.Price) / getPrice(code)) * 100.0
    }
    return 0
}

func getEarnPercentWithPrice(code string, pr float64) float64 {
    accMtx.Lock()
    defer accMtx.Unlock()
    if v, ok := wallet.Get(code); ok {
        t := v.(*TradeWallet)
        if t.Locked {
            return 0
        }
        if t.Count == 0 {
            return 0
        }
        return (1 - (t.Price) / pr) * 100.0
    }
    return 0
}

func getRemain(code string) float64 {
    accMtx.Lock()
    defer accMtx.Unlock()
    if v, ok := account.Get(code); ok {
        b := v.(Account).Balance
        fb, err := strconv.ParseFloat(b, 64)
        if err != nil {
            return 0
        }
        return fb
    }
    return 0
}

func getRemainLocked(code string) float64 {
    accMtx.Lock()
    defer accMtx.Unlock()
    if v, ok := account.Get(code); ok {
        b := v.(Account).Locked
        fb, err := strconv.ParseFloat(b, 64)
        if err != nil {
            return 0
        }
        return fb
    }
    return 0
}

func Color(colorString string) func(...interface{}) string {
    sprint := func(args ...interface{}) string {
        return fmt.Sprintf(colorString,
            fmt.Sprint(args...))
    }
    return sprint
}

func printWallet() string {
    var totalCount float64 = 0
    var totalAmount float64 = 0
    var willAmount float64 = 0
    accMtx.Lock()
    for v := range wallet.Iter() {
        w := v.Value.(*TradeWallet)

        priceWon := getPrice(w.Code)
        if priceWon < 1 {
            continue
        }

        totalCount += w.Count
        totalAmount += w.UsedAmount
        willAmount += w.UsedAmount + w.Count * priceWon
    }
    accMtx.Unlock()

    totalAmount += getRemain("KRW")

    elapsed := float64(time.Now().Sub(startTime)) / float64(time.Second)
    return fmt.Sprintf("[%08d] 코인 %s개, 현재 %s원, 손익 %s원", int32(elapsed), Format(int64(totalCount)), Format(int64(totalAmount)), Format(int64(willAmount)))
}

func tempUsedAmount() float64 {
    accMtx.Lock()
    defer accMtx.Unlock()
    var totalAmount float64 = 0
    for v := range wallet.Iter() {
        w := v.Value.(*TradeWallet)

        priceWon := getPrice(w.Code)
        if priceWon < 1 {
            continue
        }

        totalAmount += w.UsedAmount
    }

    return totalAmount
}

func Format(n int64) string {
    in := strconv.FormatInt(n, 10)
    numOfDigits := len(in)
    if n < 0 {
        numOfDigits-- // First character is the - sign (not a digit)
    }
    numOfCommas := (numOfDigits - 1) / 3

    out := make([]byte, len(in)+numOfCommas)
    if n < 0 {
        in, out[0] = in[1:], '-'
    }

    for i, j, k := len(in)-1, len(out)-1, 0; ; i, j = i-1, j-1 {
        out[j] = in[i]
        if i == 0 {
            return string(out)
        }
        if k++; k == 3 {
            j, k = j-1, 0
            out[j] = ','
        }
    }
}

func minCount(code string) float64 {
    cc, ok := chance.Get(code)
    if !ok {
        return 0
    }

    fcc := cc.(*OrderChance).Market.Ask.MinTotal
    fc, err := strconv.ParseFloat(fcc, 64)
    if err != nil {
        return 0
    }

    pr := getPrice(code)

    return math.Ceil((fc * VAR_TC / pr) * 100000000.0) / 100000000.0
}

func minTotal(code string) float64 {
    cc, ok := chance.Get(code)
    if !ok {
        return 0
    }

    fcc := cc.(*OrderChance).Market.Ask.MinTotal
    fc, err := strconv.ParseFloat(fcc, 64)
    if err != nil {
        return 0
    }

    return fc
}

func checkEarnStats() {
    elapsed := float64(time.Now().Sub(statTime)) / float64(time.Minute)
    if elapsed < float64(KAKAO_MIN) { // getWalletRemain(strong.Code) < 1 &&
        return
    }

    statTime = time.Now()

    initAccount()
    nowAsset := getAsset()
    earn := nowAsset - lastAsset
    earnPercent := (earn / lastAsset) * 100.0
    message  := fmt.Sprintf("지난 1시간동안의 매매 결과 안내입니다.\n\n매수 : %s건 / 매도 : %s건\n이익 : %s원 (%.2f%%)\n\n총 보유자산 : %s원", Format(int64(buyCount)), Format(int64(sellCount)), Format(int64(earn)), earnPercent, Format(int64(nowAsset)))

    lastAsset = nowAsset
    buyCount = 0
    sellCount = 0

    log.Println(message)
    // 문자나 슬랙등으로 1시간 매매 결과 안내를 받고 싶으신 경우
    // 여기에서 처리하시면 됩니다.
}

func getAsset() float64 {
    accMtx.Lock()
    var sum float64 = 0
    for v := range wallet.Iter() {
        code := v.Key.(string)
        w := v.Value.(*TradeWallet)

        sum += w.Count * getPrice(code)
    }
    accMtx.Unlock()
    return sum + getRemain("KRW") + getRemainLocked("KRW")
}
