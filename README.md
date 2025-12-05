# Project ETS: "Employer Tracking System"

## Background

_If recruiters use AI to find candidates, why can't we do it?_

The purpose of this project is to make job search more fair. With autorejects, ghost postings and tedious processes of re-writing 
your resume to annoying forms, we all burn out. And for what? For AI to choose less qualified candidates with AI-generated resumes 
that contain keywords that AI tracks over the person that is more fit for the position?

We want to return self-confidence to candidates and give them a tool that makes job search fair.

**This is only a part of a bigger project. This MCP server is only responsible for collecting jobs at this current moment.**

## MCP server tools and their purposes

- `job_search`
Accepts query/filters and returns structured job objects. It’s the data feed the client/LLM reads to understand postings.
- `persist_keywords`
It is proven that AI mostly looks at keywords and not if candidate is a good fit, so this is necessary. Takes `{job_id, keywords[], optional confidence/notes}` 
payloads and writes them into the job store/graph so downstream tools have durable keyword data. 
- `job_analysis`
Given job IDs (and optionally a profile/focus string) it pulls stored jobs+keywords to produce match analysis, prep notes, 
or prioritization using Graph RAG pipeline.
- `graph_tool`
Developer utility; focuses on Cypher queries or graph inspection, independent from the user-facing flow.
- `sheets_export`
Takes either job IDs (server refetches data) or fully specified rows (ID, title, keywords, notes) 
along with spreadsheet metadata and writes them to Google Sheets.

## User Flow
TODO