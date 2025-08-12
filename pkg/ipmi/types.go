package ipmi

import (
	"errors"
	"fmt"
	"math"
)

type Request struct {
	Addr    uintptr
	AddrLen uint
	Msgid   int
	Msg     ipmiMsg
}

type Response struct {
	Ccode   uint8
	Data    [1024]uint8
	DataLen int32
}

type ipmiAddr struct {
	AddrType int
	Channel  int16
	Data     [32]byte
}

type ipmiMsg struct {
	Netfn   uint8
	Cmd     uint8
	DataLen uint16
	Data    uintptr
}

type ipmiRecv struct {
	RecvType int
	Addr     uintptr
	AddrLen  uint
	Msgid    int
	Msg      ipmiMsg
}

type ipmiSystemInterfaceAddr struct {
	AddrType uint32
	Channel  uint8
	Lun      uint8
}

// Constants.
const (
	unknown = "Unknown"
)

// All of the below structs and conversions are nicked from
// https://github.com/gebn/bmc project. They are modified to current
// library needs.

// ConversionFactors contains inputs to the linear formula in 30.3 and 36.3 of
// v1.5 and v2.0 respectively. This struct exists as conversion factors can come
// from two sources: full sensor records, and the Get Sensor Reading Factors
// command response. In practice, we get them from the former for linear and
// linearised sensors, as these have constant factors. We need to obtain them
// from the Get Sensor Reading Factors command for non-linear sensors, as they
// vary by reading here. Both FullSensorRecord and GetSensorReadingFactorsRsp
// embed this type.
//
// Note that we split application of the formula into "conversion" and
// "linearisation". Conversion happens first, and is the linear formula applied
// to the raw value. The linearisation step, which is a no-op for linear and
// non-linear sensors, applies one of the formulae in the specification to the
// result of the conversion. This struct only deals with conversion; see
// Lineariser for linearisation.
type ConversionFactors struct {
	// M is the constant multiplier. This is a 10-bit 2's complement number on
	// the wire.
	M int16

	// B is the additive offset. This is a 10-bit 2's complement number on the
	// wire.
	B int16

	// BExp is the exponent, controlling the location of the decimal point in B.
	// This is also referred to as K1 in the spec, and is a 4-bit 2's complement
	// number on the wire.
	BExp int8

	// RExp is the result exponent, controlling the location of the decimal
	// point in the result of the linear formula and hence input to the
	// linearisation function. This is also referred to as K2 in the spec, and
	// is a 4-bit 2's complement number on the wire.
	RExp int8
}

// ConvertReading applies the linear formula to a raw sensor reading, without
// the linearisation formula. It is independent of unit. This method takes an
// int16 rather than uint8 as raw values can be in 1 or 2's complement, or
// unsigned, so it must accept from -128 (lowest 2's complement) to 255 (highest
// unsigned). The conversion from the raw format to a native int must be done
// before calling this method.
func (f *ConversionFactors) ConvertReading(raw int16) float64 {
	mX := int64(f.M) * int64(raw)
	b10k1 := float64(f.B) * math.Pow10(int(f.BExp))

	return (float64(mX) + b10k1) * math.Pow10(int(f.RExp))
}

// FullSensorRecord is specified in 37.1 and 43.1 of v1.5 and v2.0 respectively.
// It describes any type of sensor, and is the only record type that can
// describe a sensor generating analogue (i.e. non-enumerated/discrete)
// readings, e.g. a temperature sensor. It is specified as 64 bytes. This layer
// represents the record key and record body sections.
type FullSensorRecord struct {
	ConversionFactors

	// Sensor number that will be used in request to get reading
	Number uint8

	// BaseUnit gives the primary unit of the sensor's reading, e.g. Celsius or
	// Fahrenheit for a temperature sensor.
	BaseUnit SensorUnit

	// ModifierUnit is contained in the Sensor Units 3 field. Note this is
	// distinct from the identically-named 2-bit field in Sensor Units 1. 0x0
	// means unused.
	ModifierUnit SensorUnit

	// Linearisation indicates whether the sensor is linear, linearised or
	// non-linear. This controls post-processing after applying the linear
	// conversion formula to the raw reading.
	Linearisation Linearisation

	// Tolerance gives the absolute accuracy of the sensor in +/- half raw
	// counts. This is a 6-bit uint on the wire.
	Tolerance uint8

	// Accuracy gives the sensor accuracy in 0.01% increments when raised to
	// AccuracyExp. This is a 10-bit int on the wire.
	Accuracy int16

	// AccuracyExp is the quantity Accuracy is raised to the power of to give
	// the final accuracy.
	AccuracyExp uint8

	// Identity is a descriptive string for the sensor. This can be up to 16
	// bytes long, which translates into 16-32 characters depending on the
	// format used. There are no conventions around this, and it is provided for
	// informational purposes only. Contrary to the name, attempting to identify
	// sensors based on this value is doomed to fail.
	Identity string
}

func (r *FullSensorRecord) DecodeFromBytes(data []uint8) error {
	if len(data) < 51 {
		return fmt.Errorf("full sensor records are at least 51 bytes long, got %v",
			len(data))
	}

	r.Number = data[10]

	r.BaseUnit = SensorUnit(data[24])
	r.ModifierUnit = SensorUnit(data[25])

	r.Linearisation = Linearisation(data[26] & 0x7f)

	buf := [...]byte{data[28] >> 6, data[27]}
	r.M = twos(buf, 10)
	r.Tolerance = data[28] & 0x3f
	buf[1] = data[29]
	buf[0] = data[30] >> 6
	r.B = twos(buf, 10)
	buf[1] = data[30]&0x3f | ((data[31] & 0xf0) << 2)
	buf[0] = (data[31] & 0xf0) >> 6
	r.Accuracy = twos(buf, 10)
	r.AccuracyExp = (data[31] & 0xc) >> 2
	buf[0] = 0
	buf[1] = data[32] >> 4
	r.RExp = int8(twos(buf, 4)) //nolint:gosec
	buf[1] = data[32] & 0xf
	r.BExp = int8(twos(buf, 4)) //nolint:gosec

	encoding := StringEncoding(data[50] >> 6)

	decoder, err := encoding.Decoder()
	if err != nil {
		// unsupported encoding; fail loudly so we can fix this
		return err
	}

	characters := int(data[50] & 0x1f)

	identity, _, err := decoder.Decode(data[51:], characters)
	if err != nil {
		// invalid bytes
		return err
	}

	r.Identity = identity

	return nil
}

// bcdPlus defines the mappings of BCD plus nibbles to runes, specified in
// 37.15 and 43.15 of v1.5 and v2.0 respectively. An N byte string consists
// of 2N characters.
var bcdPlusRunes = [16]rune{
	'0', '1', '2', '3', '4', '5', '6', '7', '8', '9',
	' ', '-', '.', ':', ',', '_',
}

// StringDecoder is implemented by things that know how to parse the final ID
// String field of full and compact SDRs.
type StringDecoder interface {
	// Decode parses the first c characters (0 <= c <= 30) in b in the expected
	// format (N.B. this could be a varying number of bytes depending on the
	// encoding), returning the resulting string and number of bytes consumed,
	// or an error if the data is too short or invalid.
	//
	// c was implemented as an int rather than uint8 to reduce the number of
	// conversions required.
	Decode(b []byte, c int) (string, int, error)
}

// StringDecoderFunc eases implementation of stateless StringDecoders.
type StringDecoderFunc func([]byte, int) (string, int, error)

// Decode calls the contained function on the inputs, passing through the
// returned values verbatim.
func (f StringDecoderFunc) Decode(b []byte, c int) (string, int, error) {
	return f(b, c)
}

// StringEncoding describes the most significant two bits of the SDR Type/Length
// Byte, specified in 37.15 and 43.15 of v1.5 and v2.0 respectively.
type StringEncoding uint8

const (
	// StringEncodingUnicode, contrary to the name, typically suggests an
	// unspecified encoding. IPMItool displays a hex representation of the
	// underlying bytes, while OpenIPMI interprets it identically to
	// StringEncoding8BitAsciiLatin1. Given Unicode is only a character set and
	// the spec does not suggest any encoding, there is no right answer. The
	// resulting variety of implementations means use of this value by a BMC
	// should be regarded as a bug.
	StringEncodingUnicode StringEncoding = iota
	StringEncodingBCDPlus
	StringEncodingPacked6BitAscii
	StringEncoding8BitAsciiLatin1
)

var (
	stringEncodingDescriptions = map[StringEncoding]string{
		StringEncodingUnicode:         "Unicode",
		StringEncodingBCDPlus:         "BCD plus",
		StringEncodingPacked6BitAscii: "6-bit ASCII, packed",
		StringEncoding8BitAsciiLatin1: "8-bit ASCII + Latin 1",
	}
	// to ease readability and testability.
	stringEncodingDecoders = map[StringEncoding]StringDecoder{
		// despite the ambiguity of StringEncodingUnicode, we follow OpenIPMI
		// and decode it as 8-bit ASCII
		StringEncodingUnicode:         StringDecoderFunc(decode8BitAsciiLatin1),
		StringEncodingBCDPlus:         StringDecoderFunc(decodeBCDPlus),
		StringEncodingPacked6BitAscii: StringDecoderFunc(decodePacked6BitAscii),
		StringEncoding8BitAsciiLatin1: StringDecoderFunc(decode8BitAsciiLatin1),
	}
)

func (e StringEncoding) Decoder() (StringDecoder, error) {
	if decoder, ok := stringEncodingDecoders[e]; ok {
		return decoder, nil
	}

	return nil, fmt.Errorf("no decoder found for encoding %v", e)
}

func (e StringEncoding) Description() string {
	if desc, ok := stringEncodingDescriptions[e]; ok {
		return desc
	}

	return unknown
}

func (e StringEncoding) String() string {
	return fmt.Sprintf("%#v(%v)", uint8(e), e.Description())
}

func decodeBCDPlus(b []byte, c int) (string, int, error) {
	// each byte contains 2 characters (1 per nibble), so the number of
	// bytes we expect equals half the number of characters, rounded up
	bytes := int(math.Ceil(float64(c) / 2))
	if len(b) < bytes {
		return "", 0, fmt.Errorf("expected %v bytes, got %v", bytes, len(b))
	}

	runes := make([]rune, c)

	for i := range c {
		shift := uint8(0)
		if i%2 == 0 {
			// character is in the most significant 4 bits; need to
			// shift down
			shift = 4
		}

		runes[i] = bcdPlusRunes[(b[i/2]>>shift)&0xf]
	}

	return string(runes), bytes, nil
}

func decodePacked6BitAscii(b []byte, c int) (string, int, error) {
	// the minimum number of bytes required to represent c characters; c does
	// not have to be a multiple of 4
	bytes := c - (c / 4)
	if len(b) < bytes {
		return "", 0, fmt.Errorf("expected %v bytes, got %v", bytes, len(b))
	}

	runes := make([]rune, c)
	acc := uint8(0)

	for i := range c {
		// offset is the start offset for the first ASCII bits of the char at i;
		// this formula required a bit of experimentation in Excel. N.B. cannot
		// remove math.Floor() as need to round towards 0, not just strip the
		// fractional component.
		offset := (i - 1) - int(math.Floor(float64(i-1)/4))

		// the switch extracts the appropriate 6 bits into acc (most significant
		// two bits will always be 0)
		switch i % 4 {
		case 0:
			// least significant 6 bits at offset
			acc = b[offset] & 0x3f
		case 1:
			// least sig 2 bits: most sig 2 bits at offset
			// most sig 4 bits: least sig 4 bits at offset + 1
			acc = b[offset] >> 6
			acc |= (b[offset+1] & 0xf) << 2
		case 2:
			// least sig 4 bits: most sig 4 bits at offset
			// most sig 2 bits: least sig 2 bits at offset + 1
			acc = b[offset] >> 4
			acc |= (b[offset+1] & 0x3) << 4
		case 3:
			// most sig 6 bits at offset
			acc = b[offset] >> 2
		}

		// observe character corresponding to code
		runes[i] = rune(acc + 0x20)
	}

	return string(runes), bytes, nil
}

func decode8BitAsciiLatin1(b []byte, c int) (string, int, error) {
	if len(b) < 2 {
		// it is unclear why this limitation exists, but it's plain to
		// see in the specification
		return "", 0, fmt.Errorf("at least 2 bytes of data must be present; got %v bytes", len(b))
	}

	// bounds check to ensure the slicing below does not panic
	if len(b) < c {
		return "", 0, fmt.Errorf("expected %v bytes, got %v", c, len(b))
	}

	// can convert straight into a string as the encoding's range is
	// identical to UTF-8
	return string(b[:c]), c, nil
}

const (
	LinearisationLinear Linearisation = iota
	LinearisationLn
	LinearisationLog10
	LinearisationLog2
	LinearisationE
	LinearisationExp10
	LinearisationExp2
	LinearisationInverse
	LinearisationSqr
	LinearisationCube
	LinearisationSqrt
	LinearisationCubeRt
	LinearisationNonLinear

	// 0x71 through 0x7f are reserved for non-linear, OEM defined
	// linearisations. It is unclear why these cannot use
	// LinearisationNonLinear, as being non-linear, they do not have a
	// linearisation formula. Waiting for a use case to emerge rather than
	// implementing a questionably useful RegisterLineariser() function.
)

var (
	// ErrNotLinearised is returned if Lineariser() is called on a linear or
	// non-linear linearisation. Linear sensors' values do not require any
	// transformation by virtue of the sensor already being linear. If the sensor
	// is non-linear, the conversion factors returned by Get Sensor Reading
	// Factors are all that are needed to obtain a real value: by being unique
	// to the raw sensor reading, there is no need for a separate linearisation
	// formula.
	//
	// Linearise() could return a no-op lineariser, however the current
	// implementation should never ask for one on a non-linearised sensor, so
	// instead we return an error to flag up a possible bug.
	ErrNotLinearised = errors.New(
		"only linearised sensors have a linearisation formula")

	linearisationDescriptions = map[Linearisation]string{
		LinearisationLinear:    "Linear",
		LinearisationLn:        "ln",
		LinearisationLog10:     "log10",
		LinearisationLog2:      "log2",
		LinearisationE:         "e",
		LinearisationExp10:     "exp10",
		LinearisationExp2:      "exp2",
		LinearisationInverse:   "1/x",
		LinearisationSqr:       "sqr(x)",
		LinearisationCube:      "cube(x)",
		LinearisationSqrt:      "sqrt(x)",
		LinearisationCubeRt:    "x^(1/3)",
		LinearisationNonLinear: "Non-linear",
	}

	// linearisationLinearisers allows us to find out what linearisation formula
	// needs to be applied to the converted output of a linearised sensor, to
	// produce a real value. Note that linear and non-linear linearisations do
	// not appear here as they don't need a linearisation formula.
	linearisationLinearisers = map[Linearisation]Lineariser{
		LinearisationLn:    LineariserFunc(math.Log),
		LinearisationLog10: LineariserFunc(math.Log10),
		LinearisationLog2:  LineariserFunc(math.Log2),
		LinearisationE:     LineariserFunc(math.Exp),
		LinearisationExp10: LineariserFunc(func(f float64) float64 {
			// cannot use math.Pow10 as that takes an int
			return math.Pow(10, f)
		}),
		LinearisationExp2: LineariserFunc(math.Exp2),
		LinearisationInverse: LineariserFunc(func(f float64) float64 {
			return math.Pow(f, -1)
		}),
		LinearisationSqr: LineariserFunc(func(f float64) float64 {
			return f * f
		}),
		LinearisationCube: LineariserFunc(func(f float64) float64 {
			return f * f * f
		}),
		LinearisationSqrt: LineariserFunc(math.Sqrt),
		LinearisationCubeRt: LineariserFunc(func(f float64) float64 {
			return math.Pow(f, 1./3)
		}),
	}
)

// Linearisation indicates whether a sensor is linear, linearised, or
// non-linear. Values are specified in the Full Sensor Record wire format table
// in 37-1 and 43-1 of v1.5 and v2.0 respectively.
//
// Linear sensors are the easiest to deal with. The sensor's raw readings are
// converted into real readings (e.g. Celsius) with a linear formula. Accuracy
// and resolution are constant in real terms across the entire range of values
// produced by the sensor.
//
// Linearised are slightly more challenging. The same linear formula is applied
// as for linear sensors, however a final "linearisation formula" is applied to
// obtain the real reading. This transformation is one of 11 defined in the
// spec, e.g. log or sqrt, and obviously does not have to be linear itself. The
// tolerance (the spec misuses accuracy as a synonym) of linearised sensors is
// also constant for all values. This is possible despite the existence of the
// linearisation formula turning raw values into disproportionate real values,
// as tolerance is expressed relative to 0. This assumes the sensor's tolerance
// does not diminish in real, absolute terms at extreme values (positive or
// negative), as there is no way of representing it (you'd have to resort to
// declaring it a non-linear sensor). Note that tolerance can only be expressed
// in half-raw value increments, which is in itself quite coarse. Regarding
// resolution, this will vary with reading due to the linearisation formula. The
// recommended way to calculate it is to retrieve and calculate the real values
// (with the help of Get Sensor Reading Factors as necessary) corresponding to
// the raw values below and above the actual raw value observed. Subtracting the
// real reading for the raw value below the observed raw value from the real
// reading for the observed value gives the negative resolution, and the process
// is equivalent for the positive resolution using the raw value one above.
//
// All consistency bets are off with non-linear sensors. Not only does
// resolution vary by reading (calculated in the same was as for linearised
// sensors), but so does tolerance. Get Sensor Reading Factors must be sent with
// each raw reading; applying the linear formula using the returned conversion
// factors yields the real reading, and can the same factors can be plugged into
// the tolerance and resolution formulae to calculate them.
type Linearisation uint8

// IsLinear returns whether the underlying sensor is linear. Calling
// Lineariser() will return an error, as there is no linearisation formula (it
// is effectively a no-op). Only the linear formula in the spec needs be applied
// to obtain a real reading.
func (l Linearisation) IsLinear() bool {
	return l == LinearisationLinear
}

// IsLinearised returns whether the underlying sensor is linearised, meaning the
// value after conversion needs to be fed through a linearisation formula as a
// final step before being used. A suitable implementation of this function is
// returned by the Lineariser() method.
func (l Linearisation) IsLinearised() bool {
	return l > LinearisationLinear && l < LinearisationNonLinear
}

// IsNonLinear returns whether the underlying sensor is not consistent enough
// for the constraints of linear and linearised. As for linear sensors,
// attempting to retrieve a Lineariser will return an error. Readings from these
// sensors require Get Sensor Reading Factors to convert them into usable
// values.
func (l Linearisation) IsNonLinear() bool {
	return l >= LinearisationNonLinear
}

// Lineariser returns a suitable Lineariser implementation that will turn the
// converted raw value produced by the underlying sensor into a usable value. If
// the sensor is already linear, or non-linear, this will return
// ErrNotLinearised.
func (l Linearisation) Lineariser() (Lineariser, error) {
	if lineariser, ok := linearisationLinearisers[l]; ok {
		return lineariser, nil
	}

	return nil, ErrNotLinearised
}

func (l Linearisation) Description() string {
	if desc, ok := linearisationDescriptions[l]; ok {
		return desc
	}

	if l >= 0x71 && l <= 0x7f {
		return "Non-linear OEM"
	}

	return unknown
}

func (l Linearisation) String() string {
	return fmt.Sprintf("%#v(%v)", uint8(l), l.Description())
}

// Lineariser is implemented by formulae that can linearise a value returned by
// the Get Sensor Reading command that has gone through the linear formula
// containing M, B, K1 and K2, used for all sensors.
type Lineariser interface {
	// Linearise applies a linearisation formula to a converted value, returning
	// the final value in the correct unit. This is the last step in the "Sensor
	// Reading Conversion Formula" described in section 30.3 of IPMI v1.5 and
	// v2.0.
	Linearise(v float64) float64
}

// LineariserFunc is the type of the function in the Lineariser interface. It
// allows us to create stateless Lineariser implementations from raw functions,
// including those in the math package.
type LineariserFunc func(float64) float64

// Linearise invokes the wrapped function, passing through the input and result.
func (l LineariserFunc) Linearise(f float64) float64 {
	return l(f)
}

// SensorUnit defines the unit of a sensor. It is specified in 37.17 and 43.17
// of v1.5 and v2.0 respectively. It is an 8-bit uint on the wire.
type SensorUnit uint8

const (
	_ SensorUnit = iota
	SensorUnitCelsius
	SensorUnitFahrenheit
	SensorUnitKelvin
	SensorUnitVolts
	SensorUnitAmps
	SensorUnitWatts
	SensorUnitJoules
	SensorUnitCoulombs
	SensorUnitVoltamperes
	SensorUnitNits
	SensorUnitLumen
	SensorUnitLux
	SensorUnitCandela
	SensorUnitKilopascals
	SensorUnitPoundsPerSquareInch
	SensorUnitNewtons
	SensorUnitCubicFeetPerMinute
	SensorUnitRotationsPerMinute
	SensorUnitHertz
	SensorUnitMicroseconds
	SensorUnitMilliseconds
	SensorUnitSeconds
	SensorUnitMinutes
	SensorUnitHours
	SensorUnitDays
	SensorUnitWeeks
	SensorUnitMils
	SensorUnitInches
	SensorUnitFeet
	SensorUnitCubicInches
	SensorUnitCubicFeet
	SensorUnitMillimeters
	SensorUnitCentimeters
	SensorUnitMeters
	SensorUnitCubicCentimeters
	SensorUnitCubicMeters
	SensorUnitLiters
	SensorUnitFluidOunces
	SensorUnitRadians
	SensorUnitSteradians
	SensorUnitRevolutions
	SensorUnitCycles
	SensorUnitGravities
	SensorUnitOunces
	SensorUnitPounds
	SensorUnitFeetPounds
	SensorUnitOunceInches
	SensorUnitGauss
	SensorUnitGilberts
	SensorUnitHenry
	SensorUnitMillihenry
	SensorUnitFarad
	SensorUnitMicrofarad
	SensorUnitOhms
	SensorUnitSiemens
	SensorUnitMoles
	SensorUnitBecquerel
	SensorUnitPartsPerMillion
	_
	SensorUnitDecibels
	SensorUnitDecibelsAFilter
	SensorUnitDecibelsCFilter
	SensorUnitGray
	SensorUnitSieverts
	SensorUnitColorTempKelvin
	SensorUnitBits
	SensorUnitKilobits
	SensorUnitMegabits
	SensorUnitGigabits
	SensorUnitBytes
	SensorUnitKilobytes
	SensorUnitMegabytes
	SensorUnitGigabytes
	SensorUnitWords
	SensorUnitDwords
	SensorUnitQwords
	SensorUnitMemoryLines
	SensorUnitHits
	SensorUnitMisses
	SensorUnitRetries
	SensorUnitResets
	SensorUnitOverflows
	SensorUnitUnderruns
	SensorUnitCollisions
	SensorUnitPackets
	SensorUnitMessages
	SensorUnitCharacters
	SensorUnitErrors
	SensorUnitCorrectableErrors
	SensorUnitUncorrectableErrors
	SensorUnitFatal
	SensorUnitGrams
)

var sensorUnitSymbols = map[SensorUnit]string{
	SensorUnitCelsius:             "C",
	SensorUnitFahrenheit:          "F",
	SensorUnitKelvin:              "K",
	SensorUnitVolts:               "V",
	SensorUnitAmps:                "A",
	SensorUnitWatts:               "W",
	SensorUnitJoules:              "J",
	SensorUnitCoulombs:            "C",
	SensorUnitVoltamperes:         "VA",
	SensorUnitNits:                "nt",
	SensorUnitLumen:               "lm",
	SensorUnitLux:                 "lx",
	SensorUnitCandela:             "cd",
	SensorUnitKilopascals:         "kPa",
	SensorUnitPoundsPerSquareInch: "psi",
	SensorUnitNewtons:             "nt",
	SensorUnitCubicFeetPerMinute:  "CFM",
	SensorUnitRotationsPerMinute:  "RPM",
	SensorUnitHertz:               "Hz",
	SensorUnitMicroseconds:        "μs",
	SensorUnitMilliseconds:        "ms",
	SensorUnitSeconds:             "s",
	SensorUnitMinutes:             "min",
	SensorUnitHours:               "hr",
	SensorUnitDays:                "d",
	SensorUnitWeeks:               "w",
	SensorUnitMils:                "mil",
	SensorUnitInches:              "in",
	SensorUnitFeet:                "ft",
	SensorUnitCubicInches:         "in³",
	SensorUnitCubicFeet:           "ft³",
	SensorUnitMillimeters:         "mm",
	SensorUnitCentimeters:         "cm",
	SensorUnitMeters:              "m",
	SensorUnitCubicCentimeters:    "cm³",
	SensorUnitCubicMeters:         "m³",
	SensorUnitLiters:              "l",
	SensorUnitFluidOunces:         "fl oz",
	SensorUnitRadians:             "rad",
	SensorUnitSteradians:          "sr",
	SensorUnitRevolutions:         "rev",
	SensorUnitCycles:              "Hz",
	SensorUnitGravities:           "g",
	SensorUnitOunces:              "oz",
	SensorUnitPounds:              "lb",
	SensorUnitFeetPounds:          "ft-lb",
	SensorUnitOunceInches:         "oz-in",
	SensorUnitGauss:               "G",
	SensorUnitGilberts:            "Gb",
	SensorUnitHenry:               "H",
	SensorUnitMillihenry:          "mH",
	SensorUnitFarad:               "F",
	SensorUnitMicrofarad:          "μF",
	SensorUnitOhms:                "Ω",
	SensorUnitSiemens:             "Ω⁻¹",
	SensorUnitMoles:               "mol",
	SensorUnitBecquerel:           "Bq",
	SensorUnitPartsPerMillion:     "ppm",
	SensorUnitDecibels:            "dB",
	SensorUnitDecibelsAFilter:     "dBA",
	SensorUnitDecibelsCFilter:     "dBC",
	SensorUnitGray:                "Gy",
	SensorUnitSieverts:            "Sv",
	SensorUnitColorTempKelvin:     "ColorK",
	SensorUnitBits:                "b",
	SensorUnitKilobits:            "Kb",
	SensorUnitMegabits:            "Mb",
	SensorUnitGigabits:            "Gb",
	SensorUnitBytes:               "B",
	SensorUnitKilobytes:           "KB",
	SensorUnitMegabytes:           "MB",
	SensorUnitGigabytes:           "GB",
	SensorUnitWords:               "word",
	SensorUnitDwords:              "dword",
	SensorUnitQwords:              "qword",
	SensorUnitMemoryLines:         "memory line",
	SensorUnitHits:                "hit",
	SensorUnitMisses:              "miss",
	SensorUnitRetries:             "retry",
	SensorUnitResets:              "reset",
	SensorUnitOverflows:           "overflow",
	SensorUnitUnderruns:           "underrun",
	SensorUnitCollisions:          "collision",
	SensorUnitPackets:             "pkt",
	SensorUnitMessages:            "msg",
	SensorUnitCharacters:          "char",
	SensorUnitErrors:              "err",
	SensorUnitCorrectableErrors:   "correctable err",
	SensorUnitUncorrectableErrors: "uncorrectable err",
	SensorUnitFatal:               "fatal",
	SensorUnitGrams:               "g",
}

func (s SensorUnit) Symbol() string {
	if s == 0 {
		return "Unspecified/Unused"
	}

	if symbol, ok := sensorUnitSymbols[s]; ok {
		return symbol
	}

	return unknown
}

func (s SensorUnit) String() string {
	return s.Symbol()
}

// twos parses two's complement numbers of up to 16 bits into a native integer.
// The input is two bytes in big-endian order, and the number of bits the binary
// representation is expected to be (0 through 16). More significant bits above
// this must be 0, e.g. twos([...]byte{0b000000xx, 0bxxxxxxxx}, 10).
func twos(bigEndian [2]byte, bits uint8) int16 {
	// this abstracts away the endian-ness of the platform; big-endian only
	// refers to the byte order of the input. It is identical to
	// binary.BigEndian.Uint16(), but avoids creating a slice.
	numerical := uint16(bigEndian[1]) | uint16(bigEndian[0])<<8

	// sign extend to 16 bits
	// (https://graphics.stanford.edu/~seander/bithacks.html, "Sign extending
	// from a variable bit-width")
	mask := uint16(1) << (uint16(bits) - 1)
	numerical = (numerical ^ mask) - mask

	// make signed - same underlying bits, just different type
	return int16(numerical) //nolint:gosec
}
