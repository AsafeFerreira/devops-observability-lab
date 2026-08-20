# Pressão de CPU, memória ou disco

Alertas: `LabApplicationCPUHigh`, `LabApplicationMemoryHigh`, `LabKubernetesWorkloadCPUHigh`, `LabKubernetesWorkloadMemoryHigh`, `LabKubernetesNodeFilesystemLow`.

## Triagem

1. Identifique processo/pod/nó e se o consumo está crescendo ou apenas teve pico.
2. Compare uso com requests/limits, reinícios, throttling, OOM e tráfego.
3. Para disco, identifique filesystem e consumidor; não apague arquivos por suposição.
4. Correlacione com latência, erros, backlog e deploy recente.

## Contenção

- Reduza carga artificial no laboratório.
- Preserve logs/evidências antes de reiniciar.
- Não faça `rm`, não reduza retenção e não aumente limites em produção sem autorização e avaliação de impacto.
- Se houver risco de OOM/disk full, avise a equipe responsável imediatamente.

## Recuperação

Corrija a causa: vazamento, consulta, retry storm, retenção, request/limit inadequado ou capacidade. Uma alteração de capacidade precisa de acompanhamento para confirmar que não mascara regressão.

## Validar

Uso abaixo do limiar por dez minutos, sem reinícios, erros ou backlog crescente. Registre gráfico antes/depois e ação tomada.
