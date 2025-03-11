# NP (Network Pipe)

*[English version available here](README.en.md)*

NP é uma ferramenta de linha de comando para criar pipes de rede bidirecionais entre máquinas. Ela funciona como uma alternativa moderna e intuitiva ao netcat, com suporte integrado para detecção de serviços e um protocolo simples de autenticação.

## O que é o NP?

O NP (Network Pipe) permite que você conecte facilmente a entrada e saída padrão (stdin/stdout) de dois processos em máquinas diferentes através da rede usando o protocolo UDP ou TCP. Esta ferramenta é ideal para:

- Transferência rápida de arquivos entre máquinas
- Comunicação em tempo real (como um chat simples)
- Piping de dados entre comandos em máquinas diferentes
- Debugging de conexões de rede
- Streaming de logs ou saídas de comandos remotos

## Características

- **Comunicação Bidirecional**: Suporte a UDP e TCP
- **Interface Intuitiva**: Modo interativo para configuração fácil
- **Segurança Básica**: Verificação de autenticação entre peers
- **Integração com Shell**: Integração perfeita com stdin/stdout para uso com pipes Unix
- **Dois Modos**: Receptor (servidor) e emissor (cliente)
- **Monitoramento**: Interface web para visualização em tempo real
- **Descoberta Automática**: Localização de serviços via mDNS (Bonjour/Avahi)
- **Múltiplas Conexões**: Suporte a modo multiplex para várias conexões simultâneas
- **Compressão**: Algoritmos de compressão em tempo real (gzip, zlib, zstd)
- **Portabilidade**: Código leve escrito em Go, compatível com múltiplas plataformas

## Instalação

### Via Go Install
```bash
go install github.com/lsferreira42/np@latest
```

### Compilando do Fonte
```bash
git clone https://github.com/lsferreira42/np.git
cd np
go build
```

### Binários Pré-compilados

Você pode encontrar binários pré-compilados para várias plataformas na [página de releases](https://github.com/lsferreira42/np/releases).

## Uso

NP opera em dois modos principais:

```bash
# Modo receptor (servidor): escuta por conexões de entrada
np --receiver

# Modo emissor (cliente): conecta-se a um receptor
np --sender -H 192.168.1.100
```

Para uma lista completa de exemplos detalhados, incluindo cenários específicos com logs do Docker, Kubernetes, systemd e arquivos de log, consulte o [Guia de Exemplos](README_EXAMPLES.md).

## Interface Web

O NP possui uma interface web integrada que permite monitorar conexões, visualizar tráfego e acessar estatísticas em tempo real.

### Ativando a Interface Web

```bash
# Ativar com as configurações padrão (porta 8080)
np --receiver --web-ui

# Especificar porta personalizada para a interface web
np --receiver --web-ui --web-port 9000
```

A interface é acessível através de qualquer navegador web moderno e atualiza os dados em tempo real.

## Opções

### Opções Globais
- `-p, --port`: Porta para conexão (padrão: 4242)
- `--web-ui`: Ativa a interface web de monitoramento
- `--web-port`: Porta para a interface web (padrão: 8080)
- `--web-bind`: Endereço para bind da interface web (padrão: 0.0.0.0)
- `--tcp`: Usa TCP em vez de UDP para comunicação
- `--http`: Usa HTTP para comunicação (útil para ambientes com restrições de firewall)
- `--mdns`: Ativa a descoberta/anúncio via mDNS
- `--multi`: Ativa o suporte a múltiplas conexões simultâneas
- `--compression`: Algoritmo de compressão (none, gzip, zlib, zstd)
- `--compress-level`: Nível de compressão (1-9, padrão: 6)
- `--relay`: Endereço do servidor de relay (padrão: relay.apisbr.dev)
- `--session`: ID da sessão para conexão via relay

### Opções do Receptor
- `-b, --bind`: Endereço para bind (padrão: 0.0.0.0)

### Opções do Emissor
- `-H, --host`: Host para conectar (padrão: 127.0.0.1)

## Protocolo

O NP utiliza um protocolo simples para autenticação:

1. Cliente envia "ISNP"
2. Servidor responde com "OK" se for uma instância válida do NP
3. Comunicação normal pode começar após esta autenticação

Este protocolo garante que o NP só se comunique com outras instâncias do NP, evitando confusão com outros serviços de rede.

## Limitações Atuais

- Utiliza apenas UDP (sem garantia de entrega para grandes volumes de dados)
- Sem criptografia nativa (use SSH tunneling para comunicações seguras)
- Tamanho do buffer limitado a 4096 bytes por pacote

## Recursos Futuros

- [x] Suporte a TCP para garantia de entrega
- [x] Suporte a HTTP para ambientes com restrições de firewall
- [x] Descoberta automática via mDNS
- [ ] Criptografia end-to-end
- [x] Modo relay para NAT traversal
- [x] Interface web para monitoramento
- [x] Modo multiplex para várias conexões simultâneas
- [x] Compressão de dados em tempo real
- [x] Suporte a múltiplos algoritmos de compressão (gzip, zlib, zstd)
- [x] Níveis de compressão configuráveis

## Resolução de Problemas

### Porta em uso
Se o NP mostrar um erro indicando que a porta já está em uso, tente:
1. Verificar se outra instância do NP está rodando
2. Verificar se outro aplicativo está usando a porta
3. Escolher uma porta diferente com o parâmetro `-p`

### Problemas de Conexão
Se o emissor não conseguir se conectar ao receptor:
1. Verifique se o receptor está rodando
2. Verifique se não há firewalls bloqueando a conexão
3. Tente usar a opção `-b 0.0.0.0` no receptor para escutar em todos os interfaces

### Interface Web Inacessível
Se a interface web não estiver acessível:
1. Verifique se você especificou a opção `--web-ui`
2. Certifique-se de que a porta da interface web não esteja bloqueada por firewall
3. Verifique se o endereço de binding permite acesso de outras máquinas
4. Use `--web-bind 0.0.0.0` para permitir acesso de qualquer endereço

## Contribuindo

Contribuições são bem-vindas! Para contribuir:

1. Faça um fork do repositório
2. Crie uma branch para sua feature (`git checkout -b feature/nova-feature`)
3. Commit suas mudanças (`git commit -am 'Adiciona nova feature'`)
4. Push para a branch (`git push origin feature/nova-feature`)
5. Crie um novo Pull Request

## Licença

Este projeto está licenciado sob a MIT License - veja o arquivo [LICENSE](LICENSE) para detalhes.

## Autor

Leandro Ferreira (@lsferreira42)

## Agradecimentos

- Inspirado pelo netcat e outras ferramentas de rede clássicas
- Agradecimentos especiais à comunidade Go por feedback e contribuições