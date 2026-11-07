# Rotação Estrita por Estilo Musical (Regra de Ouro) e Integração com Bloco Comercial

Este plano descreve a implementação da Regra de Ouro do player: garantir que cada música de um determinado estilo musical seja executada novamente **apenas após todas as outras músicas desse mesmo estilo terem sido tocadas**. O plano integra essa regra com o ajuste de duração e a prevenção de desvios (Drift Prevention) definidos anteriormente.

---

## Detalhes do Design

### 1. Particionamento por Ciclo de Rotação (Grupo "Não Tocadas")
Para evitar a necessidade de armazenar estados complexos em memória (que seriam perdidos em restarts), usaremos a tabela `play_history` e `media_library` para determinar o grupo das músicas **"Não Tocadas" no ciclo atual**:
1. Para o estilo musical ativo, contamos quantas vezes cada música baixada foi reproduzida na tabela `play_history` (limite de histórico de 48h).
2. Determinamos a quantidade mínima de reproduções (`minPlayCount`) entre todas as músicas daquele estilo.
3. O grupo de músicas candidatas (**"Não Tocadas"**) será composto estritamente pelas faixas que possuem a quantidade de reproduções igual a `minPlayCount`.
4. Assim que todas as músicas do estilo forem tocadas uma vez, o `minPlayCount` de todas passará a ser 1, reiniciando o ciclo de rotação automaticamente de forma dinâmica e auto-regenerativa.

### 2. Tratamento de Conflito com Ajuste de Duração (Duration Fitting)
Conforme escolha do usuário, a rotação estrita será mantida como prioridade máxima:
* Ao buscar uma música para caber no bloco comercial (ex: restando menos de 5 minutos):
  1. Filtramos as candidatas **apenas** dentro do grupo de músicas **"Não Tocadas"** (com quantidade de reprodução igual a `minPlayCount`).
  2. Dentro deste grupo, aplicamos o filtro de separação de artistas (últimos 45 minutos) e o ajuste de duração (mais próxima da duração necessária).
  3. Se **não houver** nenhuma música pequena disponível no grupo "Não Tocadas", selecionamos uma música longa do próprio grupo "Não Tocadas".
  4. Deixamos a regra de **Drift Prevention** do player decidir: se o atraso gerado ao tocar essa música longa for excessivo, ela será pulada e o bloco comercial entrará imediatamente, realinhando a programação e abrindo espaço de música livre no próximo bloco.

---

## Modificações Propostas

### [Componente: Agendador]

#### [MODIFY] [scheduler.go](file:///c:/imc-audio-local/scheduler.go)
* Atualizar a função [selectNextMusicTrack](file:///c:/imc-audio-local/scheduler.go#L644) para:
  * Consultar a contagem de toques de cada música no banco de dados.
  * Agrupar por estilo e determinar o `minPlayCount`.
  * Filtrar as candidatas para conter apenas aquelas onde `play_count == minPlayCount`.
  * Aplicar a separação de artistas e regras de duração estritamente sobre esse conjunto filtrado.
  * Implementar a mesma lógica de partição para o fallback de biblioteca (caso o estilo ativo não possua faixas baixadas).

---

## Plano de Verificação

### Testes de Compilação
* Compilar o player local:
  ```powershell
  go build -o imc-audio-local.exe
  ```

### Verificação de Comportamento (Manual/Script)
1. Rodar um script de teste simulando a seleção de músicas sequenciais para um estilo e verificar se nenhuma música se repete até que todas tenham sido selecionadas.
2. Monitorar os logs de rotação no console.
