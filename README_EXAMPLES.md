# Exemplos de Uso do NP

Este documento contém exemplos detalhados de uso do NP (Network Pipe) para diversos cenários. Para instruções de instalação e documentação geral, consulte o [README principal](README.md).

## Uso Básico

### Exemplo Simples

```bash
# Na máquina receptora (servidor)
np --receiver

# Na máquina enviadora (cliente)
echo "Olá, mundo!" | np --sender -H 192.168.1.100
```

### Opções Avançadas

```bash
# Usando flags longas
np --sender --host 192.168.1.100 --port 5000

# Com interface web ativada
np --sender -H 192.168.1.100 --web-ui

# Usando TCP em vez de UDP
np --sender --tcp

# Usando HTTP (útil para ambientes com restrições de firewall)
np --sender --http

# Com descoberta mDNS para encontrar servidores automaticamente
np --sender --mdns

# Com suporte a múltiplas conexões
np --sender --multi

# Com compressão de dados
np --sender --compression gzip --compress-level 9
```

## Trabalhando com Logs e Streams

### Logs do Docker

Capturando logs de um container Docker e enviando para outra máquina:

```bash
# Na máquina receptora
np --receiver --tcp > logs_docker.txt

# Na máquina enviadora
docker logs -f meu_container | np --sender -H 192.168.1.100 --tcp
```

Capturando logs de múltiplos containers Docker:

```bash
# Na máquina receptora
np --receiver --tcp --multi | grep ERROR > erros_docker.log

# Na máquina enviadora (execute em cada container de interesse)
docker logs -f container1 | np --sender -H 192.168.1.100 --tcp
docker logs -f container2 | np --sender -H 192.168.1.100 --tcp
```

### Logs do Kubernetes

Monitoramento de logs de pods em um cluster Kubernetes:

```bash
# Na máquina receptora
np --receiver --tcp --compression zstd > logs_k8s.txt

# Na máquina enviadora
kubectl logs -f deployment/minha-aplicacao | np --sender -H 192.168.1.100 --tcp --compression zstd
```

Monitoramento de logs de todos os pods com determinada label:

```bash
# Na máquina receptora
np --receiver --tcp --multi > logs_por_ambiente.txt

# Na máquina enviadora
kubectl logs -f -l app=backend | np --sender -H 192.168.1.100 --tcp
kubectl logs -f -l app=frontend | np --sender -H 192.168.1.100 --tcp
```

### Logs do Systemd

Monitorando serviços systemd em tempo real:

```bash
# Na máquina receptora
np --receiver --tcp > logs_systemd.txt

# Na máquina enviadora
journalctl -fu nginx | np --sender -H 192.168.1.100 --tcp
```

Monitorando múltiplos serviços systemd:

```bash
# Na máquina receptora
np --receiver --tcp --multi | tee logs_completos.txt | grep ERROR > apenas_erros.txt

# Na máquina enviadora
journalctl -fu nginx -fu postgresql -fu redis | np --sender -H 192.168.1.100 --tcp
```

### Logs em Arquivos

Monitorando arquivos de log em tempo real:

```bash
# Na máquina receptora
np --receiver --tcp > aplicacao_logs.txt

# Na máquina enviadora
tail -f /var/log/apache2/error.log | np --sender -H 192.168.1.100 --tcp
```

Monitorando múltiplos arquivos de log:

```bash
# Na máquina receptora
np --receiver --tcp --compression zstd | tee -a todos_logs.txt

# Na máquina enviadora
tail -f /var/log/nginx/*.log | np --sender -H 192.168.1.100 --tcp --compression zstd
```

Concatenando arquivos grandes e enviando com compressão:

```bash
# Na máquina receptora
np --receiver --tcp --compression zstd > logs_concatenados.txt

# Na máquina enviadora
cat arquivo1.log arquivo2.log | np --sender -H 192.168.1.100 --tcp --compression zstd
```

## Casos de Uso Avançados

### Descoberta Automática via mDNS

Use o mDNS para encontrar automaticamente servidores NP na sua rede local:

```bash
# Na máquina receptora (servidor)
np --receiver --mdns --tcp

# Na máquina enviadora (cliente), sem precisar especificar o endereço
np --sender --mdns --tcp
```

### Transferência de Arquivos com Compressão

Envie arquivos com compressão em tempo real para melhor performance:

```bash
# Na máquina receptora
np --receiver --tcp --compression zstd > arquivo_recebido.txt

# Na máquina enviadora
cat arquivo_grande.txt | np --sender -H 192.168.1.100 --tcp --compression zstd
```

### Múltiplas Conexões Simultâneas

Aceite e gerencie várias conexões de clientes ao mesmo tempo:

```bash
# Na máquina receptora (servidor)
np --receiver --tcp --multi --web-ui | ./processar_dados.sh

# Nas máquinas clientes
cat dados1.txt | np --sender -H 192.168.1.100 --tcp
cat dados2.txt | np --sender -H 192.168.1.100 --tcp
```

### Usando HTTP para Ambientes com Firewall

Use o modo HTTP quando firewall ou proxy bloquear conexões UDP/TCP diretas:

```bash
# Na máquina receptora (servidor)
np --receiver --http

# Na máquina enviadora (cliente)
cat dados.txt | np --sender -H 192.168.1.100 --http
```

### Usando o Servidor de Relay para Atravessar NATs

Use o servidor de relay para estabelecer conexões através de NATs e firewalls:

```bash
# Na máquina receptora (atrás de NAT)
np --receiver --relay relay.apisbr.dev --session minha-sessao

# Na máquina enviadora (atrás de outro NAT)
cat dados.txt | np --sender --relay relay.apisbr.dev --session minha-sessao
```

O servidor de relay em `relay.apisbr.dev` está disponível para todos os usuários do NP.

### Combinando Todas as Funcionalidades

Para aplicações mais complexas, combine todas as funcionalidades:

```bash
# Servidor completo com todas as features
np --receiver --tcp --mdns --multi --compression zstd --web-ui

# Cliente usando descoberta automática e compressão
np --sender --tcp --mdns --compression zstd --web-ui
```

## Processamento de Dados em Tempo Real

### Pipeline de Processamento

Criando um pipeline de processamento entre máquinas:

```bash
# Na máquina receptora
np --receiver --tcp | grep "ERRO" | sort | uniq -c > relatorio_erros.txt

# Na máquina enviadora
cat logs*.txt | np --sender -H 192.168.1.100 --tcp
```

### Processamento Distribuído

Dividindo dados para processamento paralelo em várias máquinas:

```bash
# Na máquina de controle
cat dados_grandes.csv | 
  tee >(head -n 1000 | np --sender -H worker1.local --tcp) \
      >(tail -n +1001 | head -n 1000 | np --sender -H worker2.local --tcp) \
      >(tail -n +2001 | np --sender -H worker3.local --tcp) > /dev/null

# Nas máquinas de trabalho (workers)
np --receiver --tcp | ./processar_lote.sh
```

### Monitoramento de Produção em Tempo Real

Monitorando logs de produção em tempo real com alertas:

```bash
# Na máquina de monitoramento
np --receiver --tcp --multi | 
  tee >(grep -i error | mail -s "Alerta de Erro" admin@example.com) \
      logs_completos.txt

# Nas máquinas de produção
journalctl -fu aplicacao | np --sender -H monitor.local --tcp
``` 