export type OrgProjectFieldType = 'short_text' | 'long_text' | 'single_select' | 'multi_select' | 'integer' | 'decimal' | 'percent' | 'date' | 'date_time' | 'boolean' | 'member' | 'member_array';

export type OrgProjectOption = {
  key: string,
  label: string,
  order: number,
};

export type OrgProjectField = {
  key: string,
  label: string,
  type: OrgProjectFieldType,
  order: number,
  required?: boolean,
  archived?: boolean,
  default?: unknown,
  migration_strategy?: string,
  options?: OrgProjectOption[],
};

export type ProjectFormGroupKey = 'plan' | 'status' | 'other';

const planFieldKeys = new Set(['stage', 'owner', 'followers', 'start_date', 'target_date']);
const statusFieldKeys = new Set(['progress', 'risk', 'summary', 'current_problem', 'next_action', 'next_action_owner', 'next_action_due']);

export function groupProjectFields(fields: OrgProjectField[]): Record<ProjectFormGroupKey, OrgProjectField[]> {
  const groups: Record<ProjectFormGroupKey, OrgProjectField[]> = {plan: [], status: [], other: []};
  for (const field of fields) {
    if (planFieldKeys.has(field.key)) groups.plan.push(field);
    else if (statusFieldKeys.has(field.key)) groups.status.push(field);
    else groups.other.push(field);
  }
  return groups;
}

export type OrgProjectSort = {
  field_key: string,
  direction: 'asc' | 'desc',
};

export type OrgProjectListView = {
  columns: string[],
  sort?: OrgProjectSort[],
};

export type OrgProjectFilter = {
  key: string,
  label: string,
  field_key: string,
  operator: string,
  value?: unknown,
};

export type OrgProjectMetric = {
  key: string,
  label: string,
  aggregation: 'count' | 'average',
  field_key?: string,
  group_by?: string,
};

export type OrgProjectSchema = {
  schema_version: number,
  fields: OrgProjectField[],
  list_view: OrgProjectListView,
  filters: OrgProjectFilter[],
  metrics: OrgProjectMetric[],
};

export type OrgProjectMember = {
  id: number,
  name: string,
  full_name: string,
};

export type OrgProjectDisplayField = {
  Key: string,
  Label: string,
  Value: string,
  Type: OrgProjectFieldType,
  Raw: string,
  Number?: number,
  Members?: OrgProjectMember[],
};

export type OrgProjectListRow = {
  Project: {
    ID: number,
    Slug: string,
    Name: string,
    Description: string,
    Lifecycle: string,
    UpdatedUnix: number,
  },
  Fields: OrgProjectDisplayField[],
};

export type OrgProjectMetricDisplay = {
  Key: string,
  Label: string,
  Buckets: Array<{Bucket?: string | number | boolean, Value: number}>,
};

export type OrgProjectSummary = {
  active: number,
  blocked: number,
  overdue: number,
  due_soon: number,
  average_progress: number,
};

export type OrgProjectTeam = {
  id: number,
  name: string,
  description: string,
  selected: boolean,
  deleted: boolean,
};

export type OrgProjectActivityCommit = {
  repository_id: number,
  repository_full_name: string,
  repository_link: string,
  sha: string,
  short_sha: string,
  link: string,
  message: string,
  author_name: string,
  committed_at: string,
};

export type OrgProjectProgressEvent = {
  kind: 'release' | 'pull_merged' | 'issue_closed' | 'commit',
  title: string,
  link: string,
  repository_id: number,
  repository_full_name: string,
  repository_link: string,
  author_name: string,
  occurred_at: string,
};

export type OrgProjectActivityRepository = {
  id: number,
  full_name: string,
  link: string,
  open_pulls: number,
  merged_pulls: number,
  release_count: number,
  latest_release_at?: string,
};

export type ProjectWorkbenchPerson = {
  id: number,
  name: string,
  full_name: string,
  link: string,
  owned: number,
  participating: number,
  projects: string[],
};

export type ProjectWorkbenchProject = {
  id: number,
  name: string,
  description: string,
  link: string,
  organization: string,
  organization_url: string,
  stage_key: string,
  stage: string,
  risk_key: string,
  risk: string,
  progress: number,
  owner?: ProjectWorkbenchPerson,
  participants: ProjectWorkbenchPerson[],
  current_problem: string,
  next_action: string,
  next_action_owner?: ProjectWorkbenchPerson,
  next_action_due: string,
  target_date: string,
  overdue: boolean,
  due_soon: boolean,
  stale: boolean,
  expanded: boolean,
  activity: OrgProjectActivitySummary,
  activity_error: boolean,
  updated_at: string,
  latest_event_at?: string,
};

export type ProjectWorkbenchResult = {
  projects: ProjectWorkbenchProject[],
  people: ProjectWorkbenchPerson[],
  attention: {blocked: number, overdue: number, due_soon: number, stale: number, unowned: number},
  configured_organizations: number,
};

export type OrgProjectActivitySummary = {
  since: string,
  repositories: OrgProjectActivityRepository[],
  commits: OrgProjectActivityCommit[],
  open_pulls: number,
  merged_pulls: number,
  release_count: number,
  latest_release_at?: string,
  progress: OrgProjectProgressEvent[],
  partial: boolean,
};

export function cloneJSON<T>(value: T): T {
  // eslint-disable-next-line unicorn/prefer-structured-clone -- structuredClone rejects Vue proxies.
  return JSON.parse(JSON.stringify(value)) as T;
}

export function sortedActiveFields(schema: OrgProjectSchema): OrgProjectField[] {
  return schema.fields.filter((field) => !field.archived).toSorted((a, b) => a.order - b.order || a.key.localeCompare(b.key));
}

export function initialProjectValues(schema: OrgProjectSchema, values: Record<string, unknown>): Record<string, unknown> {
  const result = cloneJSON(values);
  for (const field of sortedActiveFields(schema)) {
    if (!(field.key in result) && field.default !== undefined) result[field.key] = cloneJSON(field.default);
    if (!(field.key in result) && field.type === 'boolean') result[field.key] = false;
    if (!(field.key in result) && (field.type === 'multi_select' || field.type === 'member_array')) result[field.key] = [];
    const value = result[field.key];
    if (field.type === 'date_time' && typeof value === 'string') {
      const date = new Date(value);
      if (!Number.isNaN(date.valueOf())) result[field.key] = new Date(date.valueOf() - date.getTimezoneOffset() * 60000).toISOString().slice(0, 16);
    }
  }
  return result;
}

export function serializeProjectValues(schema: OrgProjectSchema, values: Record<string, unknown>): string {
  const result: Record<string, unknown> = {};
  for (const field of sortedActiveFields(schema)) {
    const value = values[field.key];
    if (value === undefined || value === null || value === '') continue;
    if (field.type === 'date_time' && typeof value === 'string') {
      result[field.key] = new Date(value).toISOString();
    } else {
      result[field.key] = value;
    }
  }
  return JSON.stringify(result);
}

export function normalizeSchema(schema: OrgProjectSchema): OrgProjectSchema {
  const normalized = cloneJSON(schema);
  normalized.schema_version = 1;
  normalized.fields ??= [];
  normalized.list_view ??= {columns: []};
  normalized.list_view.columns ??= [];
  normalized.list_view.sort ??= [];
  normalized.filters ??= [];
  normalized.metrics ??= [];
  for (const [index, field] of normalized.fields.entries()) {
    field.order = index;
    if (field.type === 'single_select' || field.type === 'multi_select') {
      field.options ??= [];
      for (const [optionIndex, option] of field.options.entries()) option.order = optionIndex;
    } else {
      delete field.options;
    }
  }
  return normalized;
}

export function validateSchemaClient(schema: OrgProjectSchema): string[] {
  const errors: string[] = [];
  const keys = new Set<string>();
  for (const field of schema.fields) {
    if (!/^[a-z][a-z0-9_]{0,63}$/.test(field.key)) errors.push(`Invalid field key: ${field.key || '(empty)'}`);
    if (keys.has(field.key)) errors.push(`Duplicate field key: ${field.key}`);
    keys.add(field.key);
    if (!field.label.trim()) errors.push(`Field ${field.key || '(empty)'} needs a label`);
    if ((field.type === 'single_select' || field.type === 'multi_select') && !field.options?.length) errors.push(`Field ${field.key} needs at least one option`);
  }
  return errors;
}
