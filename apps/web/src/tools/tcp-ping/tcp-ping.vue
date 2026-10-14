<script setup lang="ts">
import PingTool from '@/components/PingTool.vue'
</script>

<template>
  <PingTool kind="tcp" path="/v1/tcping/stream" />
</template>
