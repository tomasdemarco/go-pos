package packager

import (
	"github.com/tomasdemarco/iso8583/encoding"
	"github.com/tomasdemarco/iso8583/message"
	"github.com/tomasdemarco/iso8583/packager"
)

type DE126 struct {
	*message.Message
}

func NewDE126() *message.Message {
	return message.NewMessage(CreateDE126Packager())
}

//func (c *DE126) Get() any { return c }
//
//func (c *DE126) Set(v any) error {
//	return nil
//}

//func (c *DE126) Pack() (string, error) {
//	b, err := c.Message.Pack()
//	if err != nil {
//		return "", err
//	}
//	return fmt.Sprintf("%X", b), err
//}
//
//func (c *DE126) Unpack(data string) error {
//	return c.Message.Unpack(utils.Hex2Byte(data))
//}
//
//func (c *DE126) Log() (interface{}, error) {
//	return c.Message.LogMsg(), nil
//}

func CreateDE126Packager() *packager.Packager {
	return &packager.Packager{
		//Prefix: prefix.BINARY.BB, HAY QUE PONER PARA QUE SEA NULLEABLE?
		Header: &VisaHeaderPackager{},
		Bitmap: packager.NewField(
			"Primary Bitmap",
			8,
			&encoding.BINARY,
		),
		Fields: map[int]packager.FieldPackager{
			5: packager.NewField(
				"Visa Merchant Identifier",
				16,
				&encoding.EBCDIC,
			),
			6: packager.NewField(
				"Secondary Bitmap",
				34,
				&encoding.BCD,
			),
			7: packager.NewField(
				"Merchant Certificate Serial Number",
				34,
				&encoding.BCD,
			),
			8: packager.NewField(
				"Transaction ID (XID)",
				40,
				&encoding.BCD,
			),
			9: packager.NewField(
				"CAVV Data",
				40,
				&encoding.BCD,
			),
			10: packager.NewField(
				"CVV2 Authorization Request Data and American Express CID Data",
				12,
				&encoding.EBCDIC,
			),
			12: packager.NewField(
				"Service Indicators",
				6,
				&encoding.BCD,
			),
			13: packager.NewField(
				"POS Environment",
				2,
				&encoding.EBCDIC,
			),
			15: packager.NewField(
				"Mastercard UCAF Collection Indicator",
				2,
				&encoding.EBCDIC,
			),
			16: packager.NewField(
				"Mastercard UCAF Field",
				66,
				&encoding.EBCDIC,
			),
			18: packager.NewField(
				"Agent Unique Account Result",
				24,
				&encoding.BCD,
			),
			19: packager.NewField(
				"Dynamic Currency Conversion Indicator",
				2,
				&encoding.EBCDIC,
			),
			20: packager.NewField(
				"3-D Secure Indicator",
				2,
				&encoding.EBCDIC,
			),
		},
	}
}
