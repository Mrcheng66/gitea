<script setup lang="ts">
defineProps<{changes: Array<Record<string, unknown>>, emptyText: string}>();

function formatJSON(value: unknown) {
  if (typeof value === 'string') {
    try {
      return JSON.stringify(JSON.parse(value), null, 2);
    } catch {
      return value;
    }
  }
  return JSON.stringify(value, null, 2);
}
</script>

<template>
  <div v-if="changes.length" class="ui divided list">
    <details v-for="(change, index) in changes" :key="String(change.ID ?? index)" class="item org-project-history-entry">
      <summary class="tw-cursor-pointer tw-font-semibold">
        {{ change.Source || 'change' }} · {{ change.RequestID || `#${change.ID}` }}
      </summary>
      <div class="org-project-history-diff tw-mt-3">
        <div><h4>Before</h4><pre>{{ formatJSON(change.Before) }}</pre></div>
        <div><h4>After</h4><pre>{{ formatJSON(change.After) }}</pre></div>
      </div>
    </details>
  </div>
  <div v-else class="ui placeholder segment">{{ emptyText }}</div>
</template>
