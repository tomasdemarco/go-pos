package packager

import (
	"bufio"
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

func CreateVisaPackager() *packager.Packager {

	return &packager.Packager{
		Prefix: prefix.BINARY.BB,
		Header: &VisaHeaderPackager{},
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
				packager.WithPrefix(prefix.BINARY.B),
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
			5: packager.NewField(
				"Transaction Settlement",
				12,
				&encoding.BCD,
				packager.WithPadding(padding.FILL.LEFT),
				packager.WithCustomType(func() field.Field { return &field.Int{} }),
			),
			6: packager.NewField(
				"Transaction Cardholder Billing",
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
			8: packager.NewField(
				"Amount, Cardholder Billing Fee",
				8,
				&encoding.BCD,
			),
			9: packager.NewField(
				"Conversion Rate, Settlement",
				8,
				&encoding.BCD,
			),
			10: packager.NewField(
				"Conversion Rate, Cardholder Billing",
				8,
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
			16: packager.NewField(
				"Settlement Conversion",
				4,
				&encoding.BCD,
			),
			17: packager.NewField(
				"Settlement Capture",
				4,
				&encoding.BCD,
			),
			18: packager.NewField(
				"Merchant Type",
				4,
				&encoding.BCD,
			),
			19: packager.NewField(
				"Acquiring Institution Country Code",
				3,
				&encoding.BCD,
			),
			20: packager.NewField(
				"PAN Extended, Country Code",
				3,
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
			26: packager.NewField(
				"POS PIN Capture Code",
				2,
				&encoding.BCD,
			),
			28: packager.NewField(
				"Amount, Transaction Fee",
				11,
				&encoding.EBCDIC,
				packager.WithPrefix(prefix.EBCDIC.LL),
			),
			32: packager.NewField(
				"Acquiring Institution Identification Code",
				11,
				NewBcdEncoder(true),
				packager.WithPrefix(prefix.BINARY.B),
			),
			33: packager.NewField(
				"Forwarding Institution Identification Code",
				11,
				NewBcdEncoder(true),
				packager.WithPrefix(prefix.BINARY.B),
			),
			34: packager.NewField(
				"Acceptance Environment Data",
				11,
				NewBcdEncoder(true),
				packager.WithPrefix(prefix.BINARY.B),
			),
			35: packager.NewField(
				"Track II",
				37,
				NewBcdEncoder(true),
				packager.WithPrefix(prefix.BINARY.B),
			),
			37: packager.NewField(
				"Retrieval Reference Number (RRN)",
				12,
				&encoding.EBCDIC,
			),
			38: packager.NewField(
				"Authorization Identification",
				6,
				&encoding.EBCDIC,
			),
			39: packager.NewField(
				"Response Code",
				2,
				&encoding.EBCDIC,
			),
			41: packager.NewField(
				"Terminal Identification",
				8,
				&encoding.EBCDIC,
				packager.WithPadding(padding.FILL.RIGHT),
			),
			42: packager.NewField(
				"Merchant Identification",
				15,
				&encoding.EBCDIC,
				packager.WithPadding(padding.FILL.RIGHT),
			),
			43: packager.NewField(
				"Card Acceptor Name/Location",
				40,
				&encoding.EBCDIC,
			),
			44: packager.NewField(
				"Additional Data",
				99,
				&encoding.EBCDIC,
				packager.WithPrefix(prefix.BINARY.B),
			),
			45: packager.NewField(
				"Track I",
				76,
				&encoding.EBCDIC,
				packager.WithPrefix(prefix.BINARY.B),
			),
			46: packager.NewField(
				"Additional data (ISO)",
				999,
				&encoding.EBCDIC,
				packager.WithPrefix(prefix.BINARY.BB),
			),
			47: packager.NewField(
				"Additional data (National)",
				99,
				&encoding.EBCDIC,
				packager.WithPrefix(prefix.BINARY.B),
			),
			48: packager.NewField(
				"Additional data (Private)",
				99,
				&encoding.EBCDIC,
				packager.WithPrefix(prefix.BINARY.B),
			),
			49: packager.NewField(
				"Transaction Currency Code",
				3,
				&encoding.BCD,
			),
			50: packager.NewField(
				"Currency Code, Settlement",
				3,
				&encoding.BCD,
			),
			51: packager.NewField(
				"Currency Code, Cardholder Billing",
				3,
				&encoding.BCD,
			),
			52: packager.NewField(
				"PIN Data",
				8,
				&encoding.BINARY,
				packager.WithPattern(hexRegex),
			),
			53: packager.NewField(
				"Security Related Control Information",
				16,
				&encoding.BCD,
				packager.WithPattern(hexRegex),
			),
			54: packager.NewField(
				"Additional Amounts",
				12,
				&encoding.EBCDIC,
				packager.WithPrefix(prefix.BINARY.BB),
			),
			55: packager.NewField(
				"ICC Data",
				255,
				&encoding.BINARY,
				packager.WithPrefix(prefix.BINARY.B),
			),
			56: packager.NewField(
				"Payment Account Reference Data",
				255,
				&encoding.EBCDIC,
				packager.WithPrefix(prefix.BINARY.B),
			),
			57: packager.NewField(
				"Reserved National",
				255,
				&encoding.EBCDIC,
				packager.WithPrefix(prefix.BINARY.B),
			),
			58: packager.NewField(
				"Reserved National",
				255,
				&encoding.EBCDIC,
				packager.WithPrefix(prefix.BINARY.B),
			),
			59: packager.NewField(
				"National POS Geographic Data",
				14,
				&encoding.EBCDIC,
				packager.WithPrefix(prefix.BINARY.BB),
			),
			60: packager.NewField(
				"Reserved National",
				99,
				&encoding.EBCDIC,
				packager.WithPrefix(prefix.BINARY.B),
			),
			61: packager.NewField(
				"Reserved (Private)",
				36,
				&encoding.BINARY,
				packager.WithPrefix(prefix.BINARY.B),
			),
			62: packager.NewField(
				"Reserved (Private)",
				99,
				&encoding.BINARY,
				packager.WithPrefix(prefix.BINARY.B),
			),
			63: packager.NewField(
				"Reserved (Private)",
				99,
				&encoding.BINARY,
				packager.WithPrefix(prefix.BINARY.B),
			),
			66: packager.NewField(
				"Settlement Code",
				1,
				&encoding.BCD,
			),
			67: packager.NewField(
				"Extended Payment Code",
				2,
				&encoding.BCD,
			),
			68: packager.NewField(
				"Receiving Institution Country Code",
				3,
				&encoding.BCD,
			),
			69: packager.NewField(
				"Settlement Institution Country Code",
				3,
				&encoding.BCD,
			),
			70: packager.NewField(
				"Network Management Information Code",
				3,
				&encoding.BCD,
			),
			71: packager.NewField(
				"Message Number",
				4,
				&encoding.BCD,
			),
			72: packager.NewField(
				"Message Number Last",
				4,
				&encoding.BCD,
			),
			73: packager.NewField(
				"Date, Action",
				6,
				&encoding.BCD,
			),
			90: packager.NewField(
				"Original Data Elements",
				42,
				&encoding.BCD,
			),
			104: packager.NewField(
				"Transaction Description and Transaction-Specific Data",
				99,
				&encoding.BINARY,
				packager.WithPrefix(prefix.BINARY.B),
			),
			114: packager.NewField(
				"Domestic and Localized Data",
				99,
				&encoding.BINARY,
				packager.WithPrefix(prefix.BINARY.B),
			),
			126: packager.NewField(
				"Visa Private-Use Fields",
				99,
				&encoding.BINARY,
				packager.WithPrefix(prefix.BINARY.B),
				packager.WithCustomType(func() field.Field {
					return NewDE126()
				}),
			),
		},
	}
}

type VisaHeader struct {
	H1  []byte
	H2  []byte
	H3  []byte
	H4  []byte
	H5  []byte
	H6  []byte
	H7  []byte
	H8  []byte
	H9  []byte
	H10 []byte
	H11 []byte
	H12 []byte
	H13 []byte
	H14 []byte
}

func (h *VisaHeader) Get() any { return h }
func (h *VisaHeader) Set(header any) {
	if headerVal, ok := header.(*VisaHeader); ok {
		h.H1 = headerVal.H1
		h.H2 = headerVal.H2
		h.H3 = headerVal.H3
		h.H4 = headerVal.H4
		h.H5 = headerVal.H5
		h.H6 = headerVal.H6
		h.H7 = headerVal.H7
		h.H8 = headerVal.H8
		h.H9 = headerVal.H9
		h.H10 = headerVal.H10
		h.H11 = headerVal.H11
		h.H12 = headerVal.H12
	}
}

func (h *VisaHeader) Log() string {
	val := fmt.Sprintf("H1: %X | H2: %X | H3: %X | H4: %X | H5: %X | H6: %X | H7: %X | H8: %X | H9: %X | H10: %X | H11: %X | H12: %X", h.H1, h.H2, h.H3, h.H4, h.H5, h.H6, h.H7, h.H8, h.H9, h.H10, h.H11, h.H12)

	if h.H13 != nil && h.H14 != nil {
		val += fmt.Sprintf(" | H13: %X | H14: %X", h.H13, h.H14)
	}
	return val
}

func LengthVisaPack(prefixer prefix.Prefixer, lenMessage int) ([]byte, error) {
	b, err := prefixer.EncodeLength(lenMessage)
	if err != nil {
		return nil, err
	}

	return append(b, []byte{0x00, 0x00}...), nil
}

func LengthVisaUnpack(r *bufio.Reader, prefixer prefix.Prefixer) (int, error) {

	buf := make([]byte, prefixer.GetPackedLength()+2)
	_, err := io.ReadFull(r, buf)
	if err != nil {
		if err != io.EOF {
			err = fmt.Errorf("reading length: %w", err)
		}

		return 0, err
	}

	result, err := prefixer.DecodeLength(buf[:prefixer.GetPackedLength()], 0)
	if err != nil {
		return 0, err
	}

	return result, err
}

type VisaHeaderPackager struct{}

func (h *VisaHeaderPackager) Unpack(r io.Reader) (val header.Header, length int, err error) {

	headerLen := 22
	buf := make([]byte, headerLen)
	_, err = io.ReadFull(r, buf)
	if err != nil {
		if err != io.EOF {
			err = fmt.Errorf("reading header: %w", err)
		}

		return nil, 0, err
	}

	hdr := unpackHeader(buf)
	h1 := int(hdr.H1[0])
	if h1 >= 26 {
		rejectBuf := make([]byte, h1-headerLen)
		_, err = io.ReadFull(r, rejectBuf)
		if err != nil {
			if err != io.EOF {
				err = fmt.Errorf("reading header: %w", err)
			}
			return nil, 0, err
		}

		headerLen = h1

		hdr.H13 = rejectBuf[:2]
		hdr.H14 = rejectBuf[2:4]
	}

	return hdr, headerLen, nil
}

func (h *VisaHeaderPackager) Pack(val header.Header) ([]byte, int, error) {
	var b []byte
	if hdr, ok := val.Get().(*VisaHeader); ok {
		b = append(b, hdr.H1...)
		b = append(b, hdr.H2...)
		b = append(b, hdr.H3...)
		b = append(b, hdr.H4...)
		b = append(b, hdr.H5...)
		b = append(b, hdr.H6...)
		b = append(b, hdr.H7...)
		b = append(b, hdr.H8...)
		b = append(b, hdr.H9...)
		b = append(b, hdr.H10...)
		b = append(b, hdr.H11...)
		b = append(b, hdr.H12...)
	}

	return b, 22, nil
}

func assembleHeader() *VisaHeader {

	hdr := VisaHeader{}
	hdr.H1 = []byte{0x16}
	hdr.H2 = []byte{0x01}
	hdr.H3 = []byte{0x02}
	hdr.H4 = []byte{0x00, 0xe9}
	hdr.H5 = []byte{0x00, 0x00, 0x00}
	hdr.H6 = []byte{0x86, 0x39, 0x16}
	hdr.H7 = []byte{0x00}
	hdr.H8 = []byte{0x00, 0x00}
	hdr.H9 = []byte{0x00, 0x00, 0x00}
	hdr.H10 = []byte{0x00}
	hdr.H11 = []byte{0x00, 0x00, 0x00}
	hdr.H12 = []byte{0x00}

	return &hdr
}

func unpackHeader(b []byte) *VisaHeader {

	if len(b) >= 22 {
		visaHeader := VisaHeader{}
		visaHeader.H1 = b[:1]
		visaHeader.H2 = b[1:2]
		visaHeader.H3 = b[2:3]
		visaHeader.H4 = b[3:5]
		visaHeader.H5 = b[5:8]
		visaHeader.H6 = b[8:11]
		visaHeader.H7 = b[11:12]
		visaHeader.H8 = b[12:14]
		visaHeader.H9 = b[14:17]
		visaHeader.H10 = b[17:18]
		visaHeader.H11 = b[18:21]
		visaHeader.H12 = b[21:22]
		return &visaHeader
	}

	return nil
}
