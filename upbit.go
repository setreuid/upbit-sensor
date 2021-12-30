package main

import (
    "encoding/json"
    "flag"
    "fmt"
    "github.com/gorilla/websocket"
    "log"
    "os"
    "os/signal"
    "sync"
    "time"
)

func process() {
    for RUNNING {
        flag.Parse()
        log.SetFlags(0)

        interrupt := make(chan os.Signal, 1)
        signal.Notify(interrupt, os.Interrupt)

        c, _, err := websocket.DefaultDialer.Dial("wss://api.upbit.com:443/websocket/v1", nil)
        if err != nil {
            log.Fatal("dial:", err)
        }
        defer c.Close()

        done := make(chan struct{})

        go func() {
            defer close(done)
            for RUNNING {
                _, message, err := c.ReadMessage()
                if err != nil {
                    log.Println("read:", err)
                    return
                }
                // log.Printf("recv: %s", message)

                result := &TradeInfo{}
                if err := json.Unmarshal(message, result); err == nil {
                    cTicker <- result
                }
            }
        }()

        // Request trade tickers
        var codeString = ""
        for _, code := range VAR_CODES {
            codeString = fmt.Sprintf("%s,\"%s\"", codeString, code)
        }

        if len(codeString) > 0 {
            codeString = codeString[1:]
        }

        var packetString = "[{\"ticket\":\"UNIQUE_TICKET\"},{\"type\":\"trade\",\"codes\":[" + codeString + "]}]"
        c.WriteMessage(websocket.TextMessage, []byte(packetString))

        R_RUN := true
        for R_RUN {
            select {
            case <-done:
                R_RUN = false
                continue
            case <-interrupt:
                log.Println("interrupt")

                err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
                if err != nil {
                    log.Println("write close:", err)
                    R_RUN = false
                    continue
                }
                select {
                case <-done:
                case <-time.After(time.Second):
                }
                R_RUN = false
                continue
            }
        }
    }
}

func tradeInfoProcess() {
    for RUNNING {
        ti := <-cTicker

        var t interface{}
        var c []*StatusInfo // 기본
        var l []*StatusInfo // 장타
        var ok bool

        if t, ok = status.Get(ti.Code); !ok {
            c = []*StatusInfo{}
            status.Set(ti.Code, c)
            ticks.Set(ti.Code, int64(0))
        } else {
            c = t.([]*StatusInfo)
        }

        if t, ok = lts.Get(ti.Code); !ok {
            l = []*StatusInfo{}
            lts.Set(ti.Code, l)
        } else {
            l = t.([]*StatusInfo)
        }

        if t, ok = mutexes.Get(ti.Code); !ok {
            mm := &sync.Mutex{}
            mutexes.Set(ti.Code, mm)
        }

        t, _ = mutexes.Get(ti.Code)
        m := t.(*sync.Mutex)

        m.Lock()

        v := &StatusInfo{
            Code: ti.Code,
        }

        // v.Count += 1

        if ti.AskBid == "ASK" {
            v.Ask += ti.TradeVolume
            price.Set(ti.Code, ti.TradePrice)
        } else {
            v.Bid += ti.TradeVolume
            askPrice.Set(ti.Code, ti.TradePrice)
        }

        if int32(len(c)) >= VAR_TICKS[len(VAR_TICKS)-1] {
            c = c[1:]
        }

        if int32(len(l)) >= VAR_LTS[len(VAR_LTS)-1] {
            l = l[1:]
        }

        c = append(c, v)
        l = append(l, v)
        status.Set(ti.Code, c)
        lts.Set(ti.Code, l)

        if tksTemp, ok := ticks.Get(ti.Code); ok {
            tks := tksTemp.(int64)
            tks += 1
            ticks.Set(ti.Code, tks)

            if tks % DEF_TICK == 0 {
                cStrong <- ti.Code
            }
        }

        m.Unlock()
    }
}

func strongProcess() {
    for RUNNING {
        code := <-cStrong

        v, ok := status.Get(code)
        vl, okl := lts.Get(code)
        if ok && okl {
            m := v.([]*StatusInfo)
            mLts := vl.([]*StatusInfo)

            var nSum float32 = 0
            var dSum float32 = 0
            var success = true
            for _, t := range VAR_TICKS {
                length := int32(len(m))
                if length < t {
                    success = false
                } else {
                    dSum += 100 * float32(t)
                    var bidSum float64 = 0
                    var askSum float64 = 0
                    for i := length - 1; i >= length-t; i-- {
                        bidSum += m[i].Bid
                        askSum += m[i].Ask
                    }
                    percent := int(((bidSum - askSum) / (bidSum + askSum)) * 100.0)
                    nSum += float32(percent) * float32(t)
                    //nSum += float32(percent)
                }
            }

            var nSumLts float32 = 0
            var dSumLts float32 = 0
            for _, t := range VAR_LTS {
                length := int32(len(mLts))
                if length < t {
                    success = false
                } else {
                    dSumLts += 100 * float32(t)
                    var bidSum float64 = 0
                    var askSum float64 = 0
                    for i := length - 1; i >= length-t; i-- {
                        bidSum += mLts[i].Bid
                        askSum += mLts[i].Ask
                    }
                    percent := int(((bidSum - askSum) / (bidSum + askSum)) * 100.0)
                    nSumLts += float32(percent) * float32(t)
                    // nSum += float32(percent)
                }
            }

            if !success {
                continue
            }

            strong := &TradeStrong{
                Code:    code,
                Price:   getPrice(code),
                Percent: int((nSum / dSum) * 100.0),
                // Percent: int(nSum / float32(len(VAR_TICKS))),
                LtsPercent: int((nSumLts / dSumLts) * 100.0),
            }

            checkTrade(strong)
        }
    }
}

func checkTrade(strong *TradeStrong) {
    elapsed := float64(time.Now().Sub(lastTrade)) / float64(time.Millisecond)
    if elapsed < 500 { // getWalletRemain(strong.Code) < 1 &&
        return
    }

    checkEarnStats()

    if tradeLock {
        return
    }

    ep := getEarnPercent(strong.Code) // 손익율 (0인 경우 미구매 상태)
    strong.EarnPercent = ep

    // strong.Percent    : 단기 틱 주기에 따른 매수, 매도 강도 (0보다 작으면 매도세, 크면 매수세라고 봅니다)
    // strong.LtsPercent : 장기 틱 주기에 따른 매수, 매도 강도

    if strong.Percent >= 25 && ep == 0 && strong.LtsPercent < -50 {
        // 매수세가 강한 경우 매수
        if isSignal(strong.Code, true) && !isWaitLastSellDelay(strong.Code) {
            strong.Remark = "매수 강도 50% 이상이고 매수 흐름을 타고 있음"
            cBuy <- strong
        }
    } else if strong.Percent >= 60 && ep <= -5.0 && strong.LtsPercent > 0 {
        // 물타기
        if isSignal(strong.Code, true) && !isWaitLastSellDelay(strong.Code) {
            strong.Remark = "손실 상태에서 매수 강도 60% 이상이고 매수 흐름을 타는 경우 물타기"
            cBuy <- strong
        }
    } else if strong.Percent <= -80 && ep >= 5.0 {
        // 매수세가 약하지만 이득인 경우 익절
        if isSignal(strong.Code, false) {
            strong.Remark = "매도세가 강하고 상당한 이득 상태"
            cSell <- strong
        }
    } else if strong.Percent <= -50 && ep <= -10.0 && strong.LtsPercent < 0 {
        // 매수세가 약하고 손실인 경우 매도
        if isSignal(strong.Code, false) {
            strong.Remark = "매도 흐름을 타고 있으며 이미 상당한 손실"
            cSell <- strong
        }
    }
}

func initSignalCount(code string, isBuy bool) {
    if isBuy {
        sigBuy.Set(code, 0)
    } else {
        sigSell.Set(code, 0)
    }
}

func getSignalCount(code string, isBuy bool) int {
    if isBuy {
        if v, ok := sigBuy.Get(code); ok {
            return v.(int)
        } else {
            initSignalCount(code, isBuy)
        }
    } else {
        if v, ok := sigSell.Get(code); ok {
            sigSell.Set(code, 0)
            return v.(int)
        } else {
            initSignalCount(code, isBuy)
        }
    }
    return 0
}

func isSignal(code string, isBuy bool) bool {
    count := getSignalCount(code, isBuy)
    initSignalCount(code, !isBuy)
    if count >= SIG_COUNT-1 {
        initSignalCount(code, isBuy)
        return true
    } else if isBuy {
        sigBuy.Set(code, count+1)
    } else {
        sigSell.Set(code, count+1)
    }
    return false
}

func getWaitCount(uuid string) time.Time {
    t := time.Now()
    if v, ok := dealWait.Get(uuid); ok {
        return v.(time.Time)
    } else {
        dealWait.Set(uuid, t)
    }
    return t
}

func isWait(uuid string) bool {
    count := getWaitCount(uuid)
    elapsed := float64(time.Now().Sub(count)) / float64(time.Second)
    if elapsed >= float64(DEAL_WAIT) {
        dealWait.Del(uuid)
        return false
    }
    return true
}

func getWaitLastSell(code string) (time.Time, bool) {
    t := time.Now()
    if v, ok := lastSell.Get(code); ok {
        return v.(time.Time), true
    }
    return t, false
}

func isWaitLastSellDelay(code string) bool {
    if count, ok := getWaitLastSell(code); ok {
        elapsed := float64(time.Now().Sub(count)) / float64(time.Second)
        if elapsed >= float64(SIG_LAST) {
            lastSell.Del(code)
            return false
        }
        return true
    }
    return false
}

func checkWaitOrders() {
    waitMtx.Lock()

    for v := range orderBuy.Iter() {
        uuid := v.Key.(string)
        val := v.Value.(OrderWait)

        status, body, err := Get("https://api.upbit.com/v1/order", fmt.Sprintf("uuid=%s", uuid))
        if err != nil {
            continue
        }

        if status == 200 {
            var oc OrderWaitResponse
            if err := json.Unmarshal(body, &oc); err != nil {
                log.Println(err)
                continue
            }

            tw, ok := wallet.Get(val.Code)
            if !ok {
                continue
            }
            w := tw.(*TradeWallet)

            if oc.State != "done" {
                if isWait(uuid) {
                    continue
                }

                status, _, _ := Delete("https://api.upbit.com/v1/order", OrderRemoveRequest{
                    Uuid: uuid,
                })

                if status == 200 || status == 404 {
                    w.Count -= val.Count
                    w.UsedAmount += val.UsedAmount
                } else {
                    continue
                }
            } else {
                initAccount()
                uses := ToFloat64(oc.Volume) * ToFloat64(oc.Price)
                nowAsset := getAsset()
                earn := nowAsset - lastAsset
                earnPercent := (earn / lastAsset) * 100.0
                LogCsv(val.Code, true, uses, val.Strong.Percent, val.Strong.LtsPercent, earn, earnPercent, nowAsset, val.Strong.EarnPercent, val.Strong.Remark)
                buyCount++
            }

            dealWait.Del(uuid)
            orderBuy.Del(uuid)
            w.Locked = false
        }
    }

    for v := range orderSell.Iter() {
        uuid := v.Key.(string)
        val := v.Value.(OrderWait)

        status, body, err := Get("https://api.upbit.com/v1/order", fmt.Sprintf("uuid=%s", uuid))
        if err != nil {
            continue
        }

        if status == 200 {
            var oc OrderWaitResponse
            if err := json.Unmarshal(body, &oc); err != nil {
                log.Println(err)
                continue
            }

            tw, ok := wallet.Get(val.Code)
            if !ok {
                continue
            }
            w := tw.(*TradeWallet)

            if oc.State != "done" {
                if isWait(uuid) {
                    continue
                }

                status, _, _ := Delete("https://api.upbit.com/v1/order", OrderRemoveRequest{
                    Uuid: uuid,
                })

                if status == 200 || status == 404 {
                    w.Count += val.Count
                    w.UsedAmount -= val.UsedAmount
                } else {
                    continue
                }
            } else {
                initAccount()
                uses := ToFloat64(oc.Volume) * ToFloat64(oc.Price)
                nowAsset := getAsset()
                earn := nowAsset - lastAsset
                earnPercent := (earn / lastAsset) * 100.0
                LogCsv(val.Code, false, uses, val.Strong.Percent, val.Strong.LtsPercent, earn, earnPercent, nowAsset, val.Strong.EarnPercent, val.Strong.Remark)
                sellCount++
            }

            lastSell.Set(w.Code, time.Now())
            dealWait.Del(uuid)
            orderSell.Del(uuid)
            w.Locked = false
        }
    }

    waitMtx.Unlock()
}
