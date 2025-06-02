package mobilepay

type Currency string

const (
	CurrencyDKK Currency = "DKK"
	CurrencyNOK Currency = "NOK"
	CurrencyEUR Currency = "EUR"
)

type Amount struct {
	Currency Currency `json:"currency"`
	Value    int64    `json:"value"`
}

type UserFlow string

const (
	UserFlowPush   UserFlow = "PUSH_MESSAGE"
	UserFlowNative UserFlow = "NATIVE_REDIRECT"
	UserFlowWeb    UserFlow = "WEB_REDIRECT"
	UserFlowQr     UserFlow = "QR"
)

type OrderLine struct {
	ID                      string `json:"id"`
	Name                    string `json:"name"`
	TotalAmount             int64  `json:"totalAmount"`
	TotalAmountExcludingTax int64  `json:"totalAmountExcludingTax"`
	TotalTaxAmount          int64  `json:"totalTaxAmount"`
	TaxPercentage           int    `json:"taxPercentage"`
	TaxRate                 int    `json:"taxRate"`
	UnitInfo                struct {
		UnitPrice    int64  `json:"unitPrice"`
		Quantity     string `json:"quantity"`
		QuantityUnit string `json:"quantityUnit"`
	} `json:"unitInfo"`
	Discount   int64  `json:"discount"`
	ProductUrl string `json:"productUrl"`
	IsReturn   bool   `json:"isReturn"`
	IsShipping bool   `json:"isShipping"`
}
type Receipt struct {
	OrderLines []OrderLine `json:"orderLines,omitempty"`
	BottomLine BottomLine  `bottomLine,omitempty"`
}
type BottomLine struct {
	Currency Currency `json:"currency,omitempty"`
	/*
	   "currency": "NOK",
	   "tipAmount": 2000,
	   "giftCardAmount": 20000,
	   "posId": "string",
	   "totalAmount": 0,
	   "totalTax": 0,
	   "totalDiscount": 0,
	   "shippingAmount": 0,
	   "shippingInfo": {
	     "amount": 1000,
	     "amountExcludingTax": 1000,
	     "taxAmount": 250,
	     "taxPercentage": 25
	   },
	   "paymentSources": {
	     "giftCard": 0,
	     "card": 0,
	     "voucher": 0,
	     "cash": 0
	   },
	   "barcode": {
	     "format": "EAN-13",
	     "data": "string"
	   },
	   "receiptNumber": "string",
	   "terminalId": "string"
	*/
}
type PaymentState string

const (
	PaymentStateCreated    PaymentState = "CREATED"
	PaymentStateExpired    PaymentState = "EXPIRED"
	PaymentStateAuthorized PaymentState = "AUTHORIZED"
)

type PaymentMethodType string
type PaymentMethodSource string
type PaymentReference string

type PaymentMethod struct {
	Type           PaymentMethodType     `json:"type"`
	BlockedSources []PaymentMethodSource `json:"blockedSources,omitempty"`
	CardBin        string                `json:"cardBin,omitempty"`
}
type Customer struct {
	PhoneNumber string `json:"phoneNumber,omitempty"`
}
type Payment struct {
	Amount         Amount   `json:"amount"`
	Customer       Customer `json:"customer,omitempty"`
	MinimumUserAge int      `json:"minimumUserAge,omitempty"`
	/*
	  "customerInteraction": "CUSTOMER_NOT_PRESENT",
	  "industryData": {
	    "airlineData": {
	      "agencyInvoiceNumber": "string",
	      "airlineCode": "074",
	      "airlineDesignatorCode": "KL",
	      "passengerName": "FLYER / MARY MS.",
	      "ticketNumber": "123-1234567890"
	    }
	  },*/
	PaymentMethod PaymentMethod `json:"paymentMethod"`
	Profile       struct {
		Scope string `json:"scope"`
	} `json:"profile"`
	Reference          PaymentReference  `json:"reference"`
	ReturnUrl          string            `json:"returnUrl"`
	UserFlow           UserFlow          `json:"userFlow"`
	PaymentDescription string            `json:"paymentDescription"`
	Receipt            Receipt           `json:"-"` //`json:"receipt,omitempty"`
	ReceiptUrl         string            `json:"receiptUrl,omitempty"`
	Metadata           map[string]string `json:"metadata,omitempty"`
	/*
		{"amount":{"currency":"DKK","value":225},"paymentMethod":{"type":"WALLET"},"customer":{"phoneNumber":"4540733886"},"reference":"19e7cc26-f2b9-40be-9aed-449f17b31c9d","returnUrl":"/callback/mobilepay/19e7cc26-f2b9-40be-9aed-449f17b31c9d","userFlow":"WEB_REDIRECT","paymentDescription":"Gøglertilmelding"}'
			  "expiresAt": "2023-02-26T17:32:28Z",
			  "qrFormat": {
			    "format": "IMAGE/SVG+XML",
			    "size": 1024
			  },
			}
	*/
}
