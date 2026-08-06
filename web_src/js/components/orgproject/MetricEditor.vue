<script setup lang="ts">
import type {OrgProjectField, OrgProjectMetric} from './types.ts';

const props = defineProps<{fields: OrgProjectField[]}>();
const metrics = defineModel<OrgProjectMetric[]>({required: true});

function addMetric() {
  const index = metrics.value.length + 1;
  metrics.value.push({key: `metric_${index}`, label: `Metric ${index}`, aggregation: 'count'});
}
</script>

<template>
  <section class="org-project-editor-section">
    <div class="flex-left-right tw-mb-3"><h3 class="ui header tw-m-0">Metrics</h3><button type="button" class="ui small button" @click="addMetric">Add metric</button></div>
    <div v-for="(metric, index) in metrics" :key="index" class="org-project-editor-grid tw-mt-3">
      <div class="field"><label>Key</label><input v-model.trim="metric.key"></div>
      <div class="field"><label>Label</label><input v-model.trim="metric.label"></div>
      <div class="field"><label>Aggregation</label><select v-model="metric.aggregation"><option value="count">Count</option><option value="average">Average</option></select></div>
      <div class="field"><label>Value field</label><select v-model="metric.field_key"><option value="">Projects</option><option v-for="field in fields.filter((item) => !item.archived)" :key="field.key" :value="field.key">{{ field.label }}</option></select></div>
      <div class="field"><label>Group by</label><select v-model="metric.group_by"><option value="">None</option><option v-for="field in fields.filter((item) => !item.archived)" :key="field.key" :value="field.key">{{ field.label }}</option></select></div>
      <button type="button" class="ui icon basic red button org-project-remove-button" aria-label="Remove metric" @click="metrics.splice(index, 1)">×</button>
    </div>
  </section>
</template>
