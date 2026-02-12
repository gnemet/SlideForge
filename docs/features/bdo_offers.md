# BDO Offer Generation Knowledge

This document captures domain-specific knowledge for generating BDO client offers using SlideForge.

## Offer Structure

Standard BDO Financial (FDD) and Tax (TDD) offers typically follow this structure:
- **Phase 1: Analysis**: "Red Flag" reporting style, emphasizing critical risks.
- **Phase 2: Reporting**: Draft report within 14-17 business days.
- **Pricing**: Fixed fee model with specific hourly rates (e.g., 180 EUR) for out-of-scope work.

## Metadata Patterns

The following metadata fields are used to track and generate these offers:

| Field | Description | Type |
|-------|-------------|------|
| `service_line` | Engagement type (FDD, TDD, IA) | String |
| `lead_partner` | Responsible BDO Partner | String |
| `audit_period` | Timeframe for review | String |
| `total_fee` | Contract value | Numeric |
| `currency` | Fee currency (EUR, HUF) | String |

## Usage in SlideForge
This metadata can be stored in the `metadata` column of the `pptx_files` table and used during slide generation to populate placeholders or inform AI summaries.
