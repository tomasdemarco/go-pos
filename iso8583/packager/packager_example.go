package packager

import (
	"fmt"
	"github.com/tomasdemarco/iso8583/encoding"
	"github.com/tomasdemarco/iso8583/field"
	"github.com/tomasdemarco/iso8583/header"
	"github.com/tomasdemarco/iso8583/packager"
	"github.com/tomasdemarco/iso8583/padding"
	"github.com/tomasdemarco/iso8583/prefix"
	"io"
	"regexp"
)

var hexRegex = regexp.MustCompile(`^[0-9a-fA-F]{16,32}$`)

func CreatePackager() *packager.Packager {

	return &packager.Packager{
		Prefix: prefix.BINARY.BB,
		Header: &HeaderGpPackager{},
		Bitmap: packager.NewField(
			"Primary Bitmap",
			8,
			&encoding.BINARY,
		),
		Fields: map[int]packager.FieldPackager{
			0: packager.NewField(
				"Message Type Indicator",
				4,
				&encoding.BCD,
				packager.WithPattern(regexp.MustCompile(`^(0200|0210|0400|0410|0500|0510|0800|0810)$`)),
			),
			1: packager.NewField(
				"Secondary Bitmap",
				8,
				&encoding.BINARY,
				packager.WithPattern(hexRegex),
			),
			2: packager.NewField(
				"Primary account number",
				19,
				NewBcdEncoder(true),
				packager.WithPrefix(prefix.BCD.LL),
			),
			3: packager.NewField(
				"Processing Code",
				6,
				&encoding.BCD,
				packager.WithPadding(padding.FILL.RIGHT),
			),
			4: packager.NewField(
				"Transaction Amount",
				12,
				&encoding.BCD,
				packager.WithPadding(padding.FILL.LEFT),
				packager.WithCustomType(func() field.Field { return &field.Int{} }),
			),
			7: packager.NewField(
				"Transmission Date & Time",
				10,
				&encoding.BCD,
			),
			11: packager.NewField(
				"Systems Trace Audit Number (STAN)",
				6,
				&encoding.BCD,
				packager.WithPadding(padding.FILL.LEFT),
			),
			12: packager.NewField(
				"Local Transaction Time",
				6,
				&encoding.BCD,
			),
			13: packager.NewField(
				"Local Transaction Date",
				4,
				&encoding.BCD,
			),
			14: packager.NewField(
				"Expiration Date",
				4,
				&encoding.BCD,
			),
			15: packager.NewField(
				"Settlement Date",
				4,
				&encoding.BCD,
			),
			22: packager.NewField(
				"Point of Sale (POS) Entry Mode",
				3,
				&encoding.BCD,
			),
			23: packager.NewField(
				"Card Sequence Number (CSN)",
				3,
				&encoding.BCD,
			),
			24: packager.NewField(
				"Function Code",
				3,
				&encoding.BCD,
			),
			25: packager.NewField(
				"Point of Service Condition Code",
				2,
				&encoding.BCD,
			),
			35: packager.NewField(
				"Track II",
				37,
				NewBcdEncoder(true),
				packager.WithPrefix(prefix.BCD.LL),
			),
			37: packager.NewField(
				"Retrieval Reference Number (RRN)",
				12,
				&encoding.ASCII,
			),
			38: packager.NewField(
				"Authorization Identification",
				6,
				&encoding.ASCII,
			),
			39: packager.NewField(
				"Response Code",
				2,
				&encoding.ASCII,
			),
			41: packager.NewField(
				"Terminal Identification",
				8,
				&encoding.ASCII,
				packager.WithPadding(padding.FILL.RIGHT),
			),
			42: packager.NewField(
				"Merchant Identification",
				15,
				&encoding.ASCII,
				packager.WithPadding(padding.FILL.RIGHT),
			),
			45: packager.NewField(
				"Track I",
				76,
				&encoding.ASCII,
				packager.WithPrefix(prefix.BCD.LL),
			),
			46: packager.NewField(
				"Additional data (ISO)",
				45,
				&encoding.ASCII,
				packager.WithPrefix(prefix.BCD.LLL),
			),
			48: packager.NewField(
				"Additional data (Private)",
				16,
				&encoding.ASCII,
				packager.WithPrefix(prefix.BCD.LLL),
			),
			49: packager.NewField(
				"Transaction Currency Code",
				3,
				&encoding.ASCII,
			),
			52: packager.NewField(
				"Personal Identification Number (PIN)",
				16,
				&encoding.BINARY,
				packager.WithPattern(hexRegex),
			),
			53: packager.NewField(
				"Security Related Control Information (KSN)",
				16,
				&encoding.BINARY,
				packager.WithPattern(hexRegex),
			),
			54: packager.NewField(
				"Additional Amounts",
				12,
				&encoding.ASCII,
				packager.WithPrefix(prefix.BCD.LLL),
			),
			55: packager.NewField(
				"ICC Data",
				510,
				&encoding.ASCII,
				packager.WithPrefix(prefix.BCD.LLL),
			),
			59: packager.NewField(
				"Reserved (National)",
				999,
				&encoding.ASCII,
				packager.WithPrefix(prefix.BCD.LLL),
			),
			60: packager.NewField(
				"Reserved (National)",
				11,
				&encoding.ASCII,
				packager.WithPrefix(prefix.BCD.LLL),
			),
			61: packager.NewField(
				"Reserved (Private)",
				5,
				&encoding.ASCII,
				packager.WithPrefix(prefix.BCD.LLL),
			),
			62: packager.NewField(
				"Reserved (Private)",
				7,
				&encoding.ASCII,
				packager.WithPrefix(prefix.BCD.LLL),
				packager.WithCustomType(func() field.Field {
					return &BatchData{}
				}),
			),
			63: packager.NewField(
				"Reserved (Private)",
				99,
				&encoding.ASCII,
				packager.WithPrefix(prefix.BCD.LLL),
			),
		},
	}
}

type HeaderGpPackager struct{}

func (h *HeaderGpPackager) Unpack(r io.Reader) (value header.Header, length int, err error) {

	buf := make([]byte, 5)
	_, err = r.Read(buf)
	if err != nil {
		if err != io.EOF {
			err = fmt.Errorf("reading header: %w", err)
		}

		return nil, 0, err
	}

	return unpackHeaderGp(buf), 5, nil
}

func (h *HeaderGpPackager) Pack(hdr header.Header) ([]byte, int, error) {
	var b []byte
	if hdr, ok := hdr.Get().(*GpHeader); ok {
		b = append(b, hdr.MessageId...)
		b = append(b, hdr.SourceId...)
		b = append(b, hdr.DestinationId...)
	}
	return b, 5, nil
}

func unpackHeaderGp(b []byte) *GpHeader {

	if len(b) == 5 {
		h := GpHeader{}
		h.MessageId = b[:1]
		h.SourceId = b[1:3]
		h.DestinationId = b[3:5]
		return &h
	}

	return nil
}

type GpHeader struct {
	MessageId     []byte
	SourceId      []byte
	DestinationId []byte
}

func (h *GpHeader) Get() any { return h }
func (h *GpHeader) Set(hdr any) {
	if val, ok := hdr.(*GpHeader); ok {
		h.MessageId = val.MessageId
		h.SourceId = val.SourceId
		h.DestinationId = val.DestinationId
	}
}

func (h *GpHeader) Log() string {
	return fmt.Sprintf("MsgId: %X | SrcId: %X | DstId: %X", h.MessageId, h.SourceId, h.DestinationId)
}

// BatchData Un struct personalizado que implementa CustomPacker
type BatchData struct {
	Ticket string `json:"1"`
	Lote   string `json:"2"`
}

func (c *BatchData) Get() any { return c }

func (c *BatchData) Set(v any) error {
	if val, ok := v.(BatchData); ok {
		c.Ticket = val.Ticket
		c.Lote = val.Lote
		return nil
	}
	return fmt.Errorf("invalid type for String: received %T, expected string", v)
}

func (c *BatchData) Parse() (string, error) {
	return fmt.Sprintf("%s%s", c.Ticket, c.Lote), nil
}

func (c *BatchData) Unparse(data string) error {
	if len(data) < 4 {
		return fmt.Errorf("formato inválido para BatchData: se esperaban mas de 4 caracteres, se obtuvieron %d", len(data))
	}
	c.Ticket = data[:4]
	c.Lote = data[4:]
	return nil
}

func (c *BatchData) Log() (interface{}, error) {
	return c, nil
}
