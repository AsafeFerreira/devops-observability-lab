#!/usr/bin/env sh
# Imprime o ultimo ciclo firing -> resolved registrado pelo alert recorder.
# Uso: sh scripts/alert-cycle.sh [alertname] [instance]
set -eu

alerta="${1:-LabServiceDown}"
instancia="${2:-imports-worker:8081}"

sh "$(dirname "$0")/simulate-failure.sh" evidence 2>/dev/null | ALERTA="$alerta" INSTANCIA="$instancia" python3 -c '
import sys, json, os

alerta = os.environ["ALERTA"]
instancia = os.environ["INSTANCIA"]

ciclos = []
for linha in sys.stdin:
    if not linha.strip():
        continue
    evento = json.loads(linha)
    for a in evento["alerts"]:
        rotulos = a["labels"]
        if rotulos["alertname"] == alerta and rotulos.get("instance") == instancia:
            ciclos.append(a)

if not ciclos:
    print("Nenhum ciclo registrado para %s em %s." % (alerta, instancia))
    raise SystemExit(0)

def hora(ts):
    return ts[11:19] + "Z"

print()
print("  Ciclo firing -> resolved  |  %s / %s" % (alerta, instancia.split(":")[0]))
print("  " + "=" * 66)
print()

for a in ciclos[-2:]:
    aberto = a["endsAt"].startswith("0001")
    print("  STATUS      %s" % a["status"].upper())
    print("  severidade  %s" % a["labels"]["severity"])
    print("  startsAt    %s" % a["startsAt"])
    print("  endsAt      %s" % ("(em aberto)" if aberto else a["endsAt"]))
    print()

ultimo = ciclos[-1]
if not ultimo["endsAt"].startswith("0001"):
    from datetime import datetime
    fmt = "%Y-%m-%dT%H:%M:%S.%fZ"
    inicio = datetime.strptime(ultimo["startsAt"], fmt)
    fim = datetime.strptime(ultimo["endsAt"], fmt)
    total = int((fim - inicio).total_seconds())
    print("  " + "-" * 66)
    print("  Deteccao     %s   alerta disparado pelo Prometheus" % hora(ultimo["startsAt"]))
    print("  Recuperacao  %s   servico restabelecido, alerta encerrado sozinho" % hora(ultimo["endsAt"]))
    print("  Duracao      %dm%02ds       ciclo completo registrado pelo alert recorder" % (total // 60, total % 60))
    print()

print("  runbook: %s" % ultimo["annotations"]["runbook_url"].split("/main/")[-1])
print()
'
