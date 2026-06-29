<template>
  <div class="feature-card" :class="card.color" @click="$emit('click', card)">
    <div>
      <div class="header">
        <div class="icon-wrapper">
          <slot name="icon">
            <component :is="iconComponents[card.icon]?.component" v-if="card.icon && iconComponents[card.icon]"
              v-bind="iconComponents[card.icon].props" class="icon-svg" />
          </slot>
        </div>
        <!-- <span v-if="card.badge" class="badge">{{ card.badge }}</span> -->
      </div>
      <div class="content">
        <div class="title">{{ card.title }}</div>
        <div class="subtitle">{{ card.subtitle }}</div>
      </div>
      <div class="footer">
        <span v-if="card.status" class="status" :class="{ active: card.isActive }">
          {{ card.status }}
        </span>
        <!-- <span v-if="card.icon == 'baby'" class="status" :class="{ active: type == 1 }">
          {{ type == 1 ? $gettext('已启用') : $gettext('未启用') }}
        </span> -->
        <div v-if="card.extra" class="extra">{{ card.extra }}</div>
        <div v-if="card.num" class="extra badge"> <span class="extra_num">{{ card.num }}</span> {{$gettext("台设备在线")}}</div>
        <span v-if="card.tag" class="badge">{{ card.tag }}</span>
      </div>
    </div>
    <rightArrowIcon class="right-arrow" color="#99a1af" />
  </div>
</template>


<script setup lang="ts">
import navigationIcon from '/@/components/svg/navigation.vue'
import wifiIcon from '/@/components/svg/wifi.vue'
import topologyIcon from '/@/components/svg/topology.vue'
import speedIcon from '/@/components/svg/speed.vue'
import babyIcon from '/@/components/svg/baby.vue'
import appStoreIcon from '/@/components/svg/appStore.vue'
import rightArrowIcon from '/@/components/svg/rightArrow.vue'
import { useGettext } from '/@/plugins/i18n'
const { $gettext } = useGettext()
import request from '/@/request';
import { ref } from 'vue'

// 定义图标类型
type IconType = 'navigation' | 'wifi' | 'topology' | 'speed' | 'baby' | 'appStore';

// 定义颜色主题类型
type ColorTheme = 'purple' | 'blue' | 'green' | 'pink' | 'skyblue' | 'orange';

// 定义卡片接口
interface Card {
  title: string;
  subtitle: string;
  icon: IconType;
  color: ColorTheme;
  badge?: string;
  status?: string;
  isActive?: boolean;
  extra?: string;
  num?: number;
  tag?: string;
}

// 定义图标组件接口
interface IconComponent {
  component: any;
  props: {
    color: string;
  };
}

// 图标组件映射
const iconComponents: Record<IconType, IconComponent> = {
  navigation: {
    component: navigationIcon,
    props: {
      color: '#ffffff'
    }
  },
  wifi: {
    component: wifiIcon,
    props: {
      color: '#ffffff'
    }
  },
  topology: {
    component: topologyIcon,
    props: {
      color: '#ffffff'
    }
  },
  speed: {
    component: speedIcon,
    props: {
      color: '#ffffff'
    }
  },
  baby: {
    component: babyIcon,
    props: {
      color: '#ffffff'
    }
  },
  appStore: {
    component: appStoreIcon,
    props: {
      color: '#ffffff'
    }
  }
};

defineProps({
  card: {
    type: Object as () => Card,
    required: true
  }
})
</script>

<style scoped lang="scss">
.feature-card {
  flex: 1 1 0; // 占满剩余空间并可缩放
  min-width: 280px; // 设置最小宽度
  max-width: 350px;
  padding: 14px 14px 20px 14px;
  border: 2px solid var(--border-color);
  border-radius: 10px;
  cursor: pointer;
  transition: transform 0.2s ease, box-shadow 0.2s ease;
  background-color: var(--card-bg-color);
  display: flex;
  align-items: flex-start; // 让所有卡片顶部对齐，图标保持在同一水平线上
  justify-content: space-between;

  &:hover {
    transform: translateY(-2px);
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
  }

  .badge {
    font-size: 12px;
    padding: 4px 6px;
    border-radius: 6px;
    color: #fff;
    line-height: 1;
  }

  .header {
    display: flex;
    align-items: center;
    gap: 6px;
    margin-bottom: 16px;

    .icon-wrapper {
      width: 40px;
      height: 40px;
      border-radius: 10px;
      display: flex;
      align-items: center;
      justify-content: center;
      background-color: #999;
      color: #fff;

      .icon-svg {
        width: 24px;
        height: 24px;
      }
    }
  }

  .content {
    .title {
      font-weight: bold;
      color: var(--item-label_key-span-color);
      margin-bottom: 8px;
    }

    .subtitle {
      font-size: 14px;
      color: #666;
      margin-top: 2px;
    }
  }

  .footer {
    display: flex;
    align-items: center;
    margin-top: 12px;
    font-size: 12px;

    .status {
      padding: 4px 6px;
      border-radius: 6px;
      background-color: #eee;
      color: #666;
      line-height: 1;
      margin-right: 6px;

      &.active {
        background-color: #d3f9d8;
        color: #38a169;
      }
    }

    .extra {
      color: inherit;

      .extra_num {
        font-size: 16px;
      }
    }
  }

  .right-arrow {
    width: 18px;
    height: 18px;
    align-self: center;
  }

  // 颜色主题
  &.purple {

    .icon-wrapper {
      background-color: #ad46ff;
    }

    .badge {
      background-color: #f3e8ff;
      color: #8200db;
    }
  }

  &.blue {
    // border-color: #2b7fff;
    // background: #e4f0ff;

    .icon-wrapper {
      background-color: #3b82f6;
    }

    .title {
      // color: #193cb8;
    }

    .subtitle,
    .extra {
      // color: #155dfc;
    }

    .badge {
      background-color: #e6effe;
      color: #3b82f6;
    }

    .right-arrow {
      opacity: 0.8;
      width: 18px;
      height: 18px;
    }
  }

  &.green {

    .icon-wrapper {
      background-color: #22c55e;
    }

    .badge {
      background-color: #22c55e;
    }
  }

  &.pink {
    // border-color: #fcedf5;
    // background: #fcedf5;

    .icon-wrapper {
      background-color: #ec4899;
    }

    .title {
      // color: #a3004c;
    }

    .subtitle,
    .extra {
      // color: #e60076;
    }

    .badge {
      background-color: #f6339a;
    }

    .right-arrow {
      color: #f6349b;
      opacity: 0.7;
    }
  }

  &.skyblue {

    .icon-wrapper {
      background-color: #615fff;
    }

    .badge {
      background-color: #e0e7ff;
      color: #432dd7;
    }
  }

  &.orange {

    .icon-wrapper {
      background-color: #f97316;
    }

    .badge {
      background-color: #f97316;
    }
  }
}
</style>

<style lang="scss" scoped>
@media screen and (max-width: 768px) {
  .feature-card {
    min-width: 180px;
    padding: 10px;
    border-radius: 6px;
    border: 1px solid #e5e5e5;
    transition: none;

    &:hover {
      transform: none;
      box-shadow: none;
    }

    .header {
      margin-bottom: 8px;
    }

    .content {
      .title {
        font-weight: bold;
        color: #333;
        margin-bottom: 4px;
      }
    }

    .footer {
      margin-top: 6px;
    }
  }
}
</style>
