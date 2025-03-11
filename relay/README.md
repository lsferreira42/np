# NP Relay Server

Este é o servidor de relay para o NP (Network Pipe), que permite a comunicação entre clientes NP através de NATs e firewalls.

O servidor de relay atua como um intermediário entre dois clientes NP, permitindo que eles se comuniquem mesmo quando não é possível estabelecer uma conexão direta devido a restrições de rede.

## Características

- Suporte a conexões TCP na porta 42421 (padrão)
- Suporte a conexões HTTP na porta 80
- Suporte a conexões HTTPS na porta 443 (quando configurado com certificados TLS)
- Gerenciamento automático de sessões
- Limpeza automática de sessões inativas
- Interface web simples para status do servidor

## Implantação

### Usando Docker

A maneira mais fácil de executar o servidor de relay é usando Docker e Docker Compose:

```bash
# Clone o repositório
git clone https://github.com/lsferreira42/np.git
cd np/relay

# Inicie o servidor usando Docker Compose
docker-compose up -d
```

Isso iniciará o servidor de relay com as configurações padrão, escutando nas portas 42421 (TCP), 80 (HTTP) e 443 (HTTPS, se configurado).

### Configurando HTTPS

Para habilitar o HTTPS, você precisa fornecer certificados TLS. Edite o arquivo `docker-compose.yml` e descomente as linhas relacionadas aos certificados:

```yaml
command:
  - "--tcp-port=42421"
  - "--http-port=80"
  - "--https-port=443"
  - "--tcp=true"
  - "--http=true"
  - "--https=true"  # Alterado para true
  - "--tls-cert=/certs/fullchain.pem"  # Descomentado
  - "--tls-key=/certs/privkey.pem"     # Descomentado
volumes:
  - /path/to/certs:/certs  # Descomentado e ajustado para o caminho dos seus certificados
```

### Compilando e Executando Manualmente

Se preferir compilar e executar o servidor manualmente:

```bash
# Compile o servidor
cd relay
go build -o relay-server

# Execute o servidor
./relay-server --tcp-port=42421 --http-port=80 --https-port=443
```

## Opções de Configuração

O servidor de relay aceita as seguintes opções de linha de comando:

- `--tcp-port`: Porta TCP para escutar (padrão: 42421)
- `--http-port`: Porta HTTP para escutar (padrão: 80)
- `--https-port`: Porta HTTPS para escutar (padrão: 443)
- `--tcp`: Habilita o servidor TCP (padrão: true)
- `--http`: Habilita o servidor HTTP (padrão: true)
- `--https`: Habilita o servidor HTTPS (padrão: false)
- `--tls-cert`: Caminho para o arquivo de certificado TLS
- `--tls-key`: Caminho para o arquivo de chave TLS
- `--debug`: Habilita o modo de depuração (padrão: false)
- `--max-connections`: Número máximo de conexões simultâneas (padrão: 1000)
- `--idle-timeout`: Tempo limite para sessões inativas (padrão: 30m)

## Uso com o NP

Para usar o servidor de relay com o NP, você precisa configurar o NP para usar o relay:

```bash
# No cliente 1 (iniciador)
np --sender --relay relay.apisbr.dev --session minha-sessao

# No cliente 2 (receptor)
np --receiver --relay relay.apisbr.dev --session minha-sessao
```

O servidor de relay hospedado em `relay.apisbr.dev` estará disponível por padrão para todos os usuários do NP, facilitando a comunicação através de NATs e firewalls.

## Monitoramento

O servidor de relay fornece uma página de status simples acessível via HTTP:

```
http://relay.apisbr.dev/
```

Esta página mostra informações básicas sobre o servidor, incluindo o número de sessões ativas.

## Segurança

O servidor de relay não inspeciona ou modifica os dados transmitidos entre os clientes. No entanto, para comunicações sensíveis, recomenda-se:

1. Usar HTTPS para conexões ao servidor de relay
2. Usar IDs de sessão complexos e difíceis de adivinhar
3. Considerar a criptografia end-to-end dos dados antes de enviá-los através do relay

## Licença

Este projeto é licenciado sob a MIT License - veja o arquivo [LICENSE](../LICENSE) para detalhes. 