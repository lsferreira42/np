# Makefile para Network Pipe (NP)

# Variáveis de configuração
BINARY_NAME=np
RELAY_BINARY=np-relay
GO=go
GOFMT=gofmt
GOFLAGS=-v
BUILD_DIR=./build
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-X main.Version=$(VERSION)"

# Configurações padrão para execução
DEFAULT_PORT=4242
DEFAULT_BIND=0.0.0.0

# Cores para output
GREEN=\033[0;32m
YELLOW=\033[0;33m
NC=\033[0m # No Color

.PHONY: all build clean run fmt vet test build-relay run-relay run-receiver run-sender help

# Target padrão
all: fmt build

# Compila a aplicação principal
build:
	@echo "${GREEN}Compilando ${BINARY_NAME}...${NC}"
	@$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BINARY_NAME) .

# Executa a aplicação principal em modo interativo
run:
	@echo "${GREEN}Executando ${BINARY_NAME}...${NC}"
	@./$(BINARY_NAME)

# Compila o componente relay
build-relay:
	@echo "${GREEN}Compilando relay...${NC}"
	@cd relay && $(GO) build $(GOFLAGS) $(LDFLAGS) -o ../$(RELAY_BINARY) .

# Executa o componente relay
run-relay: build-relay
	@echo "${GREEN}Executando relay...${NC}"
	@./$(RELAY_BINARY)

# Executa a aplicação em modo receptor
run-receiver: build
	@echo "${GREEN}Executando NP em modo receptor...${NC}"
	@./$(BINARY_NAME) --receiver -p $(DEFAULT_PORT) -b $(DEFAULT_BIND)

# Executa a aplicação em modo emissor
run-sender: build
	@echo "${GREEN}Executando NP em modo emissor...${NC}"
	@./$(BINARY_NAME) --sender

# Limpa os binários gerados
clean:
	@echo "${YELLOW}Removendo binários...${NC}"
	@rm -f $(BINARY_NAME) $(RELAY_BINARY)
	@rm -rf $(BUILD_DIR)

# Formata o código
fmt:
	@echo "${GREEN}Formatando código...${NC}"
	@$(GOFMT) -w .

# Executa o verificador de erros estáticos
vet:
	@echo "${GREEN}Executando verificação estática...${NC}"
	@$(GO) vet ./...

# Executa os testes
test:
	@echo "${GREEN}Executando testes...${NC}"
	@$(GO) test -v ./...

# Compila para várias plataformas
build-all: clean
	@echo "${GREEN}Compilando para múltiplas plataformas...${NC}"
	@mkdir -p $(BUILD_DIR)
	@GOOS=linux GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 .
	@GOOS=windows GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe .
	@GOOS=darwin GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 .
	@GOOS=darwin GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 .

# Ajuda
help:
	@echo "Uso do Makefile para NP (Network Pipe):"
	@echo ""
	@echo "  make build         - Compila a aplicação principal"
	@echo "  make run           - Executa a aplicação em modo interativo"
	@echo "  make build-relay   - Compila o componente relay"
	@echo "  make run-relay     - Executa o componente relay"
	@echo "  make run-receiver  - Executa a aplicação em modo receptor"
	@echo "  make run-sender    - Executa a aplicação em modo emissor"
	@echo "  make clean         - Remove os binários gerados"
	@echo "  make fmt           - Formata o código fonte"
	@echo "  make vet           - Executa verificação estática de código"
	@echo "  make test          - Executa testes automatizados"
	@echo "  make build-all     - Compila para múltiplas plataformas"
	@echo "  make help          - Exibe esta mensagem"
	@echo ""
	@echo "Para personalizar portas e endereços, use:"
	@echo "  make run-receiver DEFAULT_PORT=5000 DEFAULT_BIND=127.0.0.1"
	@echo "  make run-sender HOST=192.168.1.100 PORT=5000" 