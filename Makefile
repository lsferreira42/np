# Makefile para Network Pipe (NP)

# Variáveis de configuração
BINARY_NAME=np
RELAY_BINARY=np-relay
GO=go
GOFMT=gofmt
GOFLAGS=-v
BUILD_DIR=./build
RELEASE_DIR=./release
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-X main.Version=$(VERSION)"

# Configurações padrão para execução
DEFAULT_PORT=4242
DEFAULT_BIND=0.0.0.0

# Cores para output
GREEN=\033[0;32m
YELLOW=\033[0;33m
BLUE=\033[0;34m
NC=\033[0m # No Color

.PHONY: all build clean run fmt vet test build-relay run-relay run-receiver run-sender help build-all release

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
	@rm -rf $(RELEASE_DIR)

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
	
	@# Linux (amd64, 386, arm64, arm)
	@echo "${BLUE}Compilando para Linux...${NC}"
	@GOOS=linux GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 .
	@GOOS=linux GOARCH=386 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-386 .
	@GOOS=linux GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 .
	@GOOS=linux GOARCH=arm $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm .
	
	@# Windows (amd64, 386, arm64)
	@echo "${BLUE}Compilando para Windows...${NC}"
	@GOOS=windows GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe .
	@GOOS=windows GOARCH=386 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-386.exe .
	@GOOS=windows GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-arm64.exe .
	
	@# macOS (amd64, arm64)
	@echo "${BLUE}Compilando para macOS...${NC}"
	@GOOS=darwin GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 .
	@GOOS=darwin GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 .
	
	@# FreeBSD (amd64, 386, arm64, arm)
	@echo "${BLUE}Compilando para FreeBSD...${NC}"
	@GOOS=freebsd GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-freebsd-amd64 .
	@GOOS=freebsd GOARCH=386 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-freebsd-386 .
	@GOOS=freebsd GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-freebsd-arm64 .
	@GOOS=freebsd GOARCH=arm $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-freebsd-arm .
	
	@# OpenBSD (amd64, 386, arm64, arm)
	@echo "${BLUE}Compilando para OpenBSD...${NC}"
	@GOOS=openbsd GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-openbsd-amd64 .
	@GOOS=openbsd GOARCH=386 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-openbsd-386 .
	@GOOS=openbsd GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-openbsd-arm64 .
	@GOOS=openbsd GOARCH=arm $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-openbsd-arm .
	
	@# NetBSD (amd64, 386, arm64, arm)
	@echo "${BLUE}Compilando para NetBSD...${NC}"
	@GOOS=netbsd GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-netbsd-amd64 .
	@GOOS=netbsd GOARCH=386 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-netbsd-386 .
	@GOOS=netbsd GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-netbsd-arm64 .
	@GOOS=netbsd GOARCH=arm $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-netbsd-arm .

# Cria release com GitHub CLI
release: build-all
	@echo "${GREEN}Criando release ${VERSION}...${NC}"
	@mkdir -p $(RELEASE_DIR)
	
	@# Compactar os binários em arquivos tar.gz e zip para Windows
	@echo "${BLUE}Compactando binários...${NC}"
	@for file in $(BUILD_DIR)/*; do \
		if [[ $$file == *.exe ]]; then \
			basefile=$$(basename $$file .exe); \
			echo "Criando zip para $$basefile.exe"; \
			(cd $(BUILD_DIR) && mkdir -p ../$(RELEASE_DIR) && zip -q ../$(RELEASE_DIR)/$$basefile.zip $$(basename $$file)); \
		else \
			basefile=$$(basename $$file); \
			echo "Criando tar.gz para $$basefile"; \
			mkdir -p $(RELEASE_DIR); \
			tar -czf $(RELEASE_DIR)/$$basefile.tar.gz -C $(BUILD_DIR) $$basefile; \
		fi; \
	done
	
	@# Verificar se o GitHub CLI está instalado
	@command -v gh >/dev/null 2>&1 || { echo "${YELLOW}GitHub CLI não encontrado. Instale-o para criar releases: https://cli.github.com/${NC}"; exit 1; }
	
	@# Criar release no GitHub
	@echo "${BLUE}Criando release no GitHub com tag ${VERSION}...${NC}"
	@gh release create $(VERSION) --title "Release $(VERSION)" --notes "Release automática criada em $(shell date)"
	
	@# Fazer upload dos arquivos
	@echo "${BLUE}Enviando arquivos para o GitHub...${NC}"
	@for file in $(RELEASE_DIR)/*; do \
		echo "Enviando $$(basename $$file)"; \
		gh release upload $(VERSION) $$file; \
	done
	
	@echo "${GREEN}Release $(VERSION) criada com sucesso!${NC}"

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
	@echo "  make release       - Cria uma release no GitHub com todos os binários"
	@echo "  make help          - Exibe esta mensagem"
	@echo ""
	@echo "Para personalizar portas e endereços, use:"
	@echo "  make run-receiver DEFAULT_PORT=5000 DEFAULT_BIND=127.0.0.1"
	@echo "  make run-sender HOST=192.168.1.100 PORT=5000" 