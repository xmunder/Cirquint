import type { APIRequestContext } from "@playwright/test";
import { expect, test } from "@playwright/test";

const controlUrl = "http://127.0.0.1:4100/__playwright__/scenario";
const stateUrl = "http://127.0.0.1:4100/__playwright__/state";

test.describe("circuit review workbench", () => {
  test("redirects approve submissions from queue to success state", async ({ page, request }) => {
    await setScenario(request, "approve-success");

    await page.goto("/projects/prj-001/reviews");
    await expect(page.getByRole("heading", { name: "Review queue" })).toBeVisible();
    await page.getByRole("link", { name: "Open review" }).click();

    await expect(page.getByRole("heading", { name: "Review detail" })).toBeVisible();
    await page.getByLabel("Review note").fill("approve after human review");
    await page.getByLabel("Corrected spec JSON").fill('{"confidence":0.99,"warnings":["human-corrected"]}');
    await page.getByRole("button", { name: "Submit decision" }).click();

    await expect(page).toHaveURL(/\/projects\/prj-001\/reviews\?success=approve$/);
    await expect(page.getByText("Review approved and removed from the queue.")).toBeVisible();
    await expect(page.getByText("No pending reviews for this project.")).toBeVisible();
  });

  test("redirects reject submissions from queue to success state", async ({ page, request }) => {
    await setScenario(request, "reject-success");

    await page.goto("/projects/prj-001/reviews");
    await page.getByRole("link", { name: "Open review" }).click();

    await page.getByLabel("Reject").check();
    await expect(page.getByLabel("Corrected spec JSON")).toHaveCount(0);
    await page.getByLabel("Review note").fill("reject until extraction is fixed");
    await page.getByRole("button", { name: "Submit decision" }).click();

    await expect(page).toHaveURL(/\/projects\/prj-001\/reviews\?success=reject$/);
    await expect(page.getByText("Review rejected and removed from the queue.")).toBeVisible();
  });

  test("shows backend errors on the detail page and allows retry", async ({ page, request }) => {
    await setScenario(request, "retry-error");

    await page.goto("/projects/prj-001/reviews/job-001");
    await page.getByLabel("Review note").fill("retry after backend recovers");
    await page.getByRole("button", { name: "Submit decision" }).click();

    await expect(page).toHaveURL(/\/projects\/prj-001\/reviews\/job-001$/);
    await expect(page.getByText("backend unavailable")).toBeVisible();
    await expect.poll(() => readDecisionCalls(request)).toBe(1);

    await page.getByLabel("Review note").fill("retry after backend recovers");
    await page.getByRole("button", { name: "Submit decision" }).click();
    await expect.poll(() => readDecisionCalls(request)).toBe(2);

    await expect(page).toHaveURL(/\/projects\/prj-001\/reviews\?success=approve$/);
    await expect(page.getByText("Review approved and removed from the queue.")).toBeVisible();
  });

  test("returns stale conflicts to the queue with the stale banner", async ({ page, request }) => {
    await setScenario(request, "stale-conflict");

    await page.goto("/projects/prj-001/reviews/job-001");
    await page.getByLabel("Reject").check();
    await page.getByLabel("Review note").fill("resolved elsewhere");
    await page.getByRole("button", { name: "Submit decision" }).click();

    await expect(page).toHaveURL(/\/projects\/prj-001\/reviews\?stale=1$/);
    await expect(page.getByText("That revision is no longer current. Pick the latest queued item.")).toBeVisible();
    await expect(page.getByText("cir-002")).toBeVisible();
  });
});

async function setScenario(request: APIRequestContext, scenario: string) {
  const response = await request.post(controlUrl, {
    data: { scenario },
  });

  expect(response.ok()).toBe(true);
}

async function readDecisionCalls(request: APIRequestContext): Promise<number> {
  const response = await request.get(stateUrl);
  expect(response.ok()).toBe(true);

  const body = (await response.json()) as { decisionCalls: number };
  return body.decisionCalls;
}
