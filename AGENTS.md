# AI Agent Guidelines for Hybrid Logistics Engine

This document provides essential instructions for AI agents (and human developers) working on the Hybrid Logistics Engine codebase. Compliance with these guidelines ensures that documentation remains synchronized with architectural and code changes.

## 1. Documentation-First Mentality

Whenever you modify the codebase, you **must** assess the impact on the existing documentation. 

### Docusaurus Docs
If your changes affect any of the following, update the corresponding files in `docs-site/docs/`:
- **API endpoints or request/response structures**: Update `docs/api/`.
- **Infrastructure or Deployment**: Update `docs/deployment/`.
- **Core Logic or Architecture**: Update `docs/architecture/`.
- **Database Schema**: Update `docs/database/`.

## 2. Maintaining System Transparency

- **Task Tracking**: Always maintain a `task.md` in the `.brain` or similar directory during active development.
- **Walkthroughs**: After completing a feature or a significant refactor, generate a `walkthrough.md` that visually demonstrates the changes (using screenshots or Mermaid diagrams).
- **Architecture Updates**: If changing service-to-service communication, update the Mermaid diagrams in `docs/architecture/hld.md` or related flow files.

## 3. Communication Standards

- **Low-Level Design (LLD)**: Use LLD documents to detail complex implementations before or after finishing. Refer to `trip_service_lld.md` as a template.
- **Cleanup**: Do not leave "todo" comments or debugging logs in production-ready branches. 

## 4. Verification

Before finishing a task:
- Run `npm run build` in `docs-site` to ensure no broken links were introduced.
- Verify that your changes appear correctly in the sidebar.
- Ensure all Mermaid diagrams render as expected.

---
*Following these rules ensures that Hybrid Logistics Engine remains a well-documented, state-of-the-art logistics platform.*
