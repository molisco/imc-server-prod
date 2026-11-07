# Walkthrough: Solução dos Dois Problemas Identificados pelas Imagens

## 1. Solução da Imagem 1: `Uncaught SyntaxError: Unexpected token '}'` & `copyToken is not defined`
- **Causa:** Havia um bloco de código órfão no script de `stores.html` (antigas linhas 766 a 779 com um `}` e `.catch()` desconectados). Quando o navegador encontrava essa instrução inválida no carregamento da página, ele abortava o parsing do script.
- **Resultado anterior:** As funções declaradas no arquivo (como `copyToken` e `openWebTerminal`) eram ignoradas pelo navegador.
- **Correção:** Removidas todas as linhas órfãs. A compilação do JavaScript da página agora ocorre com **0 ERROS de sintaxe**.

---

## 2. Solução da Imagem 2: Conexão do Container Docker da `Matriz 01 - Loja 001`
- **Causa:** O botão de terminal na tela de filiais passava o Token de Grade de Áudio (`c09e032d-c874-4366-bbd9-5e15062a1287`) em vez do **ID Único da Loja (`931f8631-cf18-43c7-80f5-5e438123f050`)**.
- **Resultado anterior:** O container Docker registrado no servidor é nomeado como `imc-radio-931f8631-cf18-43c7-80f5-5e438123f050`. Ao tentar buscar pelo Token da Grade, a busca não encontrava o contêiner e caía na listagem geral do `docker ps`.
- **Correção:** Atualizado o parâmetro enviado pela tabela para usar o **Store ID (`{{.ID}}`)**. Agora o botão chama exatamente `openWebTerminal('docker', '931f8631-cf18-43c7-80f5-5e438123f050', 'Matriz 01 - Loja 001')`, efetuando a conexão direta ao contêiner Docker da loja.

---

## Verificação Concluída
- Servidor reiniciado e validado.
- Botão da Matriz 01 - Loja 001 gera a chamada:
  `<button onclick="openWebTerminal('docker', '931f8631-cf18-43c7-80f5-5e438123f050', 'Matriz 01 - Loja 001')">`
