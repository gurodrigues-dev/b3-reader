package trade

import (
	"reflect"
	"testing"
	"time"
)

func TestParseHoraFechamento(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		want      string
		expectErr bool
	}{
		{"vaild hour", "123456", "12:34:56", false},
		{"short time", "1234", "", true},
		{"just zeros", "000000", "00:00:00", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseHoraFechamento(tt.input)
			if (err != nil) != tt.expectErr {
				t.Errorf("expected error=%v, but=%v (%v)", tt.expectErr, err != nil, err)
			}
			if got != tt.want {
				t.Errorf("expected %q, obtained %q", tt.want, got)
			}
		})
	}
}

func TestParseTrade(t *testing.T) {
	tests := []struct {
		name      string
		records   [][]string
		wantErr   bool
		expectLen int
	}{
		{
			name: "valid trade",
			records: [][]string{
				{"header"},
				{"", "PETR4", "", "10,50", "1000", "123456", "", "", "2024-08-16"},
			},
			wantErr:   false,
			expectLen: 1,
		},
		{
			name: "invalid datetime",
			records: [][]string{
				{"header"},
				{"", "VALE3", "", "50,00", "500", "123456", "", "", "16-08-2024"},
			},
			wantErr: true,
		},
		{
			name: "invalid price",
			records: [][]string{
				{"header"},
				{"", "ITUB4", "", "abc", "200", "123456", "", "", "2024-08-16"},
			},
			wantErr: true,
		},
		{
			name: "invalid quantity",
			records: [][]string{
				{"header"},
				{"", "BBDC3", "", "20,00", "dez", "123456", "", "", "2024-08-16"},
			},
			wantErr: true,
		},
		{
			name: "invalid hour",
			records: [][]string{
				{"header"},
				{"", "BBAS3", "", "30,00", "200", "1234", "", "", "2024-08-16"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trades, err := parseTrade(tt.records)

			if (err != nil) != tt.wantErr {
				t.Fatalf("expected error=%t, but obtained=%v", tt.wantErr, err)
			}

			if !tt.wantErr {
				if len(trades) != tt.expectLen {
					t.Errorf("expected %d trades, obtained %d", tt.expectLen, len(trades))
				}

				if len(trades) > 0 {
					trade := trades[0]
					if trade.CodigoInstrumento != "PETR4" {
						t.Errorf("expect CodigoInstrumento=PETR4, obtained=%s", trade.CodigoInstrumento)
					}
					if !reflect.DeepEqual(trade.HoraFechamento, "12:34:56") {
						t.Errorf("expect HoraFechamento=12:34:56, obtained=%s", trade.HoraFechamento)
					}
					wantDate, _ := time.Parse("2006-01-02", "2024-08-16")
					if !trade.DataNegocio.Equal(wantDate) {
						t.Errorf("expect DataNegocio=%v, obtained=%v", wantDate, trade.DataNegocio)
					}
				}
			}
		})
	}
}
