# Veritas: AI Enhanced Online Assessment and Monitoring System

This repository contains two LaTeX projects for the Veritas final-year project:

- **docs/** — Full written report (chapters, tables, figures) compiled via LaTeX.
- **veritas_presentation/** — Beamer presentation with modular section files and assets.

## Build Instructions

### Report (docs)
```bash
cd docs
latexmk -pdf -interaction=nonstopmode report.tex
```
Outputs are written to `docs/tex_out/` (ignored by git).

### Presentation (veritas_presentation)
```bash
cd veritas_presentation
latexmk -pdf -interaction=nonstopmode presentation.tex
```
Outputs are written to `veritas_presentation/tex_out/` (ignored by git).

## Structure
- docs/chapters/ — Report chapters and tables
- docs/assets/ — Diagrams and figures
- veritas_presentation/sections/ — Slide content per section
- veritas_presentation/assets/ — Diagrams and images

## Notes
- Keep generated files out of version control; common LaTeX artifacts are already ignored.
- Use `latexmk` for reliable incremental builds; it handles reruns automatically.
