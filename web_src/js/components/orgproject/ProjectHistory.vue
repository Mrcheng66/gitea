<script setup lang="ts">
type HistorySnapshot = Record<string, unknown>;

type HistoryChange = {
  id: number,
  actor_id: number,
  actor_name?: string,
  actor_link?: string,
  request_id: string,
  changed_fields: string[],
  before: HistorySnapshot,
  after: HistorySnapshot,
  source: string,
  created_at: string,
};

type Locale = {
  changed: string,
  user: string,
  source: string,
  sourceWeb: string,
  sourceApi: string,
  sourceLegacyImport: string,
  details: string,
  before: string,
  after: string,
  requestID: string,
};

const props = defineProps<{
  changes: HistoryChange[],
  fieldLabels: Record<string, string>,
  emptyText: string,
  locale: Locale,
}>();

function actorName(change: HistoryChange) {
  return change.actor_name || `${props.locale.user} #${change.actor_id}`;
}

function fieldLabel(field: string) {
  return props.fieldLabels[field] || (field.startsWith('values.') ? field.slice('values.'.length) : field);
}

function sourceLabel(source: string) {
  if (source === 'web') return props.locale.sourceWeb;
  if (source === 'api') return props.locale.sourceApi;
  if (source === 'legacy-import') return props.locale.sourceLegacyImport;
  return source;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

function fieldValue(snapshot: HistorySnapshot, field: string) {
  if (!field.startsWith('values.')) return snapshot[field];

  const key = field.slice('values.'.length);
  const values = Array.isArray(snapshot.values) ? snapshot.values : [];
  const value = values.find((item) => isRecord(item) && item.key === key);
  if (!isRecord(value)) return undefined;

  for (const property of ['text', 'number', 'time', 'bool', 'user_id']) {
    if (property in value) return value[property];
  }
  if (typeof value.json === 'string') {
    try {
      return JSON.parse(value.json);
    } catch {
      return value.json;
    }
  }
  return undefined;
}

function formatValue(value: unknown) {
  if (value === undefined || value === null || value === '') return '—';
  if (typeof value === 'string') return value;
  return JSON.stringify(value, null, 2);
}
</script>

<template>
  <div v-if="changes.length" class="org-project-history-list">
    <article v-for="change in changes" :key="change.id" class="org-project-history-item">
      <div class="org-project-history-marker" aria-hidden="true"/>
      <div class="org-project-history-content">
        <div class="org-project-history-heading">
          <div class="org-project-history-summary">
            <a v-if="change.actor_link" :href="change.actor_link" class="org-project-history-actor">{{ actorName(change) }}</a>
            <strong v-else class="org-project-history-actor">{{ actorName(change) }}</strong>
            <span>{{ locale.changed }}</span>
            <span class="org-project-history-fields">
              <span v-for="field in change.changed_fields" :key="field" class="ui label">{{ fieldLabel(field) }}</span>
            </span>
          </div>
          <div class="org-project-history-meta text light-2">
            <relative-time :datetime="change.created_at" prefix=""/>
            <span>· {{ locale.source }}: {{ sourceLabel(change.source) }}</span>
          </div>
        </div>

        <details class="org-project-history-details">
          <summary>{{ locale.details }}</summary>
          <div class="org-project-history-changes">
            <section v-for="field in change.changed_fields" :key="field" class="org-project-history-change">
              <h4>{{ fieldLabel(field) }}</h4>
              <div class="org-project-history-diff">
                <div>
                  <span class="org-project-history-value-label">{{ locale.before }}</span>
                  <pre>{{ formatValue(fieldValue(change.before, field)) }}</pre>
                </div>
                <div>
                  <span class="org-project-history-value-label">{{ locale.after }}</span>
                  <pre>{{ formatValue(fieldValue(change.after, field)) }}</pre>
                </div>
              </div>
            </section>
          </div>
          <div class="org-project-history-request text light-2">
            <span>{{ locale.requestID }}</span>
            <code>{{ change.request_id }}</code>
          </div>
        </details>
      </div>
    </article>
  </div>
  <div v-else class="ui placeholder segment">{{ emptyText }}</div>
</template>
