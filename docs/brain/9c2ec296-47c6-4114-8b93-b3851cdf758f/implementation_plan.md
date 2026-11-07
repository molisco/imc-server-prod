# Plano de Implementação: Restrição de Blocos por Porcentagem de Ofertas e Institucionais

Este plano descreve como corrigir a distribuição antecipada de comerciais para que ela respeite as porcentagens de blocos comerciais configuradas no servidor para **Ofertas** e **Institucionais** (ex: tocar ofertas em apenas 50% dos blocos do dia).

## Diagnóstico do Problema

Atualmente:
1. O servidor e o banco de dados do player local sincronizam corretamente as porcentagens de `PercentageOffer` e `PercentageInstitutional` (conforme verificado nos logs e no banco local, ambos atualizados para 50% e 50% na última sincronia).
2. Porém, o algoritmo de distribuição antecipada (`PreDistributeCommercials` em [scheduler.go](file:///c:/imc-audio-local/scheduler.go)) considera **todos os blocos do dia** como elegíveis para qualquer tipo de comercial:
   ```go
   eligibleBlocks := blockTimes
   ```
3. Como resultado, as ofertas e os comerciais institucionais acabavam sendo distribuídos em blocos aleatórios ao longo do dia inteiro, ignorando a restrição de porcentagem definida na grade da loja.

## Alterações Propostas

### Local Player Scheduler

#### [MODIFY] [scheduler.go](file:///c:/imc-audio-local/scheduler.go)
* Atualizar a função `PreDistributeCommercials` para filtrar a lista de `eligibleBlocks` com base na regra de porcentagem de blocos:
  * Se o comercial for do tipo **Oferta** (`OFFER`), ele só poderá ser planejado em blocos cujo índice satisfaça `shouldPlayForPercentage(blockIdx, blockConfig.PercentageOffer)`.
  * Se o comercial for do tipo **Institucional** (`INSTITUTIONAL`), ele só poderá ser planejado em blocos cujo índice satisfaça `shouldPlayForPercentage(blockIdx, blockConfig.PercentageInstitutional)`.
  * Demais comerciais (como comerciais gerais) continuam sendo elegíveis para todos os blocos.

## Plano de Verificação

### Testes de Compilação
* Validar a compilação do player local: `go build -o imc-audio-local.exe` no diretório `c:\imc-audio-local`

### Verificação Manual
1. Alterar a porcentagem de **Ofertas** ou **Institucionais** na grade do servidor para um valor diferente (ex: `0%` para desativar, ou `50%` para restringir a metade dos blocos).
2. Clicar no botão "Sincronizar" no dashboard do Player Local.
3. Verificar na aba **Distribuição de Comerciais (Hoje)** se as mídias daquele tipo agora aparecem distribuídas apenas nos blocos correspondentes à porcentagem, ou se deixaram de tocar completamente caso a porcentagem seja `0%`.
