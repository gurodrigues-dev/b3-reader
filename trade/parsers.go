package trade

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func parseTrade(records [][]string) ([]Trade, error) {
	var trades []Trade

	for i, record := range records {
		if i == 0 {
			continue
		}

		dataNegocio, err := time.Parse("2006-01-02", record[8])
		if err != nil {
			return nil, fmt.Errorf("parse error data_negocio at row %d: %w", i+1, err)
		}

		precoNegocioStr := strings.ReplaceAll(record[3], ",", ".")
		precoNegocio, err := strconv.ParseFloat(precoNegocioStr, 64)
		if err != nil {
			return nil, fmt.Errorf("parse error preco_negocio at row %d: %w", i+1, err)
		}

		quantidadeStr := strings.ReplaceAll(record[4], ",", "")
		quantidadeNegociada, err := strconv.Atoi(quantidadeStr)
		if err != nil {
			return nil, fmt.Errorf("erro ao analisar quantidade_negociada: %w", err)
		}

		horaFechamento, err := parseHoraFechamento(record[5])
		if err != nil {
			return nil, fmt.Errorf("parse error hora_fechamento at row %d: %w", i+1, err)
		}

		trade := Trade{
			DataNegocio:         dataNegocio,
			CodigoInstrumento:   record[1],
			PrecoNegocio:        precoNegocio,
			QuantidadeNegociada: quantidadeNegociada,
			HoraFechamento:      horaFechamento,
			CreatedAt:           time.Now(),
		}

		trades = append(trades, trade)
	}

	return trades, nil
}

func parseHoraFechamento(horaStr string) (string, error) {
	if len(horaStr) < 6 {
		return "", fmt.Errorf("invalid hora_fechamento: %s", horaStr)
	}

	hora := horaStr[:2]
	minuto := horaStr[2:4]
	segundo := horaStr[4:6]

	return fmt.Sprintf("%s:%s:%s", hora, minuto, segundo), nil
}
