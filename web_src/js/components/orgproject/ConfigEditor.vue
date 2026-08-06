<script setup lang="ts">
import {computed, ref} from 'vue';
import FieldEditor from './FieldEditor.vue';
import MetricEditor from './MetricEditor.vue';
import ViewEditor from './ViewEditor.vue';
import {cloneJSON, normalizeSchema, validateSchemaClient} from './types.ts';
import type {OrgProjectSchema} from './types.ts';

const props = defineProps<{schema: OrgProjectSchema}>();
const schema = ref(normalizeSchema(cloneJSON(props.schema)));
const normalized = computed(() => normalizeSchema(schema.value));
const errors = computed(() => validateSchemaClient(normalized.value));
const serialized = computed(() => JSON.stringify(normalized.value));
</script>

<template>
  <div class="org-project-config-editor">
    <FieldEditor v-model="schema.fields"/>
    <ViewEditor v-model:list-view="schema.list_view" v-model:filters="schema.filters" :fields="schema.fields"/>
    <MetricEditor v-model="schema.metrics" :fields="schema.fields"/>
    <div v-if="errors.length" class="ui warning message" role="alert">
      <div class="header">Review the configuration</div>
      <ul><li v-for="error in errors" :key="error">{{ error }}</li></ul>
    </div>
    <textarea class="org-project-schema-fallback" name="schema" :value="serialized" aria-label="Organization project schema JSON"/>
  </div>
</template>
