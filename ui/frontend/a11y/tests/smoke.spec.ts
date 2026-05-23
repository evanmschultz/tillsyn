import { test, expect } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';

test.describe('GET / — root route a11y', () => {
  test('axe-core: zero serious/critical violations under WCAG 2 AA', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');

    const results = await new AxeBuilder({ page })
      .withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa'])
      .analyze();

    const blockers = results.violations.filter(
      (v) => v.impact === 'serious' || v.impact === 'critical',
    );

    if (blockers.length > 0) {
      const summary = blockers
        .map((v) => `- [${v.impact}] ${v.id}: ${v.help} (${v.nodes.length} node(s))`)
        .join('\n');
      throw new Error(
        `axe-core found ${blockers.length} serious/critical WCAG 2 AA violation(s) on /:\n${summary}`,
      );
    }

    expect(blockers).toEqual([]);
  });
});
