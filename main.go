package main

import (
    "encoding/json"
    "fmt"
    "github.com/cornelk/hashmap"
    "log"
    "runtime"
    "strconv"
    "sync"
    "time"
)

var (
    // 매매 판단 틱 주기
    DEF_TICK int64 = 10

    // 매매할 종목
    VAR_CODES = []string{
        "KRW-SAND",
        "KRW-BORA",
        "KRW-MANA",
        "KRW-MOC",
        "KRW-PLA",
        "KRW-STORJ",
        "KRW-ANKR",
        "KRW-GLM",
        "KRW-HUM",
        "KRW-NU",
        "KRW-BAT",
        "KRW-HUNT",
        "KRW-HIVE",
    }

    VAR_TICKS = []int32{60, 120, 180}  // 단기 관측 틱(이 프로젝트에서는 1틱 = 거래소 해당 종목의 1회 거래체결로 봅니다)
    VAR_LTS   = []int32{300, 600, 900} // 장기 관측 틱
    MAX_PROC  = 10
    MAX_CPU   = 12
    ORD_TYPE  = "market" // market인 경우 시장가 매매, limit의 경우 지정가 매매
    SIG_COUNT = 1        // 주문 시그널 제한 (매매 판단이 몇번 중복 발생할때 실제 매매가 들어갈지 정합니다)
    DEAL_WAIT = 5        // 지정가 주문의 경우  대기 시간 (초)
    SIG_LAST  = 30       // 마지막 매도로부터 다음 매매 딜레이 (초)
    VAR_TC    = 1.0      // 건당 거래 배율 (시드가 많은 경우 늘리면 됩니다)
    KAKAO_MIN = 60       // 매매 결과를 안내 받기 위한 시간 (분)

    price     = &hashmap.HashMap{}
    askPrice  = &hashmap.HashMap{}
    status    = &hashmap.HashMap{} // map[string]map[int32]*StatusInfo{}
    lts       = &hashmap.HashMap{} // map[string]map[int32]*StatusInfo{}
    mutexes   = &hashmap.HashMap{}
    chance    = &hashmap.HashMap{}
    account   = &hashmap.HashMap{}
    orderBuy  = &hashmap.HashMap{}
    orderSell = &hashmap.HashMap{}
    ticks     = &hashmap.HashMap{}
    sigBuy    = &hashmap.HashMap{}
    sigSell   = &hashmap.HashMap{}
    lastSell  = &hashmap.HashMap{}
    dealWait  = &hashmap.HashMap{}

    cTicker  = make(chan *TradeInfo, 300)
    cStrong  = make(chan string, 50)
    cBuy     = make(chan *TradeStrong, 5)
    cSell    = make(chan *TradeStrong, 5)

    tradeLock = false
    tradeMtx  = &sync.Mutex{}
    waitMtx   = &sync.Mutex{}
    accMtx    = &sync.Mutex{}

    startTime = time.Now()
    lastTrade = time.Now()

    statTime  = time.Now()
    lastAsset = 0.0
    buyCount  = 0
    sellCount = 0

    RUNNING bool
    api     *Api
)

func main() {
    runtime.GOMAXPROCS(MAX_CPU)
    RUNNING = true

    initCoin()

    for i := 0; i < MAX_PROC; i++ {
        go tradeInfoProcess()
        go strongProcess()
    }

    go buyProcess()
    go sellProcess()
    go process()

    ticker := time.NewTicker(time.Millisecond * 5000)
    go func() {
        for _ = range ticker.C {
            checkWaitOrders()
            initAccount()
        }
    }()

    api.Run()
}

func initCoin() {
    initAccount()
    initChance()
}

func initAccount() {
    status, body, err := Get("https://api.upbit.com/v1/accounts", "")
    if err != nil {
        log.Println(err)
        return
    }

    if status == 200 {
        var oc []Account
        if err := json.Unmarshal(body, &oc); err != nil {
            log.Println(err)
            return
        }

        accMtx.Lock()
        a := &hashmap.HashMap{}
        w := &hashmap.HashMap{}

        for _, item := range oc {
            a.Set(item.Currency, item)

            if item.Currency != "KRW" {
                code := fmt.Sprintf("%s-%s", item.UnitCurrency, item.Currency)
                fb, err1 := strconv.ParseFloat(item.Balance, 64)
                fa, err2 := strconv.ParseFloat(item.AvgBuyPrice, 64)
                if err1 == nil && err2 == nil {
                    w.Set(code, &TradeWallet{
                        Code:       code,
                        Count:      fb,
                        Price:      fa,
                        UsedAmount: fb * fa * -1,
                        Locked:     false,
                    })
                }
            }
        }

        account = a
        wallet = w
        accMtx.Unlock()

        nowAsset := getAsset()
        earn := nowAsset - lastAsset
        earnPercent := (earn / lastAsset) * 100.0
        log.Println(fmt.Sprintf("[########] 매수 : %s건 / 매도 : %s건 | 이익 : %s원 (%.2f%%) | 총 보유자산 : %s원", Format(int64(buyCount)), Format(int64(sellCount)), Format(int64(earn)), earnPercent, Format(int64(nowAsset))))

        if lastAsset == 0 && isReady() {
            lastAsset = getAsset() // 보유자산
        }
    } else {
        var res map[string]interface{}
        if err := json.Unmarshal(body, &res); err != nil {
            log.Println(err)
        }
        log.Println(res)
    }
}

func initChance() {
    for _, code := range VAR_CODES {
        status, body, err := Get("https://api.upbit.com/v1/orders/chance", fmt.Sprintf("market=%s", code))
        if err != nil {
            log.Println(err)
            continue
        }

        if status == 200 {
            var oc = &OrderChance{}
            if err := json.Unmarshal(body, oc); err != nil {
                log.Println(err)
                continue
            }

            chance.Set(code, oc)
        } else {
            var res map[string]interface{}
            if err := json.Unmarshal(body, &res); err != nil {
                log.Println(err)
            }
            log.Println(res)
        }
    }
}

func isReady() bool {
    for v := range wallet.Iter() {
        code := v.Key.(string)
        if _, ok := price.Get(code); !ok {
            return false
        }
    }
    return true
}
