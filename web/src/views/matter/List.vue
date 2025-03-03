<template>
  <div class="backyard-matter-list">
    <div class="row">

      <div class="col-md-8 mb10">
        <button class="btn btn-primary btn-sm mr5 mb5"
                v-if="accessControl.role !=='PROJECT_PROVIDER' && selectedMatters.length !== pager.data.length"
                @click.stop.prevent="checkAll">
          <i class="fa fa-check-square"></i>
          {{ $t("selectAll") }}
        </button>

        <button class="btn btn-primary btn-sm mr5 mb5"
                v-if="accessControl.role !=='PROJECT_PROVIDER' && pager.data.length && selectedMatters.length === pager.data.length"
                @click.stop.prevent="checkNone">
          <i class="fa fa-square-o"></i>
          {{ $t("cancel") }}
        </button>

        <button class="btn btn-primary btn-sm mr5 mb5"
                v-if="accessControl.role !=='PROJECT_PROVIDER' && selectedMatters.length"
                @click.stop.prevent="deleteBatch">
          <i class="fa fa-trash"></i>
          {{ $t("delete") }}
        </button>

        <button class="btn btn-primary btn-sm mr5 mb5"
                v-if="selectedMatters.length"
                @click.stop.prevent="downloadZip">
          <i class="fa fa-download"></i>
          {{ $t("matter.download") }}
        </button>

        <el-dialog
          :title="$t('matter.share')"
          :visible.sync="shareDialogVisible"
          :append-to-body="true">

          <SharePanel :matters="selectedMatters"
                      @close="shareDialogVisible = false"/>

        </el-dialog>


        <span class="btn btn-primary btn-sm btn-file mr5 mb5">
          <slot name="button">
            <i class="fa fa-cloud-upload"></i>
            <span> {{ $t('matter.upload') }} </span>
          </slot>
          <input ref="refFile" type="file" multiple="multiple" @change.prevent.stop="triggerUpload"/>
        </span>

        <span class="btn btn-primary btn-sm btn-file mr5 mb5">
          <slot name="button">
            <i class="fa fa-cloud-upload"></i>
            <span> {{ $t('matter.uploadDir') }} </span>
          </slot>
          <input ref="refDir" type="file" webkitdirectory @change.prevent.stop="triggerUploadDir"/>
        </span>

        <button class="btn btn-sm btn-primary mr5 mb5" @click.stop.prevent="createDirectory">
          <i class="fa fa-folder"></i>
          {{ $t("matter.create") }}
        </button>

      </div>

      <div class="col-md-4 mb10">
        <div class="input-group">
          <input type="text" class="form-control" v-model="searchText" @keyup.enter="searchFile"
                 :placeholder="$t('matter.searchFile')">
          <span class="input-group-btn">
            <button type="button" class="btn btn-primary" @click.prevent.stop="searchFile">
              <i class="fa fa-search"></i>
            </button>
          </span>
        </div>
      </div>

      <div class="col-md-12">


        <div v-for="m in uploadMatters">
          <UploadMatterPanel :matter="m"/>
        </div>

        <div v-if="director.createMode">
          <MatterPanel ref="newMatterPanel" @createDirectorySuccess="refresh()"
                       :matter="newMatter"
                       :director="director"/>
        </div>
        <div v-for="matter in pager.data">
          <MatterPanel :key="matter.uuid"
                       @goToDirectory="goToDirectory"
                       @deleteSuccess="refresh()"
                       :matter="matter"
                       :director="director"
                       @checkMatter="checkMatter"
                       @previewImage="previewImage"
          />
        </div>

        <div>
          <NbPager :pager="pager" :callback="refresh" :emptyHint="$t('matter.noContentYet')"/>
        </div>
      </div>


    </div>

  </div>
</template>
<script>
  import Vue from 'vue'
  import MatterPanel from './widget/MatterPanel'
  import UploadMatterPanel from './widget/UploadMatterPanel'
  import MoveBatchPanel from './widget/MoveBatchPanel'
  import SharePanel from './widget/ShareOperationPanel'
  import NbSlidePanel from '../../components/NbSlidePanel.vue'
  import NbExpanding from '../../components/NbExpanding.vue'
  import NbCheckbox from '../../components/NbCheckbox.vue'
  import NbFilter from '../../components/filter/NbFilter'
  import NbPager from '../../components/NbPager'
  import Matter from '../../model/matter/Matter'
  import Pager from '../../model/base/Pager'
  import Director from './widget/Director'
  import {Message, MessageBox} from 'element-ui'
  import {UserRole} from "../../model/user/UserRole";
  import {SortDirection} from "../../model/base/SortDirection";
  import {humanFileSize} from "../../common/filter/str";
  import Share from "../../model/share/Share";
  import {mapState} from 'vuex';

  export default {
    data() {
      return {
        //当前文件夹信息。
        matter: new Matter(),
        //准备新建的文件。
        newMatter: new Matter(),
        //准备上传的一系列文件
        uploadMatters: this.$store.state.uploadMatters,
        //当前选中的文件
        selectedMatters: [],
        //搜索的文字
        searchText: null,
        pager: new Pager(Matter, 20),

        //临时文件list，用于上传文件夹功能
        tempUploadList: [],
        uploadErrorLogs: [],

        //移动的目标文件夹
        targetMatterUuid: null,
        user: this.$store.state.user,
        preference: this.$store.state.preference,
        breadcrumbs: this.$store.state.breadcrumbs,
        director: new Director(),

        share: new Share(),
        //分享的弹框
        shareDialogVisible: false

      }
    },
    computed: {
      ...mapState(['accessControl'])
    },
    components: {
      MatterPanel,
      UploadMatterPanel,
      MoveBatchPanel,
      SharePanel,
      NbCheckbox,
      NbFilter,
      NbPager,
      NbSlidePanel,
      NbExpanding
    },
    methods: {
      reset() {
        this.pager.page = 1
        this.pager.resetFilter()
        this.pager.enableHistory()
      },
      search() {
        this.pager.page = 1
        this.refresh()
      },
      refresh() {

        let puuid = this.$route.query.puuid
        if (puuid) {
          this.pager.setFilterValue('puuid', puuid)
        } else {
          this.pager.setFilterValue('puuid', 'root')
        }

        //如果所有的排序都没有设置，那么默认以时间降序。
        this.pager.setFilterValue('orderCreateTime', SortDirection.DESC)
        this.pager.setFilterValue("orderDir", SortDirection.DESC)

        //如果没有设置用户的话，那么默认显示当前登录用户的资料
        // if (!this.pager.getFilterValue('userUuid')) {
        //   this.pager.setFilterValue('userUuid', this.user.uuid)
        // }

        this.pager.setFilterValue("name", this.searchText)

        //刷新面包屑
        this.refreshBreadcrumbs()

        this.pager.httpFastPage()
      },
      goToDirectory(uuid) {
        this.pager.setFilterValue('puuid', uuid)
        this.pager.page = 1
        let query = this.pager.getParams()

        //采用router去管理路由，否则浏览器的回退按钮出现意想不到的问题。
        this.$router.push({
          path: '/',
          query: query
        })

      },
      refreshBreadcrumbs() {

        let that = this

        //清空暂存区
        this.selectedMatters.splice(0, this.selectedMatters.length)

        let uuid = that.pager.getFilterValue('puuid')

        //根目录简单处理即可。
        if (!uuid || uuid === 'root') {

          this.matter.uuid = 'root'
          that.breadcrumbs.splice(0, that.breadcrumbs.length)
          that.breadcrumbs.push({
            title: 'matter.allFiles'
          })

        } else {

          this.matter.uuid = uuid
          this.matter.httpDetail(function () {

            let arr = []
            let cur = that.matter.parent
            while (cur) {
              arr.push(cur)
              cur = cur.parent
            }

            that.breadcrumbs.splice(0, that.breadcrumbs.length)
            let query = that.pager.getParams()
            query['puuid'] = 'root'
            //添加一个随机数，防止watch $route失败
            query['_t'] = new Date().getTime()
            that.breadcrumbs.push({
              title: 'matter.allFiles',
              path: '/',
              query: query
            })

            for (let i = arr.length - 1; i >= 0; i--) {
              let m = arr[i]
              let query = that.pager.getParams()
              query['puuid'] = m.uuid
              query['_t'] = new Date().getTime()
              that.breadcrumbs.push({
                title: m.name,
                displayDirect: true,
                path: '/',
                query: query
              })
            }
            //第一个文件
            that.breadcrumbs.push({
              title: that.matter.name,
              displayDirect: true,
            })
          })
        }
      },
      createDirectory() {
        let that = this
        that.newMatter.name = 'matter.allFiles'
        that.newMatter.dir = true
        that.newMatter.editMode = true
        that.newMatter.puuid = that.matter.uuid
        if (!that.newMatter.puuid) {
          that.newMatter.puuid = 'root'
        }


        //指定为当前选择的用户。
        //如果没有设置用户的话，那么默认显示当前登录用户的资料
        if (!that.pager.getFilterValue('userUuid')) {
          that.newMatter.userUuid = that.user.uuid
        } else {
          that.newMatter.userUuid = that.pager.getFilterValue('userUuid')
        }

        that.director.createMode = true

        setTimeout(function () {
          that.$refs.newMatterPanel.highLight()
        }, 100)
      },
      triggerUpload() {
        let that = this

        let domFiles = that.$refs['refFile'].files;
        if (!domFiles || !domFiles.length) {
          that.$message.error(that.$t('matter.allFiles'))
          return;
        }

        if (domFiles.length > 1000) {
          that.$message.error(that.$t('matter.exceed1000'))
          return;
        }

        this.launchUpload(domFiles);
      },

      triggerUploadDir() {
        let that = this

        let domFiles = that.$refs['refDir'].files;
        if (!domFiles || !domFiles.length) {
          that.$message.error(that.$t('matter.allFiles'))
          return
        }

        // 仅支持单层文件夹
        for(let i=0; i<domFiles.length; i++) {
          const file = domFiles[i];
          const paths = file.webkitRelativePath.split("/");
          if (paths.length > 2) {
            that.$message.error('暂不支持多级文件夹上传')
            return
          }
        }
        const file = domFiles[0];
        const paths = file.webkitRelativePath.split("/");
        const dirName = paths[0];
        const m = new Matter(this);
        m.name = dirName;
        m.puuid = this.matter.uuid;
        m.userUuid = this.user.uuid;

        m.httpCreateDirectory(
          (response) => {
            console.log(response)
            debugger
            if (response.data.result) {
              this.launchUpload(domFiles, response.data.data.uuid);
            } else {
              this.$message.error(response.data.msg)
            }
          },
          (msg) => {
            this.uploadErrorLogs.push([
              file.name,
              file.webkitRelativePath,
              msg,
            ]);
          }
        );

      },

      // TODO: 支持递归文件夹
      // debounce(func, wait) {
      //   debugger
      //   let timer = null;
      //   return (fileObj) => {
      //     const { file } = fileObj;
      //     this.tempUploadList.push(file);
      //     if (timer) {
      //       clearTimeout(timer);
      //     }
      //     timer = setTimeout(func, wait);
      //   };
      // },

      // triggerUploadDir() {
      //   debugger
      //   return this.debounce(async () => {
      //     await this.uploadDirectory();
      //     if (!this.uploadErrorLogs.length) {
      //       this.$message.success(this.$t("文件夹上传成功"));
      //     } else {
      //       // 上传错误弹出提示框
      //       console.log(this.uploadErrorLogs);
      //       this.$message.error(this.uploadErrorLogs.join('\n'));
      //     }
      //     this.tempUploadList = [];
      //     this.uploadErrorLogs = [];
      //     this.refresh();
      //   }, 0);
      // },

      // async uploadDirectory() {
      //   const dirPathUuidMap = {}; // 存储已经创建好的文件夹map
      //   for (let i = 0; i < this.tempUploadList.length; i++) {
      //     const file = this.tempUploadList[i];
      //     const paths = file.webkitRelativePath.split("/");
      //     const pPaths = paths.slice(0, paths.length - 1);
      //     const pPathStr = pPaths.join("/");
      //     if (!dirPathUuidMap[pPathStr]) {
      //       // 递归创建其父文件夹
      //       for (let j = 1; j <= pPaths.length; j++) {
      //         const midPaths = pPaths.slice(0, j);
      //         const midPathStr = midPaths.join("/");
      //         if (!dirPathUuidMap[midPathStr]) {
      //           const m = new Matter(this);
      //           m.name = midPaths[midPaths.length - 1];
      //           m.puuid =
      //             j > 1
      //               ? dirPathUuidMap[midPaths.slice(0, j - 1).join("/")]
      //               : this.matter.uuid;
      //           m.userUuid = this.user.uuid;
      //
      //           await m.httpCreateDirectory(
      //             () => {
      //               dirPathUuidMap[midPathStr] = m.uuid;
      //             },
      //             (msg) => {
      //               this.uploadErrorLogs.push([
      //                 file.name,
      //                 file.webkitRelativePath,
      //                 msg,
      //               ]);
      //             }
      //           );
      //         }
      //       }
      //     }
      //
      //     if (dirPathUuidMap[pPathStr]) {
      //       this.launchUpload(file, dirPathUuidMap[pPathStr], (msg) => {
      //         this.uploadErrorLogs.push([
      //           file.name,
      //           file.webkitRelativePath,
      //           `${msg}`,
      //         ]);
      //       });
      //     }
      //
      //   }
      // },

      launchUpload(domFiles, puuid = this.matter.uuid) {
          let that = this;
          for (let i = 0; i < domFiles.length; i++) {
              let domFile = domFiles[i];
              let m = new Matter()
              m.dir = false
              m.puuid = puuid

              //指定为当前选择的用户。
              //如果没有设置用户的话，那么默认显示当前登录用户的资料
              if (!that.pager.getFilterValue('userUuid')) {
                  m.userUuid = that.user.uuid
              } else {
                  m.userUuid = that.pager.getFilterValue('userUuid')
              }

              //判断文件大小。
              if (that.user.sizeLimit >= 0) {
                  if (domFile.size > that.user.sizeLimit) {
                      that.$message.error(that.$t('matter.sizeExceedLimit', humanFileSize(domFile.size), humanFileSize(that.user.sizeLimit)))
                      continue
                  }
              }

              m.file = domFile

              m.httpUpload(function () {
                  that.$store.state.uploadListInstance.refresh()
              })

              that.uploadMatters.push(m)
          }
      },

      previewImage(matter) {
        let that = this;

        //从matter开始预览图片
        let imageArray = []
        let startIndex = -1;
        this.pager.data.forEach(function (item, index) {
          if (item.isImage()) {
            imageArray.push(item.getPreviewUrl())
            if (item.uuid === matter.uuid) {
              startIndex = imageArray.length - 1
            }
          }
        })

        that.$photoSwipePlugin.showPhotos(imageArray, startIndex)

      },
      //全选
      checkAll() {
        this.pager.data.forEach(function (i, index) {
          i.check = true
        })
        this.checkMatter()
      },
      //取消全选
      checkNone() {
        this.pager.data.forEach(function (i, index) {
          i.check = false
        })
        this.checkMatter()
      },
      //选择文件时放入暂存区等待操作
      checkMatter(matter) {
        let that = this
        //统计所有的勾选
        this.selectedMatters.splice(0, this.selectedMatters.length)
        this.pager.data.forEach(function (matter, index) {
          if (matter.check) {
            that.selectedMatters.push(matter)
          }
        })

      },

      //批量下载
      downloadZip() {
        let that = this
        let uuids = []

        that.selectedMatters.forEach(function (item, index) {
          uuids.push(item.uuid)
        })

        that.matter.downloadZip(uuids.toString())
      },
      //批量删除
      deleteBatch() {
        let that = this
        MessageBox.confirm(that.$t("actionCanNotRevertConfirm"), that.$t("prompt"), {
          confirmButtonText: that.$t("confirm"),
          cancelButtonText: that.$t("cancel"),
          type: 'warning',
          callback: function (action, instance) {
            if (action === 'confirm') {
              let uuids = ""
              that.selectedMatters.forEach(function (item, index) {
                if (index === 0) {
                  uuids = item.uuid
                } else {
                  uuids = uuids + "," + item.uuid
                }
              })
              that.matter.httpDeleteBatch(uuids, function (response) {
                Message.success(that.$t("operationSuccess"))
                that.refresh()
              })
            }

          }
        })
      },

      //批量移动
      moveBatch(createElement) {
        let that = this

        let dom = createElement(MoveBatchPanel, {
          props: {
            version: (new Date()).getTime(),
            userUuid: that.selectedMatters[0].userUuid,
            callback: function (matter) {
              if (matter.uuid) {
                that.targetMatterUuid = matter.uuid
              } else {
                that.targetMatterUuid = "root"
              }
            }
          }
        })

        MessageBox({
          title: '移动到',
          message: dom,
          customClass: 'wp50',
          confirmButtonText: that.$t("confirm"),
          showCancelButton: true,
          cancelButtonText: '关闭',
          callback: (action, instance) => {
            if (action === 'confirm') {
              let uuids = ""
              that.selectedMatters.forEach(function (item, index) {
                if (index === 0) {
                  uuids = item.uuid
                } else {
                  uuids = uuids + "," + item.uuid
                }
              })

              that.matter.httpMove(uuids, that.targetMatterUuid, function (response) {
                Message.success('移动成功！')
                that.refresh()
              })
            }
          }
        })
      },
      searchFile() {

        let that = this;
        if (that.searchText) {

          //刷新面包屑
          that.refreshBreadcrumbs()
          that.pager.resetFilter()
          that.pager.setFilterValue('puuid', null)
          that.pager.setFilterValue("orderCreateTime", SortDirection.DESC)
          that.pager.setFilterValue("name", that.searchText)

          that.pager.httpFastPage()


        } else {

          that.refresh()
        }

      }
    },
    watch: {
      '$route'(newVal, oldVal) {

        this.refresh()

      },
      'searchText'(newVal, oldVal) {
        if (oldVal && !newVal) {
          this.refresh()
        }
      }

    },
    created() {
      /*初始化inputSelection*/
      if (this.user.role === UserRole.ADMINISTRATOR) {
        this.pager.getFilter('userUuid').visible = true
      } else {
        this.pager.setFilterValue('userUuid', this.user.uuid)
      }
      Vue.prototype.dropUploadFile = this.launchUpload
    },
    mounted() {
      let that = this
      this.pager.enableHistory()
      //更新vuex中List实例，主要解决大文件上传持有的功能。
      this.$store.state.uploadListInstance = this

      this.refresh()
    },
    destroyed() {
       Vue.prototype.dropUploadFile = null;
    }
  }
</script>
<style lang="less" rel="stylesheet/less">
  @import "../../assets/css/global/variables";

  .backyard-matter-list {

  }
</style>
