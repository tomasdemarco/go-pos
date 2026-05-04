package main

import (
	"github.com/tomasdemarco/iso8583/encoding"
	"github.com/tomasdemarco/iso8583/packager"
	"github.com/tomasdemarco/iso8583/padding"
	"github.com/tomasdemarco/iso8583/prefix"
	"regexp"
)

var hexRegex = regexp.MustCompile(`^[0-9a-fA-F]{16,32}$`)

func createPackager() *packager.Packager {

	return &packager.Packager{
		Header: &HeaderGpPackager{},
		Fields: map[int]packager.FieldPackager{
			0: packager.NewField(
				"Message Type Indicator",
				packager.Numeric,
				4,
				&encoding.BCD{},
				packager.WithPattern(regexp.MustCompile(`^(0200|0210|0400|0410|0500|0510|0800|0810)$`)),
			),
			1: packager.NewField(
				"Secondary Bitmap",
				packager.Binary,
				16,
				&encoding.BINARY{},
				packager.WithPattern(hexRegex),
			),
			2: packager.NewField(
				"Primary account number",
				packager.Numeric,
				19,
				encoding.NewBcdEncoder(false),
				packager.WithPrefix(prefix.BCD.LL),
			),
			3: packager.NewField(
				"Processing Code",
				packager.Numeric,
				6,
				&encoding.BCD{},
				packager.WithPadding(padding.NewFillPadder(false, "0")),
			),
			4: packager.NewField(
				"Transaction Amount",
				packager.Numeric,
				12,
				&encoding.BCD{},
				packager.WithPadding(padding.NewFillPadder(true, "0")),
			),
			7: packager.NewField(
				"Transmission Date & Time",
				packager.Numeric,
				10,
				&encoding.BCD{},
			),
			11: packager.NewField(
				"Systems Trace Audit Number (STAN)",
				packager.Numeric,
				6,
				&encoding.BCD{},
				packager.WithPadding(padding.NewFillPadder(true, "0")),
			),
			12: packager.NewField(
				"Local Transaction Time",
				packager.Numeric,
				6,
				&encoding.BCD{},
			),
			13: packager.NewField(
				"Local Transaction Date",
				packager.Numeric,
				4,
				&encoding.BCD{},
			),
			14: packager.NewField(
				"Expiration Date",
				packager.Numeric,
				4,
				&encoding.BCD{},
			),
			15: packager.NewField(
				"Settlement Date",
				packager.Numeric,
				4,
				&encoding.BCD{},
			),
			22: packager.NewField(
				"Point of Sale (POS) Entry Mode",
				packager.Numeric,
				3,
				&encoding.BCD{},
			),
			23: packager.NewField(
				"Card Sequence Number (CSN)",
				packager.Numeric,
				3,
				&encoding.BCD{},
			),
			24: packager.NewField(
				"Function Code",
				packager.Numeric,
				3,
				&encoding.BCD{},
			),
			25: packager.NewField(
				"Point of Service Condition Code",
				packager.Numeric,
				3,
				&encoding.BCD{},
			),
			35: packager.NewField(
				"Track II",
				packager.Numeric,
				37,
				encoding.NewBcdEncoder(false),
				packager.WithPrefix(prefix.BCD.LL),
			),
			37: packager.NewField(
				"Retrieval Reference Number (RRN)",
				packager.String,
				12,
				&encoding.ASCII{},
			),
			38: packager.NewField(
				"Authorization Identification",
				packager.String,
				6,
				&encoding.ASCII{},
			),
			39: packager.NewField(
				"Response Code",
				packager.String,
				2,
				&encoding.ASCII{},
			),
			41: packager.NewField(
				"Terminal Identification",
				packager.String,
				8,
				&encoding.ASCII{},
				packager.WithPadding(padding.NewFillPadder(true, " ")),
			),
			42: packager.NewField(
				"Merchant Identification",
				packager.String,
				15,
				&encoding.ASCII{},
				packager.WithPadding(padding.NewFillPadder(true, " ")),
			),
			45: packager.NewField(
				"Track I",
				packager.String,
				76,
				&encoding.ASCII{},
				packager.WithPrefix(prefix.BCD.LL),
			),
			46: packager.NewField(
				"Additional data (ISO)",
				packager.String,
				45,
				&encoding.ASCII{},
				packager.WithPrefix(prefix.BCD.LLL),
			),
			48: packager.NewField(
				"Additional data (Private)",
				packager.String,
				16,
				&encoding.ASCII{},
				packager.WithPrefix(prefix.BCD.LLL),
			),
			49: packager.NewField(
				"Transaction Currency Code",
				packager.String,
				3,
				&encoding.ASCII{},
			),
			52: packager.NewField(
				"Personal Identification Number (PIN)",
				packager.Binary,
				16,
				&encoding.BINARY{},
				packager.WithPattern(hexRegex),
			),
			53: packager.NewField(
				"Security Related Control Information (KSN)",
				packager.Binary,
				16,
				&encoding.BINARY{},
				packager.WithPattern(hexRegex),
			),
			54: packager.NewField(
				"Additional Amounts",
				packager.String,
				12,
				&encoding.ASCII{},
				packager.WithPrefix(prefix.BCD.LLL),
			),
			55: packager.NewField(
				"ICC Data",
				packager.String,
				510,
				&encoding.ASCII{},
				packager.WithPrefix(prefix.BCD.LLL),
			),
			59: packager.NewField(
				"Reserved (National)",
				packager.String,
				999,
				&encoding.ASCII{},
				packager.WithPrefix(prefix.BCD.LLL),
			),
			60: packager.NewField(
				"Reserved (National)",
				packager.String,
				11,
				&encoding.ASCII{},
				packager.WithPrefix(prefix.BCD.LLL),
			),
			61: packager.NewField(
				"Reserved (Private)",
				packager.String,
				5,
				&encoding.ASCII{},
				packager.WithPrefix(prefix.BCD.LLL),
			),
			62: packager.NewField(
				"Reserved (Private)",
				packager.String,
				7,
				&encoding.ASCII{},
				packager.WithPrefix(prefix.BCD.LLL),
			),
			63: packager.NewField(
				"Reserved (Private)",
				packager.String,
				99,
				&encoding.ASCII{},
				packager.WithPrefix(prefix.BCD.LLL),
			),
		},
	}
}
