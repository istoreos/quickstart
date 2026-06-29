<template>
  <div class="system-update-banner" v-if="isShow">
    <div class="icon_box">
      <downloadIcon color="#ffffff" class="download-svg" />
    </div>
    <div class="tip">
      <div class="tip_left">
        <div class="title">{{ $gettext("系统更新可用") }}</div>
        <div class="info">{{ $gettext("检测到新版本") }}，{{ $gettext("包含安全修复和性能改进") }}</div>
      </div>
      <div class="tip_right">
        <button class="btn-primary" @click="onRenew">{{ $gettext("立即更新") }}</button>
        <button class="btn-outline" @click="isShow = false">{{ $gettext("稍后提醒") }}</button>
        <closeIcon class="close-icon" color="#50a2ff" @click="isShow = false" />
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ref } from "vue"
import downloadIcon from "/@/components/svg/download1.vue"
import closeIcon from "/@/components/svg/close.vue"
import { useGettext } from '/@/plugins/i18n'
const { $gettext } = useGettext()
const isShow = ref(true)
const onRenew = () => {
    window.location.href = '/cgi-bin/luci/admin/system/ota'
}
</script>

<style lang="scss" scoped>
.system-update-banner {
  padding: 20px 16px;
  background: var(--SystemUpdateBanner-bg-color);
  border-radius: 8px;
  border: 1px solid var(--SystemUpdateBanner-border-color);
  display: flex;
  margin-bottom: 30px;

  .icon_box {
    width: 35px;
    height: 35px;
    background: #2b7fff;
    border-radius: 50%;
    display: flex;
    justify-content: center;
    align-items: center;
    margin-right: 16px;
  }

  .download-svg {
    width: 18px;
    height: 18px;
  }

  .tip {
    display: flex;
    justify-content: space-between;
    flex: 1;

    .tip_left {
      display: flex;
      flex-direction: column;
      justify-content: center;

      .title {
        color: #2b7fff;
        margin-bottom: 8px;
      }

      .info {
        color: #1447e6;
        font-size: 13px;
      }
    }

    .tip_right {
      display: flex;
      align-items: center;

      .close-icon {
        width: 16px;
        height: 16px;
        margin-left: 12px;
        cursor: pointer;
      }
    }
  }
}

.btn-primary {
  background-color: #2d8cf0;
  color: white;
  border: none;
  padding: 5px 16px;
  border-radius: 8px;
  font-size: 14px;
  cursor: pointer;
  transition: background 0.2s, transform 0.1s;
  margin-left: 6px;
}

.btn-primary:hover {
  background-color: #1b6edc;
}

.btn-primary:active {
  transform: scale(0.95);
}

.btn-outline {
  background-color: white;
  color: #2d8cf0;
  border: 1px solid #2d8cf0;
  padding: 5px 16px;
  border-radius: 8px;
  font-size: 14px;
  cursor: pointer;
  transition: background 0.2s, transform 0.1s;
  margin-left: 6px;
}

.btn-outline:hover {
  background-color: #f0f8ff;
}

.btn-outline:active {
  transform: scale(0.95);
}
</style>

<style lang="scss" scoped>
@media screen and (max-width: 768px) {
  .system-update-banner {
    position: relative;
    padding: 10px 8px;
    border-radius: 6px;
    margin-bottom: 10px;

    .icon_box {
      width: 26px;
      height: 26px;
      margin-right: 8px;
    }

    .download-svg {
      width: 16px;
      height: 16px;
    }

    .tip {
      flex-direction: column;
      .tip_right {
        display: flex;
        align-items: center;
        justify-content: space-around;
        padding-right: 34px;
        margin-top: 6px;

        .close-icon {
          width: 14px;
          height: 14px;
        }
      }
    }
  }

  .btn-primary {
    padding: 5px 32px;
    margin-left: 0;
    margin-bottom: 0;
    border-radius: 6px;
  }

  .btn-outline {
    padding: 4px 30px;
    margin-left: 6px;
    margin-bottom: 0;
    border-radius: 6px;
  }

  .close-icon {
    position: absolute;
    top: 10px;
    right: 10px;
  }
}
</style>
