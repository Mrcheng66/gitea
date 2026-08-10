<script setup lang="ts">
import {computed} from 'vue';
import type {OrgProjectField, OrgProjectMetric} from './types.ts';

const props = defineProps<{fields: OrgProjectField[], labels: Record<string, string>}>();
const metrics = defineModel<OrgProjectMetric[]>({required: true});
const activeFields = computed(() => props.fields.filter((item) => !item.archived));

function label(key: string) {
  return props.labels[key] || key;
}

function format(key: string, value: number) {
  return label(key).replace('{n}', String(value));
}

function addMetric() {
  const index = metrics.value.length + 1;
  metrics.value.push({key: `metric_${index}`, label: format('newMetric', index), aggregation: 'count'});
}
</script>

<template>
  <section id="org-project-settings-metrics" class="ui segment org-project-settings-panel" aria-labelledby="org-project-settings-metrics-title">
    <header class="org-project-settings-panel-heading">
      <div>
        <h3 id="org-project-settings-metrics-title">{{ label('metrics') }}</h3>
        <p>{{ label('metricsDescription') }}</p>
      </div>
      <button type="button" class="ui small button" @click="addMetric">{{ label('addMetric') }}</button>
    </header>
    <div v-if="metrics.length" class="org-project-rule-list org-project-settings-panel-body">
      <div v-for="(metric, index) in metrics" :key="index" class="org-project-rule org-project-metric-rule">
        <span class="org-project-rule-index">{{ index + 1 }}</span>
        <div class="field"><label>{{ label('key') }}</label><input v-model.trim="metric.key"></div>
        <div class="field"><label>{{ label('label') }}</label><input v-model.trim="metric.label"></div>
        <div class="field"><label>{{ label('aggregation') }}</label><select v-model="metric.aggregation" class="ui fluid selection dropdown org-project-dropdown"><option value="count">{{ label('count') }}</option><option value="average">{{ label('average') }}</option></select></div>
        <div class="field"><label>{{ label('valueField') }}</label><select v-model="metric.field_key" class="ui fluid selection dropdown org-project-dropdown"><option value="">{{ label('projects') }}</option><option v-for="field in activeFields" :key="field.key" :value="field.key">{{ field.label }}</option></select></div>
        <div class="field"><label>{{ label('groupBy') }}</label><select v-model="metric.group_by" class="ui fluid selection dropdown org-project-dropdown"><option value="">{{ label('none') }}</option><option v-for="field in activeFields" :key="field.key" :value="field.key">{{ field.label }}</option></select></div>
        <button type="button" class="ui tiny basic red button org-project-rule-remove" :aria-label="label('removeMetric')" @click="metrics.splice(index, 1)">{{ label('remove') }}</button>
      </div>
    </div>
    <div v-else class="org-project-settings-empty">{{ label('emptyMetrics') }}</div>
  </section>
</template>
