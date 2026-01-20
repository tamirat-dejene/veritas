# Veritas: AI Enhanced Online Assessment and Monitoring System
## Comprehensive Presentation Documentation

This directory contains a detailed, modular LaTeX Beamer presentation covering all aspects of the Veritas project.

## Structure

### Main File
- `presentation.tex` - Main presentation file that imports all sections

### Section Files (in `sections/` directory)
1. **01_introduction.tex** - Introduction, Background, Problem Statement, Objectives, Scope, Methodology
2. **02_literature_review.tex** - Evolution of online assessment, related works, gaps identified, lessons learned
3. **03_problem_analysis.tex** - Existing systems overview and identified problems
4. **04_requirements.tex** - Comprehensive functional and non-functional requirements
5. **05_system_modeling.tex** - Use cases, sequence diagrams, activity diagrams, verification & validation
6. **06_project_plan.tex** - Project phases, timeline, risk management, budget
7. **07_conclusion.tex** - Summary, innovations, contributions, future work, recommendations

### Assets
- `assets/` - Contains all diagrams and images
  - `aastu_logo.jpg` - University logo
  - `uc_diagrams/` - Use case diagrams
  - `seq_diagrams/` - Sequence diagrams
  - `activity_diagrams/` - Activity diagrams
  - `state_machine_diagrams/` - State machine diagrams

## Presentation Statistics

- **Total Slides:** 101 slides
- **Sections:** 7 major sections
- **Coverage:** Complete documentation from Chapters 1-3 of the report
- **Diagrams:** 25+ UML diagrams included
- **Tables:** 15+ detailed requirement tables

## Content Coverage

### Chapter 1: Introduction (18 slides)
- Background and digital transformation context
- Problem statement (3 slides covering all aspects)
- General and specific objectives
- Scope and limitations
- Methodology (Agile Scrum framework)
- Research methods (interviews, literature review, observational studies)
- SDLC phases
- Technology stack
- Significance of the study
- Report structure overview

### Chapter 2: Literature Review (12 slides)
- Evolution of online assessment (5 phases)
- Key related works (Topuz, Kurniati, Hussein, Arno, Jia & He, LLM-based studies)
- Milestones by phase
- Critical gaps identified (5 major gaps)
- Lessons learned (8 key lessons)

### Chapter 3: Problem Analysis (4 slides)
- Existing systems in Ethiopia (foreign and local)
- Technical limitations
- Economic and operational limitations
- Ethical and regulatory limitations

### Requirements Specification (15 slides)
- Requirements overview
- Multi-tenancy requirements
- Authentication requirements
- Examination management requirements
- Candidate interface requirements
- AI proctoring requirements
- Grading and analytics requirements
- Dashboard and reporting requirements
- Payment and subscription requirements
- Non-functional requirements (performance, security, reliability, usability, compliance, scalability)

### System Modeling (25 slides)
- System actors
- Use case overview and diagrams
- Detailed use cases (UC-01, UC-08, UC-18)
- Sequence diagrams (8 detailed workflows)
  - Enterprise registration & approval
  - Subscription payment & dunning
  - Candidate login with face verification
  - Session timeout & re-authentication
  - Bulk enrollment & token generation
  - Exam configuration
  - Answer auto-save & sync
  - Proctoring event loop
  - Hybrid grading workflow
  - Async report generation
- Activity diagrams (4 workflows)
  - Enterprise registration & authentication
  - Exam creation & scheduling
  - Candidate exam + AI proctoring
  - Grading + certificate generation
- Verification & validation
- Traceability analysis

### Project Plan (12 slides)
- Plan of activities overview
- Phase 1: Requirements analysis & research
- Phase 2: System design
- Phase 3: Implementation (2 slides)
- Phase 4: Testing & QA (2 slides)
- Phase 5: Documentation & finalization
- Timeline overview
- Risk management & monitoring
- Budget breakdown and justification

### Conclusion (15 slides)
- Project summary
- Key innovations (3 major innovations)
- Addressing literature gaps
- Academic and practical contributions
- Benefits to stakeholders
- Challenges and lessons learned
- Future work (short-term and long-term)
- Recommendations
- Expected impact
- Alignment with Digital Ethiopia 2030
- Final thoughts
- Thank you slide
- Backup slides (technology stack, security measures)

## Compilation Instructions

### Prerequisites
- LaTeX distribution (TeX Live, MiKTeX, or MacTeX)
- Beamer class
- Required packages: graphicx, booktabs, tikz, caption, hyperref, longtable, tabularx, array, multirow, adjustbox

### Compile the Presentation

```bash
# Navigate to the presentation directory
cd veritas_presentation

# Compile (run twice for proper references)
pdflatex presentation.tex
pdflatex presentation.tex

# Or use the automated script
pdflatex -interaction=nonstopmode presentation.tex && pdflatex -interaction=nonstopmode presentation.tex
```

### Output
- `presentation.pdf` - Final presentation (101 pages, ~4.7 MB)

## Key Features

### Modular Design
- Each section is in a separate file for easy maintenance
- Can be edited independently
- Easy to add/remove sections

### Comprehensive Coverage
- All documentation content included
- Detailed tables for requirements
- All UML diagrams embedded
- Complete traceability from requirements to design

### Professional Formatting
- Madrid theme with whale color scheme
- Professional fonts
- Numbered captions
- Frame numbers in footer
- Proper spacing and layout

### Accessibility
- Clear structure with table of contents
- Consistent formatting
- High-contrast color scheme
- Readable font sizes

## Customization

### Changing Theme
Edit `presentation.tex`:
```latex
\usetheme{Madrid}        % Change to: Berlin, Copenhagen, etc.
\usecolortheme{whale}    % Change to: dolphin, orchid, etc.
```

### Adding Slides
1. Open the appropriate section file in `sections/`
2. Add new frame:
```latex
\begin{frame}{Slide Title}
    Content here
\end{frame}
```

### Modifying Content
- Edit section files directly
- Recompile to see changes
- All changes are automatically reflected in the main presentation

## Notes for Presenters

### Estimated Presentation Time
- **Full presentation:** 90-120 minutes
- **Condensed version:** 45-60 minutes (select key slides from each section)
- **Defense presentation:** 30-45 minutes (focus on introduction, methodology, system design, and conclusion)

### Suggested Condensed Version (for 45-minute defense)
1. Introduction: Slides 1-8 (Background, Problem, Objectives)
2. Literature Review: Slides 19-21, 24-26 (Evolution, Gaps)
3. Requirements: Slides 35-37, 42-44 (Overview, Key requirements)
4. System Modeling: Slides 50-52, 55-56, 59-64 (Actors, Use cases, Key sequence diagrams)
5. Project Plan: Slides 76-78, 82-84 (Phases, Timeline, Budget)
6. Conclusion: Slides 88-91, 94-98 (Summary, Innovations, Future work)

### Presentation Tips
- Use presenter notes for detailed explanations
- Highlight key points on each slide
- Pause at diagram slides for questions
- Reference specific requirement IDs when discussing features
- Connect back to problem statement throughout

## Maintenance

### Updating Diagrams
1. Replace image files in `assets/` subdirectories
2. Keep same filenames or update references in section files
3. Recompile presentation

### Adding New Sections
1. Create new file: `sections/XX_section_name.tex`
2. Add `\input{sections/XX_section_name.tex}` to `presentation.tex`
3. Update table of contents

## Version History

- **Version 1.0** (January 2026) - Initial comprehensive presentation
  - 101 slides covering all documentation
  - Modular structure with 7 sections
  - All UML diagrams and requirement tables included

## Contact

For questions or modifications, contact the Veritas team:
- Abraham Mulugeta (ETS0107/14)
- Alazar Gebre (ETS0132/14)
- Tadiyos Dejene (ETS1522/14)
- Tamirat Dejene (ETS1518/14)
- Yohannes Tigistu (ETS1703/14)

**Advisor:** Mr. Muleta Taye (PhD)

**Institution:** Department of Software Engineering, Addis Ababa Science and Technology University
