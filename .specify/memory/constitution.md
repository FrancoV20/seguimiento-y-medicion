<!--
Sync Impact Report
- Version change: 1.0.0 → 1.1.0
- Modified principles: Technology & Quality Constraints expanded with mandatory PostgreSQL persistence
- Added sections: VI. PostgreSQL Persistence
- Removed sections: none
- Reason: establish PostgreSQL as the mandatory persistence technology and require concrete
	implementations for every storage interface before system integration.
- Deferred items: none.
-->

# Seguimiento y Medición Constitution

## Core Principles

### I. Value-Driven Product Scope
The project MUST deliver end-to-end software project management capabilities: backlog, sprint planning, effort tracking, defect management, estimation, and measurement dashboards. Every feature MUST map to a user or team need and MUST be traceable to a clear project outcome, not to isolated implementation preferences.

Rationale: The project exists to support software delivery decisions with evidence. Scope creep and undocumented features reduce trust in the estimates and measures produced by the system.

### II. Quality by Construction
The application MUST prioritize maintainability, readability, modularity, and explicit business rules. Go code MUST be organized around bounded responsibilities, clear interfaces, and predictable data flow; shared logic MUST be extracted when reused across modules instead of duplicated.

Rationale: This project combines planning, metrics, and operational workflows. Quality is not a later review step; it is the default condition under which business logic remains understandable and testable.

### III. Test-First and Evidence-Driven Delivery
All user-visible behavior MUST be specified before implementation, and the team MUST follow a RED → GREEN → REFACTOR cycle. Acceptance scenarios MUST be expressed with Given–When–Then language, and tests MUST verify behavior before code is considered complete.

Rationale: The project explicitly adopts SDD, BDD, and TDD as governing practices. Without executable evidence, estimation and tracking features cannot be trusted.

### IV. Agile, Collaborative Execution
The team MUST operate in short, inspectable delivery cycles aligned to Scrum practices, with sprint goals, backlog prioritization, and explicit ownership of responsibilities. Work MUST be visible in progress, reviewed before completion, and adjusted using team feedback rather than individual assumptions.

Rationale: The product spans planning, execution, and measurement. Regular inspection and shared accountability are required to keep the team aligned on project value and delivery risk.

### V. Measurement and Continuous Improvement
The system MUST expose metrics that support decision making: velocity, effort variance, defect trends, sprint progress, and project health indicators. The team MUST review these metrics during sprint execution and use them to improve estimation quality, team flow, and delivery predictability.

Rationale: The project is not only about managing software work; it is also about learning from that work and improving the team's engineering performance over time.

### VI. PostgreSQL Persistence
La persistencia del sistema MUST implementarse con PostgreSQL. Toda interfaz de almacenamiento
definida en una feature, incluyendo `ProjectStore`, MUST contar con una implementación concreta
contra PostgreSQL, con sus pruebas de integración correspondientes, antes de integrarse al sistema.

Rationale: Centralizar la persistencia en PostgreSQL evita divergencias entre implementaciones,
garantiza que las reglas de almacenamiento se verifiquen contra el motor real y establece un
criterio común para integrar nuevas features.

## Technology & Quality Constraints

The project MUST be implemented primarily in Go, with the business rules and planning logic treated as first-class, testable domain behavior. The architecture MUST support backlog management, sprint execution, estimation workflows, defect tracking, and reporting without mixing unrelated concerns into a single layer.

All system persistence MUST use PostgreSQL. Storage interfaces introduced by features MUST NOT
be integrated as abstractions alone; each MUST be accompanied by a concrete PostgreSQL adapter
and integration tests against PostgreSQL before the feature is considered integrated.

The team MUST follow Scrum as the operating model and use specification-driven and behavior-driven development as the default path from requirement to code. AI tools MAY support analysis, code generation, documentation, and review only when they improve clarity, correctness, and traceability without replacing product judgment or test evidence.

The team MUST preserve explicit acceptance criteria, story relationships, and measurable outcomes in code, tests, and documentation. Any change to technology, process, or delivery standards MUST be documented as a governance update before it becomes the new normal.

## Development Workflow & Quality Gates

1. Every feature or bug fix MUST begin with a clear requirement, acceptance criteria, and user-visible behavior statement.
2. Tests MUST be written or updated before implementation for the affected behavior, and a failing state MUST be observed before the fix is considered complete.
3. Implementation MUST proceed in small, reviewable increments with frequent validation against the relevant tests.
4. Pull requests and merge decisions MUST verify conformance with the constitution, including scope discipline, test coverage, and requirement traceability.
5. Sprint review and retrospective outcomes MUST be used to refine backlog priorities, estimation practices, and quality standards.

## Governance

This constitution supersedes informal process assumptions and acts as the governing standard for product, engineering, and delivery decisions in this project. Compliance is mandatory for all work performed under this repository and any related project artifacts.

Amendments MUST be recorded in the project constitution, include a clear rationale for the change, and be reviewed by the team before adoption. Versioning MUST follow semantic versioning: MAJOR for breaking or incompatible governance changes, MINOR for new principles or materially expanded guidance, and PATCH for clarifications and non-semantic refinements. The team MUST review compliance at sprint boundaries and treat unresolved deviations as corrective action items.

**Version**: 1.1.0 | **Ratified**: 2026-09-29 | **Last Amended**: 2026-10-01
