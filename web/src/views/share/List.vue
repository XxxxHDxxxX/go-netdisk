<template>
  <div class="page-share-list animated fadeIn">

    <div class="row">

      <div class="col-md-12" v-for="(share,index) in pager.data">
        <ShareBar
          :share="share"
          @deleteSuccess="refresh"
        />
      </div>

      <div class="col-md-12 mt20">
        <NbPager :pager="pager" :callback="refresh"></NbPager>
      </div>

    </div>
  </div>
</template>

<script>
  import NbFilter from '../../components/filter/NbFilter.vue'
  import NbPager from '../../components/NbPager.vue'
  import Pager from '../../model/base/Pager'
  import Share from "../../model/share/Share";
  import ShareBar from "./widget/ShareBar"
  import {MessageBox, Message} from "element-ui"

  export default {

    data() {
      return {
        pager: new Pager(Share, Pager.MAX_PAGE_SIZE),
        user: this.$store.state.user,
        selectedShares: []
      }
    },
    props: {

    },
    components: {
      NbFilter,
      NbPager,
      ShareBar
    },
    methods: {
      search() {
        this.pager.page = 1
        this.refresh()
      },
      refresh() {

        this.pager.httpFastPage()
      }
    },
    mounted() {
      this.pager.enableHistory()
      this.refresh()
    }
  }
</script>

<style lang="less" rel="stylesheet/less">
  .page-share-list {

  }
</style>
