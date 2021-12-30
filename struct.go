package main

import (
    "github.com/dgrijalva/jwt-go/v4"
    "time"
)

type StatusInfo struct {
    Code             string  `json:"code"`     // 시장
    Tick             int32   `json:"tick"`     // 기준
    Ask              float64 `json:"ask"`      // 매도
    Bid              float64 `json:"bid"`      // 매수
    Sub              float64 `json:"sub"`      // 차이 (매수 - 매도)
    Percent          int     `json:"percent"`  // 차이 퍼센트
    Count            int32   `json:"count"`    // 쓰기 횟수 (tick 카운트 용도)
}

type TradeInfo struct {
    Type             string  `json:"type"`
    Code             string  `json:"code"`
    Timestamp        int64   `json:"timestamp"`
    TradeDate        string  `json:"trade_date"`
    TradeTime        string  `json:"trade_time"`
    TradeTimestamp   int64   `json:"trade_timestamp"`
    TradePrice       float64 `json:"trade_price"`
    TradeVolume      float64 `json:"trade_volume"`
    AskBid           string  `json:"ask_bid"`
    PrevClosingPrice float64 `json:"prev_closing_price"`
    Change           string  `json:"change"`
    ChangePrice      float64 `json:"change_price"`
    SequentialId     int64   `json:"sequential_id"`
    StreamType       string  `json:"stream_type"`
}

type TradePublish struct {
    Topic            string
    Content          interface{}
}

type TradeStrong struct {
    Code             string  `json:"code"`
    Percent          int     `json:"percent"`
    LtsPercent       int     `json:"lts_percent"`
    Price            float64 `json:"price"`
    Market           bool    `json:"market"`
    Remark           string  `json:"remark"`
    EarnPercent      float64 `json:"earn_percent"`
}

type TradeWallet struct {
    Code             string  `json:"code"`
    Count            float64 `json:"count"`
    UsedAmount       float64 `json:"used_amount"`
    Price            float64 `json:"price"`
    Locked           bool    `json:"locked"`
}

type AuthTokenClaims struct {
    AccessKey        string  `json:"access_key"`
    Nonce            string  `json:"nonce"`
    QueryHash        string  `json:"query_hash"`
    QueryHashAlg     string  `json:"query_hash_alg"`
    jwt.StandardClaims
}

type OrderChance struct {
    BidFee      string `json:"bid_fee"`
    AskFee      string `json:"ask_fee"`
    MakerBidFee string `json:"maker_bid_fee"`
    MakerAskFee string `json:"maker_ask_fee"`
    Market      struct {
        Id         string        `json:"id"`
        Name       string        `json:"name"`
        OrderTypes []interface{} `json:"order_types"`
        OrderSides []string      `json:"order_sides"`
        Bid        struct {
            Currency  string      `json:"currency"`
            PriceUnit interface{} `json:"price_unit"`
            MinTotal  string      `json:"min_total"`
        } `json:"bid"`
        Ask struct {
            Currency  string      `json:"currency"`
            PriceUnit interface{} `json:"price_unit"`
            MinTotal  string      `json:"min_total"`
        } `json:"ask"`
        MaxTotal string `json:"max_total"`
        State    string `json:"state"`
    } `json:"market"`
    BidAccount struct {
        Currency            string `json:"currency"`
        Balance             string `json:"balance"`
        Locked              string `json:"locked"`
        AvgBuyPrice         string `json:"avg_buy_price"`
        AvgBuyPriceModified bool   `json:"avg_buy_price_modified"`
        UnitCurrency        string `json:"unit_currency"`
    } `json:"bid_account"`
    AskAccount struct {
        Currency            string `json:"currency"`
        Balance             string `json:"balance"`
        Locked              string `json:"locked"`
        AvgBuyPrice         string `json:"avg_buy_price"`
        AvgBuyPriceModified bool   `json:"avg_buy_price_modified"`
        UnitCurrency        string `json:"unit_currency"`
    } `json:"ask_account"`
}

type Account struct {
    Currency            string `json:"currency"`
    Balance             string `json:"balance"`
    Locked              string `json:"locked"`
    AvgBuyPrice         string `json:"avg_buy_price"`
    AvgBuyPriceModified bool   `json:"avg_buy_price_modified"`
    UnitCurrency        string `json:"unit_currency"`
}

type OrderBody struct {
    Market  string `json:"market"`
    Side    string `json:"side"`
    Volume  *string `json:"volume,omitempty"`
    Price   *string `json:"price,omitempty"`
    OrdType string `json:"ord_type"`
}

type OrderResponse struct {
    Uuid            string    `json:"uuid"`
    Side            string    `json:"side"`
    OrdType         string    `json:"ord_type"`
    Price           string    `json:"price"`
    State           string    `json:"state"`
    Market          string    `json:"market"`
    CreatedAt       time.Time `json:"created_at"`
    Volume          string    `json:"volume"`
    RemainingVolume string    `json:"remaining_volume"`
    ReservedFee     string    `json:"reserved_fee"`
    RemainingFee    string    `json:"remaining_fee"`
    PaidFee         string    `json:"paid_fee"`
    Locked          string    `json:"locked"`
    ExecutedVolume  string    `json:"executed_volume"`
    TradesCount     int       `json:"trades_count"`
}

type OrderWait struct {
    Code       string  `json:"code"`
    Count      float64 `json:"count"`
    UsedAmount float64 `json:"used_amount"`
    Strong     *TradeStrong
    Detail     *OrderResponse
}

type OrderWaitResponse struct {
    Uuid            string    `json:"uuid"`
    Side            string    `json:"side"`
    OrdType         string    `json:"ord_type"`
    Price           string    `json:"price"`
    State           string    `json:"state"`
    Market          string    `json:"market"`
    CreatedAt       time.Time `json:"created_at"`
    Volume          string    `json:"volume"`
    RemainingVolume string    `json:"remaining_volume"`
    ReservedFee     string    `json:"reserved_fee"`
    RemainingFee    string    `json:"remaining_fee"`
    PaidFee         string    `json:"paid_fee"`
    Locked          string    `json:"locked"`
    ExecutedVolume  string    `json:"executed_volume"`
    TradesCount     int       `json:"trades_count"`
    Trades          []struct {
        Market string `json:"market"`
        Uuid   string `json:"uuid"`
        Price  string `json:"price"`
        Volume string `json:"volume"`
        Funds  string `json:"funds"`
        Side   string `json:"side"`
    } `json:"trades"`
}

type OrderRemoveRequest struct {
    Uuid string `json:"uuid"`
}
