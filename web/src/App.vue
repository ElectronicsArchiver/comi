<template>
  <n-config-provider :theme="lightTheme">
    <div class="app">
      <n-message-provider>
        <!-- 路由出口 路由匹配到的组件将渲染在这里 -->
        <router-view></router-view>
      </n-message-provider>
    </div>
  </n-config-provider>
</template>

<script>
// @ is an alias to /src
// import ScrollMode from "@/views/ScrollMode.vue";
// import FlipMode from "@/views/FlipMode.vue";
// import BookShelf from "@/views/BookShelf.vue";
import { useCookies } from "vue3-cookies";
import { defineComponent } from 'vue'
import { NMessageProvider, NConfigProvider, darkTheme, lightTheme } from 'naive-ui'

export default defineComponent({
  name: "ComigoHome", //默认为 default。如果 <router-view>设置了名称，则会渲染对应的路由配置中 components 下的相应组件。
  components: {
    NMessageProvider,
    NConfigProvider,//调整主题：https://www.naiveui.com/zh-CN/light/docs/customize-theme
  },
  setup() {
    const { cookies } = useCookies();
    return { cookies, darkTheme, lightTheme };
  },
  data() {
    return {
      selectTemplate: "",
      isAuthenticated: false,
    };
  },
  created() {
    // this.$store.dispatch("syncBookDataAction");
    this.$store.dispatch("syncSeverStatusDataAction");
    // this.$store.dispatch("syncBookShelfDataAction");
    this.selectTemplate = this.getDefaultTemplate;
  },
  beforeMount() {
    if (this.$store.state.server_status.ServerName != null) {
      document.title = this.$store.state.server_status.ServerName
    }
  },

  methods: {
    goToDashboard() {
      if (this.isAuthenticated) {
        this.$router.push('/dashboard')
      } else {
        this.$router.push('/login')
      }
    },
    OnSetTemplate(value) {
      localStorage.setItem("nowTemplate", value);
      this.selectTemplate = value;
    },
    getNumber: function (number) {
      this.page = number;
      console.log(number);
    },
  },
  //计算属性
  computed: {
    username() {
      // 我们很快就会看到 `params` 是什么
      return this.$route.params.username
    },
    book() {
      return this.$store.state.book;
    },
    setting() {
      return this.$store.state.setting;
    },
    getDefaultTemplate: function () {
      var localValue = localStorage.getItem("nowTemplate");
      if (localValue !== null) {
        return localValue;
      }
      //不管服务器设置，完全按照本地值来
      // if (this.setting.template) {
      //   localStorage.setItem("nowTemplate", this.setting.template)
      //   return this.setting.template;
      // }
      return "scroll";
    },
  },
});
</script>

<style>
#app {
  font-family: Avenir, Helvetica, Arial, sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  text-align: center;
  /* 整体颜色，做成用户设定？ */
  background-color: #f6f7eb;
  align-items: center;
}
/* 覆盖8px的浏览器默认值 */
* {
  /* 外边距，不指定的话，浏览器默认设置成8px */
  margin: 0px;
  /* 内边框 */
  padding: 0px;
}
</style>
