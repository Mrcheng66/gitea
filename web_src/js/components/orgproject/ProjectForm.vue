<script setup lang="ts">
import {computed, ref} from 'vue';
import {initialProjectValues, serializeProjectValues, sortedActiveFields} from './types.ts';
import type {OrgProjectMember, OrgProjectSchema} from './types.ts';

const props = defineProps<{
  schema: OrgProjectSchema,
  initialValues: Record<string, unknown>,
  members: OrgProjectMember[],
}>();

const values = ref<Record<string, any>>(initialProjectValues(props.schema, props.initialValues));
const fields = computed(() => sortedActiveFields(props.schema));
const serializedValues = computed(() => serializeProjectValues(props.schema, values.value));

function inputID(key: string) {
  return `org-project-field-${key}`;
}
</script>

<template>
  <div class="org-project-form-fields">
    <div v-for="field in fields" :key="field.key" class="field" :class="{required: field.required}">
      <label :for="inputID(field.key)">{{ field.label }}</label>
      <textarea v-if="field.type === 'long_text'" :id="inputID(field.key)" v-model="values[field.key]" rows="4" :required="field.required"/>
      <select v-else-if="field.type === 'single_select'" :id="inputID(field.key)" v-model="values[field.key]" :required="field.required">
        <option v-if="!field.required" value="">—</option>
        <option v-for="option in field.options" :key="option.key" :value="option.key">{{ option.label }}</option>
      </select>
      <select v-else-if="field.type === 'multi_select'" :id="inputID(field.key)" v-model="values[field.key]" multiple>
        <option v-for="option in field.options" :key="option.key" :value="option.key">{{ option.label }}</option>
      </select>
      <select v-else-if="field.type === 'member'" :id="inputID(field.key)" v-model.number="values[field.key]" :required="field.required">
        <option v-if="!field.required" :value="null">—</option>
        <option v-for="member in members" :key="member.id" :value="member.id">{{ member.full_name || member.name }}</option>
      </select>
      <select v-else-if="field.type === 'member_array'" :id="inputID(field.key)" v-model="values[field.key]" multiple>
        <option v-for="member in members" :key="member.id" :value="member.id">{{ member.full_name || member.name }}</option>
      </select>
      <div v-else-if="field.type === 'boolean'" class="ui checkbox">
        <input :id="inputID(field.key)" v-model="values[field.key]" type="checkbox">
        <label :for="inputID(field.key)">{{ field.label }}</label>
      </div>
      <input v-else-if="field.type === 'integer'" :id="inputID(field.key)" v-model.number="values[field.key]" type="number" step="1" :required="field.required">
      <input v-else-if="field.type === 'decimal'" :id="inputID(field.key)" v-model.number="values[field.key]" type="number" step="any" :required="field.required">
      <input v-else-if="field.type === 'percent'" :id="inputID(field.key)" v-model.number="values[field.key]" type="number" step="0.01" min="0" max="100" :required="field.required">
      <input v-else-if="field.type === 'date'" :id="inputID(field.key)" v-model="values[field.key]" type="date" :required="field.required">
      <input v-else-if="field.type === 'date_time'" :id="inputID(field.key)" v-model="values[field.key]" type="datetime-local" :required="field.required">
      <input v-else :id="inputID(field.key)" v-model.trim="values[field.key]" type="text" :required="field.required">
    </div>
    <input type="hidden" name="values" :value="serializedValues">
  </div>
</template>
