<template>
    <div class="page-state" :class="`page-state--${kind}`" :role="kind === 'error' ? 'alert' : 'status'">
        <div class="page-state__icon" aria-hidden="true">
            <span v-if="kind === 'loading'" class="page-state__spinner"></span>
            <span v-else-if="kind === 'error'">!</span>
            <span v-else-if="kind === 'unavailable'">–</span>
            <span v-else>○</span>
        </div>
        <div class="page-state__body">
            <strong>{{ title }}</strong>
            <span v-if="description">{{ description }}</span>
        </div>
        <button v-if="actionLabel" type="button" @click="$emit('action')">{{ actionLabel }}</button>
    </div>
</template>

<script setup lang="ts">
withDefaults(defineProps<{
    kind?: 'loading' | 'empty' | 'error' | 'unavailable' | 'info'
    title: string
    description?: string
    actionLabel?: string
}>(), {
    kind: 'info',
    description: '',
    actionLabel: '',
})

defineEmits<{
    (event: 'action'): void
}>()
</script>

<style lang="scss" scoped>
.page-state {
    display: flex;
    align-items: center;
    gap: 12px;
    min-height: 64px;
    margin: 12px 0;
    padding: 14px 16px;
    color: var(--flow-span-color);
    background: rgba(85, 58, 254, 0.05);
    border: 1px solid rgba(85, 58, 254, 0.16);
    border-radius: 8px;
}

.page-state--error {
    background: #fff7e6;
    border-color: #ffd591;
}

.page-state--unavailable {
    background: rgba(0, 0, 0, 0.025);
    border-color: rgba(0, 0, 0, 0.12);
}

.page-state__icon {
    display: grid;
    flex: none;
    width: 28px;
    height: 28px;
    place-items: center;
    color: #553afe;
    border: 1px solid currentColor;
    border-radius: 50%;
    font-weight: 600;
}

.page-state--error .page-state__icon {
    color: #ad6800;
}

.page-state__body {
    display: flex;
    flex: 1;
    flex-direction: column;
    gap: 3px;
    min-width: 0;
}

.page-state__body span {
    opacity: 0.72;
    font-size: 13px;
}

button {
    flex: none;
    min-height: 34px;
    padding: 6px 14px;
    color: #553afe;
    background: transparent;
    border: 1px solid #553afe;
    border-radius: 6px;
    cursor: pointer;
}

.page-state__spinner {
    width: 12px;
    height: 12px;
    border: 2px solid rgba(85, 58, 254, 0.25);
    border-top-color: #553afe;
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
}

@keyframes spin {
    to { transform: rotate(360deg); }
}

@media (max-width: 480px) {
    .page-state {
        align-items: flex-start;
        flex-wrap: wrap;
    }

    button {
        width: 100%;
    }
}
</style>
