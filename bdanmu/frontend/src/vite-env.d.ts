/// <reference types="vite/client" />

interface Window {
  $message: import('naive-ui').MessageApi
  $dialog: import('naive-ui').DialogApi
  $loadingBar: import('naive-ui').LoadingBarApi
  $notification: import('naive-ui').NotificationApi
}
