<template>
  <div class="custom-table-container">
    <div class="custom-table-wrapper" :style="{ overflowX: isMobile ? 'auto' : 'hidden' }">
      <table class="custom-table" :class="{ 'has-selection': showSelection }" :style="tableStyle">
        <!-- 表头 -->
        <thead :style="{ background: theadBgColor }">
          <tr>
            <th v-if="showSelection" class="selection-header">
              <input type="checkbox" v-model="allSelected" @change="toggleAllSelection" />
            </th>
            <th v-for="(column, index) in columns" :key="index" :style="getColumnStyle(column)">
              {{ gettext(column.label) }}
            </th>
          </tr>
        </thead>

        <!-- 表格内容 -->
        <tbody>
          <tr v-for="(row, rowIndex) in paginatedData" :key="rowIndex"
            :class="{ 'last-row': rowIndex === paginatedData.length - 1 }">
            <td v-if="showSelection" class="selection-cell">
              <input type="checkbox" :checked="selectedRows.includes(row[rowKey])"
                @change="(e) => updateSelection(row[rowKey], e.target.checked)" />
            </td>
            <td v-for="(column, colIndex) in columns" :key="colIndex" :style="{ textAlign: column.align || 'center' }">
              <template v-if="column.slot">
                <slot :name="column.slot" :row="row" :index="rowIndex"></slot>
              </template>
              <template v-else>
                {{ row[column.prop] }}
              </template>
            </td>
          </tr>

          <!-- 空数据提示 -->
          <tr v-if="paginatedData.length === 0" class="empty-row">
            <td :colspan="showSelection ? columns.length + 1 : columns.length">
              {{ emptyText }}
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- 分页 -->
    <div v-if="showPagination" class="pagination-wrapper">
      <div class="pagination-info">
        {{ gettext('显示') }} {{ startItem }} {{ gettext('到') }} {{ endItem }} {{ gettext('条') }}，{{ gettext('共') }} {{
          total
        }}
        {{ gettext('条') }}
      </div>
      <div class="pagination-controls">
        <button :disabled="currentPage === 1" @click="changePage(currentPage - 1)">
          {{ gettext('上一页') }}
        </button>
        <button v-for="page in visiblePages" :key="page" :class="{ active: page === currentPage }"
          @click="changePage(page)">
          {{ page }}
        </button>
        <button :disabled="currentPage === totalPages" @click="changePage(currentPage + 1)">
          {{ gettext('下一页') }}
        </button>
      </div>
    </div>
  </div>
</template>

<script>
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { useGettext } from '/@/plugins/i18n'

export default {
  name: 'CustomTable',
  props: {
    data: {
      type: Array,
      default: () => []
    },
    columns: {
      type: Array,
      required: true,
      validator: (value) => {
        return value.every(
          (column) => column.label && column.prop
        )
      }
    },
    showSelection: {
      type: Boolean,
      default: false
    },
    rowKey: {
      type: String,
      default: 'id'
    },
    showPagination: {
      type: Boolean,
      default: true
    },
    pageSize: {
      type: Number,
      default: 10
    },
    currentPage: {
      type: Number,
      default: 1
    },
    emptyText: {
      type: String,
      default: '暂无数据'
    },
    maxPagerCount: {
      type: Number,
      default: 5
    },
    theadBgColor: {
      type: String,
      default: '#F8F8F8'
    },
  },
  emits: ['selection-change', 'update:currentPage', 'page-change'],
  setup(props, { emit }) {
    const { $gettext: gettext } = useGettext()
    const selectedRows = ref([])
    const allSelected = ref(false)
    const isMobile = ref(false)
    const tableWidth = ref(null)

    // 计算属性
    const total = computed(() => props.data.length)
    const totalPages = computed(() => Math.ceil(total.value / props.pageSize))
    const paginatedData = computed(() => {
      if (!props.showPagination) return props.data
      const start = (props.currentPage - 1) * props.pageSize
      const end = start + props.pageSize
      return props.data.slice(start, end)
    })
    const startItem = computed(() => {
      return (props.currentPage - 1) * props.pageSize + 1
    })
    const endItem = computed(() => {
      const end = props.currentPage * props.pageSize
      return end > total.value ? total.value : end
    })
    const tableStyle = computed(() => {
      const minWidth = props.columns.reduce((sum, col) => {
        return sum + (parseInt(col.width) || 50)
      }, props.showSelection ? 50 : 0)
      return {
        minWidth: `${minWidth}px`
      }
    })

    // 分页按钮显示逻辑
    const visiblePages = computed(() => {
      const pages = []
      const half = Math.floor(props.maxPagerCount / 2)
      let start = props.currentPage - half
      let end = props.currentPage + half

      if (start < 1) {
        start = 1
        end = Math.min(props.maxPagerCount, totalPages.value)
      }

      if (end > totalPages.value) {
        end = totalPages.value
        start = Math.max(1, end - props.maxPagerCount + 1)
      }

      for (let i = start; i <= end; i++) {
        pages.push(i)
      }

      return pages
    })

    // 方法
    const getColumnStyle = (column) => {
      return {
        width: column.width ? `${Math.max(50, parseInt(column.width))}px` : 'auto',
        minWidth: '50px',
        textAlign: column.align || 'center'
      }
    }

    const toggleAllSelection = () => {
      if (allSelected.value) {
        selectedRows.value = [...paginatedData.value.map(item => item[props.rowKey])];
      } else {
        selectedRows.value = [];
      }
      emitSelectionChange();
    }

    const handleSelectionChange = () => {
      allSelected.value = selectedRows.value.length === paginatedData.value.length &&
        paginatedData.value.length > 0;
      emitSelectionChange();
    }

    const updateSelection = (rowId, isChecked) => {
      if (isChecked) {
        if (!selectedRows.value.includes(rowId)) {
          selectedRows.value.push(rowId);
        }
      } else {
        selectedRows.value = selectedRows.value.filter(id => id !== rowId);
      }
      handleSelectionChange();
    }

    const emitSelectionChange = () => {
      const selectedItems = props.data.filter(item =>
        selectedRows.value.includes(item[props.rowKey])
      )
      emit('selection-change', selectedItems)
    }

    const changePage = (page) => {
      if (page < 1 || page > totalPages.value) return
      emit('update:currentPage', page)
      emit('page-change', page)
    }

    const checkMobile = () => {
      isMobile.value = window.innerWidth <= 768
    }

    // 监听
    watch(
      () => props.data,
      () => {
        selectedRows.value = [];
        allSelected.value = false;
      },
      { deep: true }
    );

    // 生命周期
    onMounted(() => {
      checkMobile()
      window.addEventListener('resize', checkMobile)
    })

    onUnmounted(() => {
      window.removeEventListener('resize', checkMobile)
    })

    return {
      selectedRows,
      allSelected,
      isMobile,
      total,
      totalPages,
      paginatedData,
      startItem,
      endItem,
      visiblePages,
      tableStyle,
      gettext,
      getColumnStyle,
      toggleAllSelection,
      handleSelectionChange,
      changePage,
      updateSelection
    }
  }
}
</script>

<style>
tr>td,
tr>th,
.tr>.td,
.tr>.th,
.cbi-section-table-row::before,
#cbi-wireless>#wifi_assoclist_table>.tr:nth-child(2) {
  border: none;
}
</style>
<style scoped lang="scss">
.custom-table-container {
  width: 100%;
  font-size: 14px;
  color: var(--flow-span-color);

  input {
    margin: 0;
  }

  .custom-table-wrapper {
    width: 100%;
    overflow-x: auto;
    -webkit-overflow-scrolling: touch;
  }

  .custom-table {
    width: 100%;
    border-collapse: collapse;
    table-layout: auto;

    thead {
      border-radius: 8px !important;
      background: #F8F8F8;

      tr {
        th {
          color: var(--flow-span-color) !important;
          font-weight: 500 !important;
          border: none !important;
          padding: 18px 8px !important;
          white-space: nowrap;
        }
      }
    }

    tbody {
      tr {
        background: transparent;
        border-bottom: 1px solid #f8f8f8 !important;

        &:last-child {
          border-bottom: none !important;
        }

        td {
          padding: 24px 8px !important;
          white-space: nowrap;
        }

        &:hover {
          td {
            background-color: rgba(0, 0, 0, 0.02) !important;
          }
        }

        &.empty-row {
          td {
            text-align: center !important;
            padding: 30px 0 !important;
            color: rgba(201, 141, 141, 0.4) !important;
            border-bottom: none !important;
          }
        }
      }
    }

    .selection-header,
    .selection-cell {
      width: 50px !important;
      min-width: 50px !important;
      text-align: center !important;
    }
  }

  .pagination-wrapper {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-top: 16px;
    padding: 0 8px;

    .pagination-info {
      color: rgba(0, 0, 0, 0.6);
      font-size: 13px;
    }

    .pagination-controls {
      button {
        min-width: 32px;
        height: 32px;
        margin: 0 4px;
        padding: 0 8px;
        background: #fff;
        color: rgba(0, 0, 0, 0.65);
        cursor: pointer;
        transition: all 0.3s;

        &:hover:not(:disabled) {
          color: #1890ff;
          border-color: #1890ff;
        }

        &:disabled {
          color: rgba(0, 0, 0, 0.25);
          background-color: #f5f5f5;
          border-color: #d9d9d9;
          cursor: not-allowed;
        }

        &.active {
          color: #fff;
          background-color: #1890ff;
          border-color: #1890ff;
        }
      }
    }
  }

  @media (max-width: 768px) {
    .custom-table {
      thead {
        tr {
          th {
            padding: 8px 4px !important;
            font-size: 13px !important;
          }
        }
      }

      tbody {
        tr {
          td {
            padding: 12px 4px !important;
            font-size: 13px !important;
          }
        }
      }
    }

    .pagination-wrapper {
      flex-direction: column;
      align-items: flex-start;
      gap: 12px;

      .pagination-controls {
        display: flex;
        flex-wrap: wrap;
        gap: 4px;

        button {
          min-width: 28px;
          height: 28px;
          margin: 0;
          padding: 0 6px;
          font-size: 13px;
        }
      }
    }
  }
}
</style>