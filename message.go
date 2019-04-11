package rcss

import (
	"bytes"
	"fmt"
)

type message struct {
	name        string
	values      []string
	submessages []message
}

func (c *message) AddValues(values ...string) {
	c.values = append(c.values, values...)
}

func (c *message) AddSubmessages(messages ...message) {
	c.submessages = append(c.submessages, messages...)
}

func (c message) MarshalBinary() ([]byte, error) {
	var b bytes.Buffer

	b.WriteRune('(')
	b.WriteString(c.name)

	for _, v := range c.values {
		b.WriteString(fmt.Sprintf(" %v", v))
	}

	for _, msg := range c.submessages {
		if mb, err := msg.MarshalBinary(); err != nil {
			return nil, err
		} else {
			b.WriteRune(' ')
			b.Write(mb)
		}
	}

	b.WriteRune(')')

	return b.Bytes(), nil
}

func (c *message) UnmarshalBinary(data []byte) error {
	c.name = ""
	c.values = make([]string, 0)
	c.submessages = make([]message, 0)

	r := bytes.NewBuffer(data)

	if ch, _, err := r.ReadRune(); err != nil {
		return err
	} else if '(' != ch {
		return fmt.Errorf("corrupted message")
	}

	var name bytes.Buffer
name:
	for ch, _, err := r.ReadRune(); ; ch, _, err = r.ReadRune() {
		if err != nil {
			return err
		}

		switch ch {
		case ' ':
			break name

		case ')':
			r.UnreadByte()
			break name

		default:
			name.WriteRune(ch)
		}
	}
	c.name = name.String()

	var value bytes.Buffer
values:
	for ch, _, err := r.ReadRune(); ; ch, _, err = r.ReadRune() {
		if err != nil {
			return err
		}

		switch ch {
		case '(':
			r.UnreadRune()
			break values

		case ' ':
			c.AddValues(value.String())
			value.Reset()

		case ')':
			if value.Len() > 0 {
				c.AddValues(value.String())
				value.Reset()
			}
			r.UnreadRune()
			break values

		default:
			value.WriteRune(ch)
		}
	}

messages:
	for ch, _, err := r.ReadRune(); ; ch, _, err = r.ReadRune() {
		if err != nil {
			return err
		}

		switch ch {
		case '(':
			var subbuffer bytes.Buffer
			subbuffer.WriteRune(ch)

			for hops := 1; hops > 0; {
				if ch, _, err := r.ReadRune(); err != nil {
					return err
				} else {
					subbuffer.WriteRune(ch)

					switch ch {
					case '(':
						hops++

					case ')':
						hops--
					}
				}
			}

			var submessage message
			if err := submessage.UnmarshalBinary(subbuffer.Bytes()); err != nil {
				return err
			} else {
				c.AddSubmessages(submessage)
			}

		case ')':
			r.UnreadRune()
			break messages
		}
	}

	if ch, _, err := r.ReadRune(); err != nil {
		return err
	} else if ')' != ch {
		return fmt.Errorf("corrupted message")
	}

	return nil
}
