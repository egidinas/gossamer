import { Database, FlaskConical, GitBranch, Linkedin, Mail, ShieldCheck, Wrench } from "lucide-react";
import { OperatorPanel } from "../components/OperatorPanel";

const focusAreas = [
  {
    icon: FlaskConical,
    title: "Environmental test as a production function",
    text: "TVac, thermal, vibration, shock, vacuum, cryogenic supply, chamber readiness, fixtures, procedures, and campaign evidence treated as one operating system."
  },
  {
    icon: Database,
    title: "Unified test data and evidence",
    text: "Facility state, DUT telemetry, legacy logs, live sources, command counters, anomalies, and requirements brought into shared contracts that can be inspected and reported."
  },
  {
    icon: Wrench,
    title: "Infrastructure and instrumentation",
    text: "Hands-on work across thermal chambers, LN2 and vacuum infrastructure, dual-zone thermal control, sensor chains, automation, data acquisition, and supplier interfaces."
  },
  {
    icon: GitBranch,
    title: "Technical leadership",
    text: "Building repeatable test capability from early feasibility through derisking, design-for-test feedback, formal qualification, SOPs, team growth, and stakeholder communication."
  }
];

export function ProfileView() {
  return (
    <div className="profile-grid">
      <section className="profile-hero">
        <span className="eyebrow">technical context</span>
        <h1>Dr. Jonathan Meyer</h1>
        <p>
          I build environmental-test capability for space hardware: the facilities, instrumentation,
          procedures, data visibility, and evidence flow that turn demanding campaigns into a repeatable
          engineering and production function.
        </p>
        <p>
          My background combines six years leading environmental testing and qualification during
          startup-to-serial scale-up with a PhD foundation in cryogenic and vacuum instrumentation.
          The common thread is practical technical ownership across hardware, test infrastructure,
          data systems, and the evidence needed to make decisions.
        </p>
        <div className="profile-contact">
          <a href="mailto:jonathan@jmeyer.space"><Mail size={16} /> jonathan@jmeyer.space</a>
          <a href="https://www.linkedin.com/in/dr-jonathan-meyer-8650a986/"><Linkedin size={16} /> LinkedIn</a>
          <a href="#landing"><ShieldCheck size={16} /> Gossamer demonstrator</a>
        </div>
      </section>

      <OperatorPanel title="Technical Activities And Interests" meta="field notes">
        <div className="value-grid">
          {focusAreas.map((area) => {
            const Icon = area.icon;
            return (
              <div key={area.title}>
                <Icon size={18} />
                <strong>{area.title}</strong>
                <span>{area.text}</span>
              </div>
            );
          })}
        </div>
      </OperatorPanel>

      <OperatorPanel title="Representative Scope" meta="evidence-backed summary">
        <div className="metric-grid profile-scope-grid">
          <div>
            <span className="label">Qualification domains</span>
            <strong>TVac, thermal, vibration, shock</strong>
          </div>
          <div>
            <span className="label">Facility scope</span>
            <strong>LN2, vacuum, chambers, EGSE</strong>
          </div>
          <div>
            <span className="label">Scale-up work</span>
            <strong>parallel chambers and repeatable campaigns</strong>
          </div>
          <div>
            <span className="label">Data systems</span>
            <strong>live visibility and measurement-to-proof traceability</strong>
          </div>
        </div>
      </OperatorPanel>

      <OperatorPanel title="Connection To Gossamer" meta="environmental-test data systems">
        <div className="profile-note">
          <p>
            Gossamer explores ideas from that work: source-owned telemetry,
            tile-backed graph contracts, functional-test markers, requirement progress, and evidence links that
            make an environmental-test campaign visible before it becomes a report.
          </p>
          <p>
            Modern and legacy sources can be translated into a shared pool of current and historical data for
            engineers, operators, and stakeholders.
          </p>
        </div>
      </OperatorPanel>
    </div>
  );
}
