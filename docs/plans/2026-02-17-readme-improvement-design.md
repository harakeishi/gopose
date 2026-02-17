# README Improvement Design

## Date: 2026-02-17

## Goal

Restructure the README to be concise, scannable, and user-focused, following patterns from notable OSS projects (lazydocker, fzf, ripgrep).

## Target Audience

- Primary: Developers who want to use gopose to resolve Docker Compose port conflicts
- Secondary: Contributors (detailed dev info moves to separate docs)

## Language

- All English (including output examples)

## Approach

lazydocker-style minimal & visual-focused, with fzf-style progressive disclosure elements.

## Section Structure

1. **Hero Section** - Logo + badges + one-line description
2. **The Problem / Why gopose?** - Before/After showing the pain point and solution
3. **Quick Start** - Install + one command to run
4. **Installation** - Go Install / Binary Releases / Build from Source
5. **Usage** - Basic commands + common options table
6. **Configuration** - Minimal .gopose.yaml example + link to full docs
7. **Documentation** - Links to detailed docs in docs/
8. **Contributing** - Brief dev setup + link to details
9. **License + Contributors + Footer**

## Content Migration

| Current Section | Destination |
|---|---|
| Reserved Ports (detailed) | `docs/reserved-ports.md` |
| Network Conflict Avoidance (detailed) | `docs/network-conflict.md` |
| Configuration File (detailed) | `docs/configuration.md` |
| Directory Structure | Remove (internal detail) |
| Output Example (long Japanese logs) | Replace with short English table output |
| --detail flag output | Move to docs/ |

## Output Example Redesign

Replace 40+ line Japanese log output with concise table-format resolution summary.

## Key Principles

- 5-second rule: Visitor understands what the tool does within 5 seconds
- Progressive disclosure: Basic info in README, details in docs/
- Copy-pasteable: All commands can be directly pasted into terminal
- Scannable: Tables and short sections over walls of text
