import type { FindingView } from "../../backend/backend";

type HeatmapFinding = Pick<FindingView, "dueState" | "severity" | "status">;

interface RiskHeatmapProps {
  className?: string;
  records: readonly HeatmapFinding[];
  title?: string;
}

function coordinatesFor(record: HeatmapFinding): { impact: number; likelihood: number } {
  const overdue = record.status !== "CLOSED" && record.dueState === "OVERDUE";
  if (record.severity === "LEVEL_1_CRITICAL") return { likelihood: overdue ? 5 : 4, impact: 5 };
  if (record.severity === "LEVEL_2_MAJOR") return { likelihood: overdue ? 4 : 3, impact: 4 };
  if (record.severity === "LEVEL_3_MINOR") return { likelihood: overdue ? 3 : 2, impact: 3 };
  return { likelihood: record.status === "CLOSED" ? 1 : 2, impact: 2 };
}

function toneFor(score: number): "is-critical" | "is-high" | "is-low" | "is-medium" {
  if (score >= 15) return "is-critical";
  if (score >= 10) return "is-high";
  if (score >= 5) return "is-medium";
  return "is-low";
}

export function RiskHeatmap({
  className,
  records,
  title = "Risk Exposure Matrix",
}: RiskHeatmapProps) {
  const positioned = records.map(coordinatesFor);
  const cells = Array.from({ length: 25 }, (_, index) => {
    const likelihood = 5 - Math.floor(index / 5);
    const impact = index % 5 + 1;
    return {
      count: positioned.filter((record) => record.likelihood === likelihood && record.impact === impact).length,
      impact,
      likelihood,
      score: likelihood * impact,
    };
  });

  return (
    <section
      aria-label={title}
      className={["risk-heatmap", className].filter(Boolean).join(" ")}
      data-testid="risk-exposure-matrix"
    >
      <header className="risk-heatmap__header">
        <div>
          <span>Likelihood × Impact</span>
          <h2>{title}</h2>
        </div>
      </header>
      <div className="risk-heatmap__layout">
        <span className="risk-heatmap__axis risk-heatmap__axis--y">Higher likelihood</span>
        <div className="risk-heatmap__grid">
          {cells.map((cell) => (
            <span
              aria-label={`Likelihood ${cell.likelihood}, impact ${cell.impact}: ${cell.count} Finding records`}
              className={`risk-heatmap__cell ${toneFor(cell.score)}`}
              data-matrix-cell
              key={`${cell.likelihood}-${cell.impact}`}
              title={`Likelihood ${cell.likelihood}, Impact ${cell.impact}`}
            >
              <b>{cell.count}</b>
              <small>{cell.score}</small>
            </span>
          ))}
        </div>
        <span className="risk-heatmap__axis risk-heatmap__axis--x">Impact →</span>
      </div>
      <p className="risk-heatmap__basis">
        Finding placement follows the accepted demo severity and Due State mapping. The matrix is
        advisory management information only.
      </p>
    </section>
  );
}
