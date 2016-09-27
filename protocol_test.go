package delta

import (
	"bufio"
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"testing"
	"time"
)

func newWriter(w io.Writer) io.Writer {
	return deltaFrame{w: w}
}

func newReader(r io.Reader) io.Reader {
	return deltaFrame{r: r}
}

func protocolwithTransit(in []byte) ([]byte, []byte, error) {
	var b bytes.Buffer
	w := bufio.NewWriter(&b)
	xw := newWriter(w)

	_, err := xw.Write(in)
	w.Flush()
	if err != nil {
		return nil, nil, err
	}
	transit := b.Bytes()
	xr := newReader(&b)

	readback, err := ioutil.ReadAll(xr)
	if err != nil {
		return nil, nil, err
	}
	return readback, transit, err
}

func loopback() (Framer, chan []byte, chan []byte) {
	deltaReader, clientWriter := io.Pipe()
	clientReader, deltaWriter := io.Pipe()
	client := deltaFrame{clientReader, clientWriter}
	delta := deltaFrame{deltaReader, deltaWriter}
	deltaSend := make(chan []byte)
	deltaRecive := make(chan []byte)

	go func() {
		b := make([]byte, 256)
		n, err := delta.Read(b)
		if err != nil {
			log.Printf("Ping Read Error %v, number of bytes %d\n", err, n)
		}
		// for {
		for n == 0 {
			n, err = delta.Read(b)
			if err != nil {
				log.Printf("Ping Read Error %v, number of bytes %d\n", err, n)
				log.Printf("bytes %x\n", b)
			}
		}
		deltaRecive <- b[:n]

		for {
			select {
			case <-time.After(time.Millisecond * 1000):
				return
			case p := <-deltaSend:
				n, err = delta.Write(p)
				if err != nil {
					log.Printf("Ping Read Error %v, number of bytes %d\n", err, n)
					log.Printf("bytes %x\n", b)
				}
			}
		}

	}()

	return client, deltaSend, deltaRecive
}

func TestDeltaWrite(t *testing.T) {

	cases := []struct {
		b      []byte
		n      int
		err    error
		writen []byte
	}{
		{[]byte{0x06, 0x01, 0x04, 0x10, 0x03, 0x00, 0x0a}, 11, nil, []byte{0x02, 0x06, 0x01, 0x04, 0x10, 0x03, 0x00, 0x0a, 0xe2, 0xd6, 0x03}},
		{[]byte{0x06, 0x01, 0x06, 0x13, 0x03, 0x00, 0x00, 0x40, 0x74}, 13, nil, []byte{0x02, 0x06, 0x01, 0x06, 0x13, 0x03, 0x00, 0x00, 0x40, 0x74, 0xfb, 0x28, 0x03}},
		{[]byte{0x06, 0x01, 0x04, 0x10, 0x09, 0x00, 0x20}, 11, nil, []byte{0x02, 0x06, 0x01, 0x04, 0x10, 0x09, 0x00, 0x20, 0x43, 0x0b, 0x03}},
	}
	for _, c := range cases {
		var b bytes.Buffer
		w := bufio.NewWriter(&b)
		x := newWriter(w)
		n, err := x.Write(c.b)
		w.Flush()

		if err != c.err {
			t.Errorf("Expected: %v, got %v\n", c.err, err)
		}
		if n != c.n {
			t.Errorf("Expected: %d, got %d\n", c.n, n)
		}
		if string(b.Bytes()) != string(c.writen) {
			t.Errorf("Expected: %02x, got %02x\n", c.writen, b.Bytes())
		}
	}
}

func TestDeltaRead(t *testing.T) {

	cases := []struct {
		readback []byte
		err      error
		writeout []byte
	}{
		{[]byte{0x06, 0x01, 0x04, 0x10, 0x03, 0x00, 0x0a}, nil, []byte{0x02, 0x06, 0x01, 0x04, 0x10, 0x03, 0x00, 0x0a, 0xe2, 0xd6, 0x03}},
		{[]byte{0x06, 0x01, 0x04, 0x10, 0x09, 0x00, 0x20}, nil, []byte{0x02, 0x06, 0x01, 0x04, 0x10, 0x09, 0x00, 0x20, 0x43, 0x0b, 0x03}},
		{[]byte{0x06, 0x01, 0x06, 0x13, 0x03, 0x00, 0x00, 0x40, 0x74}, nil, []byte{0x02, 0x06, 0x01, 0x06, 0x13, 0x03, 0x00, 0x00, 0x40, 0x74, 0xfb, 0x28, 0x03}},
		{[]byte{0x06, 0x01, 0x04, 0x10, 0x03, 0x00, 0x0a}, errorPacketNoStartByte, []byte{0x06, 0x01, 0x04, 0x10, 0x03, 0x00, 0x0a, 0xe2, 0xd6, 0x03}},
		{[]byte{0x06, 0x01, 0x04, 0x10, 0x03, 0x00, 0x0a}, errorPacketNotEndbyte, []byte{0x02, 0x06, 0x01, 0x04, 0x10, 0x03, 0x00, 0x0a, 0xe2, 0xd6}},
		{[]byte{0x06, 0x01, 0x04, 0x10, 0x03, 0x00, 0x0a}, errorPacketBadCRC, []byte{0x02, 0x06, 0x01, 0x04, 0x10, 0x03, 0x00, 0x0a, 0xe2, 0xd8, 0x03}},
	}

	for _, c := range cases {
		r := bytes.NewReader(c.writeout)
		x := newReader(r)

		readback, err := ioutil.ReadAll(x)

		if err != c.err {
			t.Errorf("Expected: %s, got %s\n", c.err, err)
		}

		if err == nil {
			if string(readback) != string(c.readback) {
				t.Errorf("Expected: %x, got %x\n", c.readback, readback)
			}
		}
	}
}
