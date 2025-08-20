### B3 Reader — Ingestão e Agregação de Negociações da B3

Aplicação backend (Go) para ingerir, processar e expor dados agregados de negociações da B3 (ações). O projeto foi desenvolvido para o desafio de backend e atende aos requisitos do PDF anexado: ingestão eficiente dos últimos 7 dias, persistência otimizada, consulta via API REST com único endpoint e resposta agregada com max_range_value e max_daily_volume.

Este README é o guia único de execução. Ele cobre: como configurar o ambiente, como construir e rodar a ingestão, como subir a API, como consultar os dados e como validar performance. Também explica a arquitetura, modelagem de dados e decisões de engenharia.

> Imagens e fluxogramas, com documentação conjunta (em inglês). Pode ser encontrada aqui:
[B3 Reader Doc.](https://deepwiki.com/gurodrigues-dev/b3-reader/2-getting-started)

---

### Guia de reprodutibilidade

1) Clonar o repositório
```bash
git clone git@github.com:gurodrigues-dev/b3-reader.git b3-reader
cd b3-reader
```

2) Criar network
```bash
docker network create bubble
```

3) Baixar os CSVs e colocá-los em input/

4) Construir as imagens (API e ingestor)
```bash
make ingestion-logs
```
> Para verificar o motivo dos demais arquivos. Acesse `Comandos e como rodar`

5) Aguarde a finalização do ingestor e incialização da API.

6) Testar endpoint da API
```bash
curl -s "http://127.0.0.1:8080/api/v1/trades?ticker=PETR4" | jq .
```

---

#### Sumário

- Notas
- Contexto e objetivos
- Arquitetura e organização do projeto
- Fluxo de dados ponta a ponta
- Modelagem de banco de dados e migrações
- Ingestor de dados
- API REST e contrato de resposta
- Comandos e como rodar
- Validação da ingestão e benchmark

---

### Notas

Acredito que decisões melhores poderiam ter sido tomadas, tanto em nível de infraestrutura quanto de código. Meu objetivo foi entregar a melhor solução possível dentro do tempo disponível. Como estou em um modelo de trabalho híbrido e cursando pós-graduação simultaneamente, o tempo ficou bastante limitado. Por esse motivo, optei por não adicionar mais ferramentas e evitar aumentar o acoplamento. Busquei atingir todos os objetivos com a menor quantidade possível de componentes, implementando um ingestor e uma API performática, com índices bem planejados no banco de dados.

Quanto à ingestão, embora atualmente utilize a função ReadAll que consome mais memória, o processamento é feito por arquivo e os dados são transferidos por canais, permitindo comunicação entre goroutines, além da separação em lotes. Uma alternativa superior, do ponto de vista de performance, seria utilizar leitura com buffer em modo “streaming”, o que reduziria chamadas de sistema (syscalls) e melhoraria o throughput, além de favorecer um uso incremental de memória.

Outra abordagem bastante eficiente seria adotar um pipeline com workers dedicados para parsing e workers para escrita, maximizando o uso de CPU/IO. No entanto, apesar de mais performáticas, essas soluções são mais verbosas e demandariam um estudo mais aprofundado e uma bateria de testes, o que não era viável no momento.

Do ponto de vista de infraestrutura, considero que o principal aprimoramento seria introduzir um cache (por exemplo, Redis) para evitar recálculo de agregações mais acessadas. Porém, dada a boa performance observada nas consultas diretamente no banco, essa otimização não se mostrou crítica para alcançar um benchmark satisfatório na API neste contexto.

Em resumo, priorizei simplicidade arquitetural, baixo acoplamento e atendimento aos requisitos do desafio, deixando melhorias de performance mais avançadas como streaming, paralelismo por estágios e cache mapeadas como evolução futura.

### Contexto e objetivos

Com base no desafio, a solução deve:
- Ingerir arquivos CSV de negociações de ações da B3 referentes aos últimos 7 dias úteis. O download pode ser manual; coloque os arquivos em um diretório local para processamento.
- Persistir os campos relevantes em um banco relacional (PostgreSQL).
- Expor um único endpoint REST que, para um ticker e um período (opcional), retorne:
  - ticker: o código consultado
  - max_range_value: maior PrecoNegocio no período filtrado
  - max_daily_volume: maior volume diário consolidado no período filtrado
- Executar a ingestão completa em menos de 15 minutos em máquina padrão (Docker, 16GB RAM, 6 cores), com foco em eficiência (batching, I/O, índices).

### Arquitetura e organização do projeto

Arquitetura: o projeto segue uma abordagem inspirada em Clean Architecture, com separação clara entre domínio, casos de uso, portas/interfaces e adaptadores. Os pontos principais:
- Camada de Domínio (pacote trade): contém as entidades (Trade), contratos (interfaces Repository, Reader, Writer, Usecase) e a orquestração de regras de negócio no Service. Essa camada não conhece detalhes de infraestrutura (banco, HTTP).
- Portas e Casos de Uso: Usecase no pacote trade define dois casos de uso, IngestFiles e GetAggregatedData. O Service implementa esses casos de uso consumindo interfaces (Repository, Reader).
- Adaptadores de Entrada: internal/controllers expõe o caso de uso via HTTP (API REST). internal/reader implementa a leitura de arquivos CSV (CLI/serviço de ingestão).
- Adaptadores de Saída: trade/storage implementa o Repository usando PostgreSQL (pgx, CopyFrom).
- Entrypoints: cmd/api/main.go sobe a API, cmd/ingestor/main.go executa a ingestão. Cada um injeta as dependências concretas no domínio.

Benefícios: dependências apontam para dentro (domínio), o que facilita testes, troca de adaptadores (ex.: outro banco) e manutenibilidade. A inversão de dependência é aplicada por meio de interfaces no pacote de domínio (trade).

### Fluxo de dados ponta a ponta

1) Você baixa os CSVs da B3 manualmente e os coloca no diretório configurado (por padrão, input/). Os arquivos do exemplo usam “;” como separador, possuem cabeçalho e seguem a estrutura fixa do site da B3.

2) O serviço ingestor (cmd/ingestor) carrega as variáveis de ambiente, aplica migrações de banco (via golang-migrate) e inicializa o CSVReader (internal/reader.CSVReader) com o caminho dos arquivos.

3) O CSVReader detecta se o caminho é diretório ou arquivo único. Para diretório, percorre recursivamente (filepath.Walk) e, para cada arquivo, lê tudo em memória e envia as linhas para um canal. O cabeçalho é descartado posteriormente.

4) O caso de uso IngestFiles (trade.Service) consome os registros do canal, usa o parseTrade (trade/parsers.go) para mapear para []trade.Trade, cria lotes (internal/batcher, tamanho 5000) e persiste via Repository.SaveBatch.

5) O repositório (trade/storage.TradeRepository) usa pgx CopyFrom para inserir em alta performance na tabela trades com as colunas: data_negocio, codigo_instrumento, preco_negocio, quantidade_negociada, hora_fechamento, created_at.

6) A API (cmd/api) expõe um único endpoint REST. O controller lê os filtros, chama GetAggregatedData (Service) que consulta o repositório para max_range_value (maior preço unitário) e max_daily_volume (maior volume consolidado por dia) do ticker no período.

> Não adicionei arquivos na pasta de input. Visto o tamanho dos mesmos.

### Modelagem de banco de dados e migrações

Tabela alvo: trades. Colunas utilizadas pela aplicação (conforme entidades):
- data_negocio (DATE ou TIMESTAMP, dependendo da migração)
- codigo_instrumento (VARCHAR/ TEXT)
- preco_negocio (NUMERIC/DOUBLE PRECISION conforme escolha)
- quantidade_negociada (INTEGER/BIGINT)
- hora_fechamento (VARCHAR) no formato HHMMSSmmm
- created_at (TIMESTAMP)

Índices criados especificamente para otimizar as consultas e manter um bom equilíbrio entre escrita e leitura:
```sql
BEGIN;
CREATE INDEX idx_trades_ticker_date_preco ON trades (codigo_instrumento, data_negocio, preco_negocio);
CREATE INDEX idx_trades_ticker_date_qtd   ON trades (codigo_instrumento, data_negocio, quantidade_negociada);
COMMIT;

BEGIN;
CREATE INDEX idx_trades_instrumento_data ON trades (codigo_instrumento, data_negocio);
CREATE INDEX idx_trades_data             ON trades (data_negocio);
COMMIT;

BEGIN;
CREATE INDEX idx_trades_brin_date ON trades USING brin (data_negocio);
COMMIT;
```

Racional dos índices: consultas filtram por codigo_instrumento e data_negocio, e agregam max(preco_negocio) por período, além da soma diária de quantidade_negociada. Índices compostos e o BRIN em data_negocio ajudam em períodos longos e em ordenação temporal do armazenamento físico.

As migrações são aplicadas automaticamente quando o ingestor inicia (cmd/ingestor/main.go). Você também pode aplicá-las manualmente com a CLI do migrate, se preferir.

Observação técnica sobre a query: a implementação atual usa uma combinação de GROUP BY data_negocio e janela para volume. Como melhoria recomendada (para garantir retorno de linha única e sem ambiguidade), sugerimos a forma abaixo no repositório:

```sql
WITH daily AS (
  SELECT
    data_negocio,
    MAX(preco_negocio)                    AS daily_max_price,
    SUM(quantidade_negociada)::bigint     AS daily_volume
  FROM trades
  WHERE codigo_instrumento = \$1
    AND data_negocio >= \$2
  GROUP BY data_negocio
)
SELECT
  MAX(daily_max_price) AS max_range_value,
  MAX(daily_volume)    AS max_daily_volume
FROM daily;
```

### Ingestor de dados

O ingestor lê um diretório ou arquivo único e processa todos os CSVs, removendo o cabeçalho, parseando registros, loteando e persistindo via CopyFrom. O batching é de 5000 registros (constante batchSize), ajustável no código para calibrar throughput e uso de memória.

Formato esperado dos CSVs, conforme a B3 (ordem de colunas fixa). Campos relevantes:

DataNegocio
CodigoInstrumento
PrecoNegocio
QuantidadeNegociada
HoraFechamento (HHMMSSmmm)
Separador padrão: ‘;’. O CSVReader é inicializado com sep ‘;’ e FieldsPerRecord = -1, tolerante a variações de colunas extras não utilizadas.

Idempotência na ingestão: ver seção “Idempotência, confiabilidade e melhorias” para recomendações de constraints/estratégias.

### API Rest e contrato de resposta

Formato esperado dos CSVs, conforme a B3 (ordem de colunas fixa). Campos relevantes:

DataNegocio
CodigoInstrumento
PrecoNegocio
QuantidadeNegociada
HoraFechamento (HHMMSSmmm)
Separador padrão: ‘;’. O CSVReader é inicializado com sep ‘;’ e FieldsPerRecord = -1, tolerante a variações de colunas extras não utilizadas.

Idempotência na ingestão: ver seção “Idempotência, confiabilidade e melhorias” para recomendações de constraints/estratégias.

API REST e contrato de resposta
A API expõe um único endpoint que recebe:

ticker (obrigatório): como query string
data_inicio (opcional): data ISO-8601 (ex.: 2025-08-06). Se omitido, o período padrão considera os últimos 7 dias até a véspera da data atual, conforme o desafio.
Resposta JSON:

```json
{
  "ticker": "PETR4",
  "max_range_value": 20.50,
  "max_daily_volume": 150000
}
```
Definições da agregação:

- max_range_value: maior PrecoNegocio para o ticker no período filtrado.
- max_daily_volume: maior soma diária de QuantidadeNegociada para o ticker no período filtrado.
Documentação OpenAPI/Swagger: o repositório contém docs/swagger.yaml e docs/swagger.json. Se a API estiver servindo Swagger em runtime, utilize a URL e rota expostas pelo serviço. Em alternativa, importe o arquivo swagger.yaml em um visualizador de sua preferência e execute a chamada pelo próprio UI do Swagger.


### Comandos e como rodar

O Makefile fornece atalhos para tarefas comuns de desenvolvimento, CI e operação (build local, lint, testes, segurança e orquestração com Docker Compose). Para executar qualquer alvo, use:

make <alvo>

Exemplos: make test, make api, make ingestion-logs

Pré-requisitos:
- Go instalado (para alvos de build/test/lint/vet)
- Docker + Docker Compose (para alvos api/ingestion)
- golangci-lint e govulncheck instalados se você for usar lint e vulncheck
  - go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
  - go install golang.org/x/vuln/cmd/govulncheck@latest

Alvos disponíveis:

- test
  Executa testes unitários e gera um arquivo de cobertura cp.out, ignorando os pacotes cmd e config (foco no core da aplicação).
  Internamente: go test -short -coverprofile=cp.out $(go list ./... | grep -vE cmd|config)
  Quando usar: durante o desenvolvimento e em CI para validar alterações rapidamente.
  Exemplo:
    make test
  Dica: para ver o HTML de cobertura depois do cp.out:
    go tool cover -html=cp.out -o coverage.html

- tidy
  Ajusta dependências do módulo Go, removendo as não utilizadas e adicionando as necessárias.
  Internamente: go mod tidy
  Quando usar: após adicionar/remover imports ou atualizar versões manualmente.
  Exemplo:
    make tidy

- lint
  Roda o linter (golangci-lint) em todo o repositório, reforçando padrões de qualidade.
  Internamente: golangci-lint run ./...
  Quando usar: antes de abrir PR, em CI ou localmente para checagens de estilo/bugs comuns.
  Exemplo:
    make lint

- update
  Atualiza dependências para as últimas versões disponíveis e roda go mod tidy em seguida.
  Internamente: go get -u ./... && go mod tidy
  Quando usar: periodicamente para manter libs atualizadas, ou antes de releases.
  Exemplo:
    make update

- vulncheck
  Executa análise de vulnerabilidades conhecidas em dependências do projeto.
  Internamente: govulncheck ./...
  Quando usar: rotineiramente, antes de deploys, ou integrado no pipeline de CI.
  Exemplo:
    make vulncheck

- vet
  Roda go vet para detectar construções suspeitas/erros comuns.
  Internamente: go vet ./...
  Quando usar: junto com lint/test para saúde geral do código.
  Exemplo:
    make vet

- network
  Cria a rede Docker externa $(NETWORK) caso não exista.
  Internamente: checa se a rede existe e cria com docker network create $(NETWORK) se necessário.
  Requer: ter definido a variável de ambiente NETWORK, por exemplo:
    export NETWORK=bubble
  Quando usar: antes do primeiro docker compose up, se o compose referencia networks.external: true.
  Exemplo:
    export NETWORK=bubble
    make network

- build-api
  Compila o binário da API para bin/api a partir de cmd/api/main.go.
  Internamente: go build -o bin/api cmd/api/main.go
  Quando usar: build local sem Docker, para execução direta.
  Exemplo:
    make build-api
    ./bin/api

- build-ingestor
  Compila o binário do ingestor para bin/ingestor a partir de cmd/ingestor/main.go.
  Internamente: go build -o bin/ingestor cmd/ingestor/main.go
  Quando usar: build local do ingestor, execução direta.
  Exemplo:
    make build-ingestor
    ./bin/ingestor

- ingestion
  Sobe apenas o serviço de ingestão via Docker Compose usando o profile ingestion em modo daemon (-d).
  Internamente: docker compose --profile ingestion up -d
  Pré-requisitos:
    - Docker/Compose funcionando
    - A rede externa (se houver) criada (veja make network)
    - Arquivos de entrada montados conforme docker-compose.yml (ex.: ./input:/input)
    - Variáveis de ambiente (.env) válidas (ex.: DATABASE_URL e FILE_PATH)
  Quando usar: para rodar a ingestão de dados (migrações + leitura de CSV + persistência).
  Exemplo:
    make ingestion
  Dica: verifique os logs com make ingestion-logs (modo foreground) para acompanhar o progresso.

- ingestion-logs
  Sobe o serviço de ingestão mostrando os logs no foreground (sem -d), útil para debug e acompanhamento em tempo real.
  Internamente: docker compose --profile ingestion up
  Quando usar: durante desenvolvimento ou primeira execução para ver o andamento e erros.
  Exemplo:
    make ingestion-logs
  Para encerrar: Ctrl+C (ou docker compose down em outro terminal, se precisar).

- api
  Sobe a API e dependências definidas no docker-compose.yml em modo daemon.
  Internamente: docker compose up -d
  Pré-requisitos:
    - Banco de dados saudável (o compose já aguarda saúde do Postgres)
    - DATABASE_URL no .env apontando para o serviço do Postgres do compose
  Quando usar: para disponibilizar o endpoint REST após a ingestão.
  Exemplo:
    make api
  Teste:
    curl "http://127.0.0.1:8080/api/v1/trades?ticker=PETR4&data_inicio=2025-08-06"

- api-logs
  Sobe a API em foreground (sem -d), exibindo logs de inicialização no terminal.
  Internamente: docker compose up
  Quando usar: debug, primeira subida, inspeção de logs no console.
  Exemplo:
    make api-logs
  Para encerrar: Ctrl+C (ou docker compose down em outro terminal).

Fluxos recomendados:

- Primeiro setup com Docker
  1) Crie a rede externa:
     export NETWORK=bubble
     make network
  2) Suba o banco e a ingestão (profile ingestion):
     make ingestion-logs   # ou make ingestion para rodar em background
  3) Suba a API:
     make api
  4) Teste o endpoint:
     curl "http://127.0.0.1:8080/api/v1/trades?ticker=PETR4"

- Desenvolvimento local (sem Docker)
  1) Rode testes/lint:
     make test
     make vet
     make lint
  2) Build binários:
     make build-ingestor
     make build-api
  3) Execute os binários:
     ./bin/ingestor
     ./bin/api

Notas:
- Os alvos ingestion e api dependem do docker-compose.yml na raiz do projeto.
- Se você estiver no macOS/Windows, ajuste os recursos do Docker Desktop (CPUs/RAM) em Settings > Resources para um desempenho adequado.
- Se o compose utiliza uma rede externa (networks: bubble: external: true), a rede precisa existir antes (use make network com NETWORK=bubble).

### Validação da ingestão e benchmark

Para garantir a correção e a eficiência do pipeline de ingestão, realizamos um teste completo utilizando 7 arquivos de negociações (últimos 7 dias úteis), com as seguintes quantidades de linhas por arquivo:

- 06-08-2025: 9.545.228 linhas
- 07-08-2025: 9.910.228 linhas
- 08-08-2025: 9.364.308 linhas
- 11-08-2025: 8.683.494 linhas
- 12-08-2025: 10.204.893 linhas
- 13-08-2025: 9.491.808 linhas
- 14-08-2025: 9.663.537 linhas

Total processado: 66.863.496 linhas.

Metodologia de validação:
- Após a ingestão, foi executado um COUNT(*) na tabela de destino (trades) para verificar a correspondência exata com o total esperado. O resultado do COUNT(*) coincidiu com o somatório das linhas dos arquivos, validando que todos os registros foram devidamente persistidos, sem perdas ou duplicidades nesse cenário de teste.
- A ingestão foi feita via serviço ingestor com batching (5.000 registros) e CopyFrom (pgx), com índices conforme descrito na seção de modelagem e tuning.

Performance observada:
- Tempo de ingestão: aproximadamente 15 minutos para os 7 arquivos (≈ 66,86 milhões de linhas) em uma máquina de desenvolvimento padrão (Docker, 16 GB de RAM, 6 cores).
- Tempo de resposta da API: entre 100 ms e 1 s nas consultas agregadas típicas, dependendo do ticker, do intervalo de datas e do aquecimento do cache do banco de dados.

Notas:
- Em execuções subsequentes, o tempo da API tende a melhorar para consultas repetidas (efeito de cache do Postgres e do SO).
- Caso sua máquina tenha menos recursos, considere reduzir concorrência e/ou ajustar o batch size para manter estabilidade; com mais recursos (CPU/SSD rápidos), é possível reduzir o tempo total de ingestão.


