package nip05

import (
	json "encoding/json"
	"errors"
	"unsafe"

	nostr "fiatjaf.com/nostr"
	easyjson "github.com/mailru/easyjson"
	jlexer "github.com/mailru/easyjson/jlexer"
	jwriter "github.com/mailru/easyjson/jwriter"
)

// suppress unused package warning
var (
	_ *json.RawMessage
	_ *jlexer.Lexer
	_ *jwriter.Writer
	_ easyjson.Marshaler
)

func easyjsonDecode(in *jlexer.Lexer, out *WellKnownResponse) {
	isTopLevel := in.IsStart()
	if in.IsNull() {
		if isTopLevel {
			in.Consumed()
		}
		in.Skip()
		return
	}
	in.Delim('{')
	for !in.IsDelim('}') {
		key := in.UnsafeFieldName(false)
		in.WantColon()
		if in.IsNull() {
			in.Skip()
			in.WantComma()
			continue
		}
		switch key {
		case "names":
			if in.IsNull() {
				in.Skip()
			} else {
				in.Delim('{')
				out.Names = make(map[string]nostr.PubKey)
				for !in.IsDelim('}') {
					key := string(in.String())
					in.WantColon()
					var pk nostr.PubKey
					if data := in.Raw(); in.Ok() {
						var err error
						if len(data) < 2 {
							err = errors.New("names[]{pubkey} must be a string")
						} else {
							data = data[1 : len(data)-1]
							pk, _ = nostr.PubKeyFromHex(unsafe.String(unsafe.SliceData(data), len(data)))
						}
						in.AddError(err)
					}
					if pk != nostr.ZeroPK {
						out.Names[key] = pk
					}
					in.WantComma()
				}
				in.Delim('}')
			}
		case "relays":
			if in.IsNull() {
				in.Skip()
			} else {
				in.Delim('{')
				if !in.IsDelim('}') {
					out.Relays = make(map[nostr.PubKey][]string)
				} else {
					out.Relays = nil
				}
				for !in.IsDelim('}') {
					var key nostr.PubKey
					if data := in.Raw(); in.Ok() {
						var err error
						if len(data) < 2 {
							err = errors.New("relays[pubkey] must be a string")
						} else {
							data = data[1 : len(data)-1]
							key, _ = nostr.PubKeyFromHex(unsafe.String(unsafe.SliceData(data), len(data)))
						}
						in.AddError(err)
					}
					in.WantColon()
					var relays []string
					if in.IsNull() {
						in.Skip()
						relays = nil
					} else {
						in.Delim('[')
						if relays == nil {
							if !in.IsDelim(']') {
								relays = make([]string, 0, 4)
							} else {
								relays = []string{}
							}
						} else {
							relays = (relays)[:0]
						}
						for !in.IsDelim(']') {
							relays = append(relays, string(in.String()))
							in.WantComma()
						}
						in.Delim(']')
					}
					if key != nostr.ZeroPK {
						out.Relays[key] = relays
					}
					in.WantComma()
				}
				in.Delim('}')
			}
		case "nip46":
			if in.IsNull() {
				in.Skip()
			} else {
				in.Delim('{')
				if !in.IsDelim('}') {
					out.NIP46 = make(map[nostr.PubKey][]string)
				} else {
					out.NIP46 = nil
				}
				for !in.IsDelim('}') {
					var key nostr.PubKey
					if data := in.Raw(); in.Ok() {
						var err error
						if len(data) < 2 {
							err = errors.New("nip46[pubkey] must be a string")
						} else {
							data = data[1 : len(data)-1]
							key, _ = nostr.PubKeyFromHex(unsafe.String(unsafe.SliceData(data), len(data)))
						}
						in.AddError(err)
					}
					in.WantColon()
					var bunkers []string
					if in.IsNull() {
						in.Skip()
						bunkers = nil
					} else {
						in.Delim('[')
						if bunkers == nil {
							if !in.IsDelim(']') {
								bunkers = make([]string, 0, 4)
							} else {
								bunkers = []string{}
							}
						} else {
							bunkers = (bunkers)[:0]
						}
						for !in.IsDelim(']') {
							var bunker string
							bunker = string(in.String())
							bunkers = append(bunkers, bunker)
							in.WantComma()
						}
						in.Delim(']')
					}
					if key != nostr.ZeroPK {
						out.NIP46[key] = bunkers
					}
					in.WantComma()
				}
				in.Delim('}')
			}
		default:
			in.SkipRecursive()
		}
		in.WantComma()
	}
	in.Delim('}')
	if isTopLevel {
		in.Consumed()
	}
}

func easyjsonEncode(out *jwriter.Writer, in WellKnownResponse) {
	out.RawByte('{')
	first := true
	_ = first
	{
		const prefix string = ",\"names\":"
		out.RawString(prefix[1:])
		if in.Names == nil && (out.Flags&jwriter.NilMapAsEmpty) == 0 {
			out.RawString(`null`)
		} else {
			out.RawByte('{')
			isFirst := true
			for name, pk := range in.Names {
				if isFirst {
					isFirst = false
				} else {
					out.RawByte(',')
				}
				out.String(name)
				out.RawByte(':')
				out.String(pk.Hex())
			}
			out.RawByte('}')
		}
	}
	if len(in.Relays) != 0 {
		const prefix string = ",\"relays\":"
		out.RawString(prefix)
		{
			out.RawByte('{')
			isFirst := true
			for pk, relays := range in.Relays {
				if isFirst {
					isFirst = false
				} else {
					out.RawByte(',')
				}
				out.String(pk.Hex())
				out.RawByte(':')
				if relays == nil && (out.Flags&jwriter.NilSliceAsEmpty) == 0 {
					out.RawString("null")
				} else {
					out.RawByte('[')
					for i, relay := range relays {
						if i > 0 {
							out.RawByte(',')
						}
						out.String(relay)
					}
					out.RawByte(']')
				}
			}
			out.RawByte('}')
		}
	}
	if len(in.NIP46) != 0 {
		const prefix string = ",\"nip46\":"
		out.RawString(prefix)
		{
			out.RawByte('{')
			isFirst := true
			for pk, bunkers := range in.NIP46 {
				if isFirst {
					isFirst = false
				} else {
					out.RawByte(',')
				}
				out.String(pk.Hex())
				out.RawByte(':')
				if bunkers == nil && (out.Flags&jwriter.NilSliceAsEmpty) == 0 {
					out.RawString("null")
				} else {
					out.RawByte('[')
					for i, bunker := range bunkers {
						if i > 0 {
							out.RawByte(',')
						}
						out.String(bunker)
					}
					out.RawByte(']')
				}
			}
			out.RawByte('}')
		}
	}
	out.RawByte('}')
}

// MarshalJSON supports json.Marshaler interface
func (v WellKnownResponse) MarshalJSON() ([]byte, error) {
	w := jwriter.Writer{}
	easyjsonEncode(&w, v)
	return w.Buffer.BuildBytes(), w.Error
}

// MarshalEasyJSON supports easyjson.Marshaler interface
func (v WellKnownResponse) MarshalEasyJSON(w *jwriter.Writer) {
	easyjsonEncode(w, v)
}

// UnmarshalJSON supports json.Unmarshaler interface
func (v *WellKnownResponse) UnmarshalJSON(data []byte) error {
	r := jlexer.Lexer{Data: data}
	easyjsonDecode(&r, v)
	return r.Error()
}

// UnmarshalEasyJSON supports easyjson.Unmarshaler interface
func (v *WellKnownResponse) UnmarshalEasyJSON(l *jlexer.Lexer) {
	easyjsonDecode(l, v)
}
