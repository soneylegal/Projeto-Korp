# Desafio DevOps — Projeto Korp

Solução para o desafio técnico de DevOps da Korp: um microsserviço HTTP em Go, conteinerizado com Docker Compose, monitorado com Prometheus e Grafana, e provisionado por um playbook Ansible.

---

## Arquitetura

```
                                    ┌─── Rede korp-net ───────────────┐
                                    │                                 │
  Cliente ──── porta 80 ──────────► │  Nginx (reverse proxy)          │
                                    │       │                         │
                                    │       ▼ proxy_pass :8080        │
                                    │  http-server-projeto-korp (Go)  │
                                    │       │ /metrics                │
                                    │       ▼                         │
  Navegador ── porta 9090 ────────► │  Prometheus                     │
                                    │       │                         │
                                    │       ▼ query                   │
  Navegador ── porta 3000 ────────► │  Grafana                        │
                                    └─────────────────────────────────┘
```

- **http-server-projeto-korp**: Aplicação Go na porta interna 8080. Expõe `GET /projeto-korp` (JSON com nome e horário UTC) e `GET /metrics` (contadores no padrão Prometheus). Não mapeia portas para o host — só é acessível via Nginx pela rede Docker.
- **Nginx**: Reverse proxy na porta 80. Encaminha requisições para a aplicação Go.
- **Prometheus**: Coleta métricas do endpoint `/metrics` a cada 15 segundos.
- **Grafana**: Exibe o dashboard com disponibilidade (`up`) e volume de requisições (`rate(http_requests_total[1m])`). O datasource e o dashboard são provisionados via arquivos de configuração no repositório, sem configuração manual na UI.

---

## Estrutura do Repositório

```
.
├── ansible.cfg                                # Configuração do Ansible
├── inventory.ini                              # Inventário de hosts
├── playbook.yml                               # Playbook — provisiona todo o ambiente
├── requirements.yml                           # Collection community.docker
├── docker-compose.yml                         # Orquestração dos 4 containers
├── http-server-projeto-korp/
│   ├── Dockerfile                             # Multi-stage build (golang → alpine)
│   ├── go.mod                                 # Dependências Go
│   └── main.go                                # Código da API e instrumentação de métricas
├── nginx/
│   └── http-server-projeto-korp.conf          # Configuração do proxy reverso
├── prometheus/
│   └── prometheus.yml                         # Configuração de scrape
└── grafana/
    └── provisioning/
        ├── datasources/
        │   └── datasource.yml                 # Datasource Prometheus (automático)
        └── dashboards/
            ├── dashboards.yml                 # Provider de dashboards
            └── http-server-projeto-korp-dashboard.json
```

---

## Como executar

### Pré-requisitos

- Ansible (`ansible-core >= 2.15`) instalado na máquina de controle.
- Acesso SSH com `sudo` ao servidor de destino (testado em Ubuntu 24.04 LTS).

### 1. Instalar a collection do Ansible

```bash
ansible-galaxy collection install -r requirements.yml
```

### 2. Configurar o inventário

Edite `inventory.ini` com o IP e a chave SSH do servidor:

```ini
[korp]
korp-server ansible_host=SEU_IP_AQUI ansible_user=ubuntu ansible_ssh_private_key_file=~/.ssh/sua_chave
```

### 3. Executar o playbook

```bash
ansible-playbook -i inventory.ini playbook.yml
```

O playbook executa, nesta ordem:
1. Instalação do Docker Engine e Docker Compose via repositório oficial.
2. Instalação do SDK Python do Docker via pip (com `--break-system-packages` para compatibilidade com o PEP 668 do Ubuntu 24.04).
3. Clone do repositório em `/opt/projeto-korp`.
4. Criação da rede Docker bridge `korp-net`.
5. Build da imagem Go e subida dos 4 containers via `docker compose`.
6. Validação: requisição HTTP ao endpoint `/projeto-korp` com exibição do JSON no console.

---

## Validação

### Endpoint da aplicação

```bash
curl http://<IP_DO_SERVIDOR>/projeto-korp
```

Resposta esperada:
```json
{"nome":"Projeto Korp","horario":"2026-09-18T23:46:12Z"}
```

### Métricas

```bash
curl http://<IP_DO_SERVIDOR>/metrics | grep http_requests_total
```

### Prometheus e Grafana

Por design, apenas a porta 80 é exposta publicamente. Prometheus (9090) e Grafana (3000) ficam acessíveis somente pela rede interna do servidor — não há como abri-los direto no navegador a partir do IP público. Para acessá-los, abra um túnel SSH antes:

```bash
ssh -i <sua-chave> -L 9090:localhost:9090 -L 3000:localhost:3000 ubuntu@<IP_DO_SERVIDOR>
```

Com o túnel ativo:

**Prometheus**: acesse `http://localhost:9090` → **Status > Targets** → target `http-server-projeto-korp` com status UP.

**Grafana**: acesse `http://localhost:3000` (login: `admin`/`admin`) → **Dashboards** → `http-server-projeto-korp`.

O dashboard contém dois painéis:
- **Disponibilidade**: Stat panel mostrando UP ou DOWN.
- **Volume de requisições (req/s por status)**: Gráfico de série temporal com `rate(http_requests_total[1m])` agrupado por status HTTP.

Para gerar tráfego e visualizar o gráfico se formando:
```bash
while true; do curl -s http://<IP_DO_SERVIDOR>/projeto-korp > /dev/null; sleep 1; done
```

---

## Decisões técnicas

- **Isolamento de rede**: O container Go não expõe portas ao host. O acesso externo passa exclusivamente pelo Nginx na porta 80. Prometheus e Grafana também não são expostos publicamente, só via túnel SSH.
- **Idempotência**: O playbook usa módulos nativos do Ansible (`ansible.builtin.*`, `community.docker.*`) e pode ser executado repetidas vezes sem efeitos colaterais.
- **PEP 668**: O Ubuntu 24.04 bloqueia `pip install` no sistema por padrão. Tratado com `extra_args: --break-system-packages` na task do pip.
- **Provisionamento do Grafana**: O datasource e o dashboard são carregados automaticamente via arquivos em `grafana/provisioning/`, sem necessidade de configuração manual na interface.
