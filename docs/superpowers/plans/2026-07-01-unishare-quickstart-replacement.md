# UniShare Quickstart Replacement Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the old NAS Samba/WebDAV quick setup with UniShare-backed quick setup so one user and one share can enable both Samba and WebDAV.

**Architecture:** The quickstart frontend will keep the NAS wizard entry but submit to the existing Share APIs that write `/etc/config/unishare`. Backend status will read UniShare as the source of truth, while legacy `/nas/samba/create`, `/nas/webdav/create`, and `/nas/webdav/status/` quick-create endpoints are removed from the route surface.

**Tech Stack:** Go 1.23, `github.com/digineo/go-uci` replaced by `github.com/linkease/go-uci`, `httprouter`, Vue 3, TypeScript, Vite.

---

## File Structure

- Modify `backend/modules/nas/routes.go`: remove NAS quick-create route methods and route registrations for Samba/WebDAV.
- Modify `backend/service/nas_backend.go`: remove `ServiceBackend` methods that delegate to the deleted NAS quick-create handlers.
- Modify `backend/service/nas.go`: remove legacy `NasServiceSambaCreate`, `NasServiceWebdavCreate`, `NasServiceWebdavStatus`, `NasServiceSambaStatus`, and unused root-template helper.
- Modify `backend/service/nas_service_config_usecase.go`, `backend/service/nas_service_config_store.go`, `backend/service/nas_service_config_types.go`: delete legacy serviceconfig adapter and store code that writes `samba4`/`gowebdav`.
- Modify or delete `backend/modules/nas/serviceconfig/*`: remove the package if no callers remain.
- Modify `backend/modules/nas/routes_test.go`: remove legacy route tests and add coverage that the old routes are not registered.
- Modify `backend/service/nas_service_config_usecase_test.go`, `backend/service/nas_service_config_store_test.go`: delete these tests with the deleted implementation.
- Modify `backend/modules/share/user/service.go`, `backend/service/share_user_usecase.go`: ensure quick setup can create or update a UniShare user through a clear backend operation.
- Modify `backend/modules/share/service/service.go`, `backend/service/share_service_usecase.go`: ensure quick setup can create or update one UniShare share with `samba` and/or `webdav` protocol flags and an RW user.
- Modify `backend/service/nas_service_config_usecase.go` only if a compatibility facade is kept for status; otherwise delete.
- Modify `backend/service/nas.go` and `backend/service/nas_service_config_store.go` status readers so storage service status is derived from UniShare shares plus LinkEase status.
- Modify `web/src/request/request.ts`: add Share user/service request helpers and remove old `Nas.Samba.Create`, `Nas.Webdav.Create`, `Nas.Webdav.Status` helpers.
- Modify `web/src/types/response.d.ts` and `web/src/types/nas.d.ts`: add active Share types and replace `NasCreateSamba`/`NasCreateWebdav` with a unified quick share type.
- Replace `web/src/components/action-nas/samba/index.vue` and `web/src/components/action-nas/webdav/index.vue` with one unified UniShare quick setup component, or create `web/src/components/action-nas/unishare/index.vue` and remove the old component imports.
- Modify `web/src/components/action-nas/index.vue`: default to unified Samba + WebDAV quick setup, check/install `luci-app-unishare`, `unishare`, `samba4-server`, and `webdav2` as needed, then open the unified form.
- Modify `web/src/pages/index/Storage/index.vue`, `web/src/pages/index/Storage/samba.vue`, `web/src/pages/index/Storage/webdav.vue`: display UniShare-backed status and link to `/cgi-bin/luci/admin/nas/unishare`.

---

### Task 1: Backend Share Service Supports Idempotent Quick Setup

**Files:**
- Modify: `backend/modules/share/user/service.go`
- Modify: `backend/service/share_user_usecase.go`
- Modify: `backend/modules/share/service/service.go`
- Modify: `backend/service/share_service_usecase.go`
- Test: `backend/modules/share/user/service_test.go`
- Test: `backend/modules/share/service/service_test.go`
- Test: `backend/service/share_user_usecase_test.go`
- Test: `backend/service/share_service_usecase_test.go`

- [ ] **Step 1: Write failing tests for idempotent user creation**

Add a test to `backend/modules/share/user/service_test.go`:

```go
func TestServiceEnsureCreatesOrUpdatesUser(t *testing.T) {
	store := &fakeStore{}
	svc := NewService(store)

	if err := svc.Ensure(context.Background(), CreateInput{UserName: "media", Password: "pw1"}); err != nil {
		t.Fatalf("Ensure create: %v", err)
	}
	if len(store.created) != 1 || store.created[0].UserName != "media" || store.created[0].Password != "pw1" {
		t.Fatalf("created = %#v", store.created)
	}

	store.users = []*models.ShareUserInfo{{UserName: "media", Password: "old"}}
	if err := svc.Ensure(context.Background(), CreateInput{UserName: "media", Password: "pw2"}); err != nil {
		t.Fatalf("Ensure update: %v", err)
	}
	if len(store.updated) != 1 || store.updated[0].UserName != "media" || store.updated[0].Password != "pw2" {
		t.Fatalf("updated = %#v", store.updated)
	}
}
```

Update the test fake store with:

```go
created []CreateInput
updated []UpdateInput

func (store *fakeStore) CreateUser(ctx context.Context, index int, input CreateInput) error {
	store.created = append(store.created, input)
	return store.createErr
}

func (store *fakeStore) UpdateUser(ctx context.Context, index int, password string) error {
	userName := ""
	if index >= 0 && index < len(store.users) && store.users[index] != nil {
		userName = store.users[index].UserName
	}
	store.updated = append(store.updated, UpdateInput{UserName: userName, Password: password})
	return store.updateErr
}
```

- [ ] **Step 2: Run the user service test and verify it fails**

Run:

```bash
rtk go test ./modules/share/user -run TestServiceEnsureCreatesOrUpdatesUser -count=1
```

Expected: FAIL because `Service.Ensure` is not defined.

- [ ] **Step 3: Implement `Service.Ensure`**

Add to `backend/modules/share/user/service.go`:

```go
func (svc *Service) Ensure(ctx context.Context, input CreateInput) error {
	if err := validateCreateInput(input); err != nil {
		return err
	}

	users, err := svc.store.ReadUsers(ctx)
	if err != nil {
		return err
	}
	for idx, user := range users {
		if user != nil && input.UserName == user.UserName {
			return svc.store.UpdateUser(ctx, idx, input.Password)
		}
	}
	return svc.store.CreateUser(ctx, len(users), input)
}
```

- [ ] **Step 4: Write failing tests for idempotent share creation**

Add a test to `backend/modules/share/service/service_test.go`:

```go
func TestServiceEnsureCreatesOrUpdatesShare(t *testing.T) {
	input := CreateInput{
		Name:   "media",
		Path:   "/mnt/media",
		Samba:  true,
		Webdav: true,
		Users: []*models.ShareServiceUserPermission{{UserName: "media", Rw: true}},
	}
	store := &fakeStore{}
	svc := NewService(store)

	if err := svc.Ensure(context.Background(), input); err != nil {
		t.Fatalf("Ensure create: %v", err)
	}
	if len(store.created) != 1 || !store.created[0].Samba || !store.created[0].Webdav {
		t.Fatalf("created = %#v", store.created)
	}

	store.shares = []*ShareRecord{{Name: "media", Path: "/mnt/old", Proto: []string{"samba"}}}
	if err := svc.Ensure(context.Background(), input); err != nil {
		t.Fatalf("Ensure update: %v", err)
	}
	if len(store.updated) != 1 || store.updated[0].Path != "/mnt/media" || !store.updated[0].Webdav {
		t.Fatalf("updated = %#v", store.updated)
	}
}
```

Update the test fake store with:

```go
created []CreateInput
updated []UpdateInput

func (store *fakeStore) CreateShare(ctx context.Context, index int, input CreateInput) error {
	store.created = append(store.created, input)
	return store.createErr
}

func (store *fakeStore) UpdateShare(ctx context.Context, index int, input UpdateInput) error {
	store.updated = append(store.updated, input)
	return store.updateErr
}
```

- [ ] **Step 5: Run the share service test and verify it fails**

Run:

```bash
rtk go test ./modules/share/service -run TestServiceEnsureCreatesOrUpdatesShare -count=1
```

Expected: FAIL because `Service.Ensure` is not defined.

- [ ] **Step 6: Implement `Service.Ensure` for shares**

Add to `backend/modules/share/service/service.go`:

```go
func (svc *Service) Ensure(ctx context.Context, input CreateInput) error {
	if err := validateShareInput(input.Name, input.Path); err != nil {
		return err
	}

	shares, _, err := svc.store.ReadConfig(ctx)
	if err != nil {
		return err
	}
	index := findShareIndex(shares, input.Name)
	if index >= 0 {
		return svc.store.UpdateShare(ctx, index, UpdateInput(input))
	}
	return svc.store.CreateShare(ctx, len(shares), input)
}
```

- [ ] **Step 7: Expose typed ensure operations in service layer**

Add methods to `backend/service/share_user_usecase.go`:

```go
func ShareUserEnsureTyped(ctx context.Context, req models.ShareUserCreateRequest) (*models.SDKNormalResponse, error) {
	if err := newShareUserService().Ensure(ctx, shareuser.CreateInput{
		UserName: req.UserName,
		Password: req.Password,
	}); err != nil {
		return nil, err
	}
	success := models.ResponseSuccess(int64(0))
	return &models.SDKNormalResponse{Success: &success}, nil
}
```

Add methods to `backend/service/share_service_usecase.go`:

```go
func ShareServiceEnsureTyped(ctx context.Context, req models.ShareServiceCreateRequest) (*models.SDKNormalResponse, error) {
	if err := newShareService().Ensure(ctx, shareservice.CreateInput{
		Name:   req.Name,
		Path:   req.Path,
		Samba:  req.Samba,
		Webdav: req.Webdav,
		Users:  req.Users,
	}); err != nil {
		return nil, err
	}
	success := models.ResponseSuccess(int64(0))
	return &models.SDKNormalResponse{Success: &success}, nil
}
```

Update the local facade interfaces in those files to include `Ensure(ctx context.Context, input ...) error`.

- [ ] **Step 8: Run backend Share tests**

Run:

```bash
rtk go test ./modules/share/... ./service -run 'Share(User|Service)|TestServiceEnsure' -count=1
```

Expected: PASS.

- [ ] **Step 9: Commit**

```bash
git add backend/modules/share backend/service/share_user_usecase.go backend/service/share_service_usecase.go
git commit -m "feat: add unishare ensure operations"
```

---

### Task 2: Backend NAS Status Reads UniShare

**Files:**
- Modify: `backend/service/nas.go`
- Modify: `backend/service/nas_service_config_usecase.go`
- Modify: `backend/service/nas_service_config_store.go`
- Test: `backend/service/nas_service_config_store_test.go`
- Test: `backend/service/nas_service_config_usecase_test.go`

- [ ] **Step 1: Write failing status tests for UniShare-backed NAS status**

In `backend/service/nas_service_config_store_test.go`, replace Samba/GowebDAV status expectations with UniShare records. Add:

```go
func TestDefaultNasServiceStatusReaderReadsUniShareServices(t *testing.T) {
	originalLoad := loadNasServiceConfig
	originalSections := getNasServiceSections
	originalGet := getNasServiceLast
	defer func() {
		loadNasServiceConfig = originalLoad
		getNasServiceSections = originalSections
		getNasServiceLast = originalGet
	}()

	loads := []string{}
	loadNasServiceConfig = func(config string) { loads = append(loads, config) }
	getNasServiceSections = func(config string, sectionType string) ([]string, bool) {
		if config == "unishare" && sectionType == "share" {
			return []string{"@share[0]", "@share[1]"}, true
		}
		return nil, false
	}
	getNasServiceLast = func(config string, section string, option string) (string, bool) {
		values := map[string]string{
			"@share[0].name": "media",
			"@share[0].path": "/mnt/media",
			"@share[1].name": "docs",
			"@share[1].path": "/mnt/docs",
			"@global[0].webdav_port": "8080",
		}
		value, ok := values[section+"."+option]
		return value, ok
	}

	reader := newDefaultNasServiceStatusReader()
	sambas := reader.ReadSambaShares()
	if len(sambas) != 2 || sambas[0].ShareName != "media" || sambas[1].Path != "/mnt/docs" {
		t.Fatalf("sambas = %#v", sambas)
	}
	webdav := reader.ReadWebdavInfo()
	if webdav.Port != "8080" || webdav.Path != "/mnt/media" {
		t.Fatalf("webdav = %#v", webdav)
	}
	if len(loads) == 0 || loads[0] != "unishare" {
		t.Fatalf("loads = %#v", loads)
	}
}
```

- [ ] **Step 2: Run the test and verify it fails**

Run:

```bash
rtk go test ./service -run TestDefaultNasServiceStatusReaderReadsUniShareServices -count=1
```

Expected: FAIL because the reader still loads `samba4` and `gowebdav`.

- [ ] **Step 3: Implement UniShare status reader**

Update `backend/service/nas_service_config_store.go` so `ReadSambaShares`, `ReadWebdavPort`, and `ReadWebdavInfo` load `unishare`.

Use this behavior:

```go
func (defaultNasServiceStatusReader) ReadSambaShares() []*models.NasServiceSambaInfo {
	loadNasServiceConfig("unishare")
	shares, _ := getNasServiceSections("unishare", "share")
	items := make([]*models.NasServiceSambaInfo, 0, len(shares))
	for _, section := range shares {
		item := &models.NasServiceSambaInfo{}
		if value, ok := getNasServiceLast("unishare", section, "name"); ok {
			item.ShareName = value
		}
		if value, ok := getNasServiceLast("unishare", section, "path"); ok {
			item.Path = value
		}
		if item.ShareName != "" || item.Path != "" {
			items = append(items, item)
		}
	}
	return items
}
```

For WebDAV status, use the first UniShare share path as `Path`, `unishare.@global[0].webdav_port` as `Port`, and leave `Username`/`Password` empty because users are now shared and should be listed through Share user APIs.

- [ ] **Step 4: Run status tests**

Run:

```bash
rtk go test ./service -run 'NasServiceStatus|DefaultNasServiceStatusReader' -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/service/nas.go backend/service/nas_service_config_usecase.go backend/service/nas_service_config_store.go backend/service/nas_service_config_store_test.go backend/service/nas_service_config_usecase_test.go
git commit -m "refactor: read nas storage status from unishare"
```

---

### Task 3: Remove Legacy NAS Samba/WebDAV Quick-Create Backend

**Files:**
- Modify: `backend/modules/nas/routes.go`
- Modify: `backend/modules/nas/routes_test.go`
- Modify: `backend/service/nas_backend.go`
- Modify: `backend/service/nas.go`
- Delete: `backend/modules/nas/serviceconfig/commands.go`
- Delete: `backend/modules/nas/serviceconfig/commands_test.go`
- Delete: `backend/modules/nas/serviceconfig/service.go`
- Delete: `backend/modules/nas/serviceconfig/service_test.go`
- Delete: `backend/modules/nas/serviceconfig/types.go`
- Delete: `backend/service/nas_service_config_types.go` if no non-status aliases remain
- Delete: legacy sections of `backend/service/nas_service_config_usecase.go` and `backend/service/nas_service_config_store.go` that only support create.

- [ ] **Step 1: Remove route interface methods and route registrations**

Delete these interface methods from `backend/modules/nas/routes.go`:

```go
PostNasDiskSambaCreate(ctx context.Context, r *http.Request) (*models.NasSambaCreateResponse, error)
PostNasDiskWebdavCreate(ctx context.Context, r *http.Request) (*models.NasWebdavCreateResponse, error)
PostNasDiskWebdavStatus(ctx context.Context, r *http.Request) (*models.NasWebdavStatusResponse, error)
```

Delete route registrations for:

```go
"/cgi-bin/luci/istore/nas/samba/create"
"/cgi-bin/luci/istore/nas/webdav/create"
"/cgi-bin/luci/istore/nas/webdav/status/"
```

- [ ] **Step 2: Update route tests**

In `backend/modules/nas/routes_test.go`, remove fake backend methods for deleted routes. Add a test:

```go
func TestRegisterRoutesDoesNotExposeLegacyNasShareQuickCreate(t *testing.T) {
	router := httprouter.New()
	RegisterRoutes(router, &fakeNasBackend{})

	for _, path := range []string{
		"/cgi-bin/luci/istore/nas/samba/create",
		"/cgi-bin/luci/istore/nas/webdav/create",
		"/cgi-bin/luci/istore/nas/webdav/status/",
	} {
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{}`))
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("%s status = %d, want 404", path, rec.Code)
		}
	}
}
```

- [ ] **Step 3: Remove service backend delegators**

Delete these methods from `backend/service/nas_backend.go`:

```go
func (backend *ServiceBackend) PostNasDiskSambaCreate(...)
func (backend *ServiceBackend) PostNasDiskWebdavCreate(...)
func (backend *ServiceBackend) PostNasDiskWebdavStatus(...)
```

- [ ] **Step 4: Delete old create implementation**

Delete from `backend/service/nas.go`:

```go
func NasServiceSambaCreate(...)
func NasServiceWebdavCreate(...)
func NasServiceWebdavStatus(...)
func enableRoot()
```

Delete old create-specific facades and store methods:

```go
newNasSambaCreateServiceFacade
newNasWebdavCreateServiceFacade
newNasWebdavStatusServiceFacade
defaultNasServiceConfigWriter.PrepareSamba
defaultNasServiceConfigWriter.CreateSambaUser
defaultNasServiceConfigWriter.WriteSambaShare
defaultNasServiceConfigWriter.WriteWebdavConfig
defaultNasServiceConfigWriter.RestartWebdav
defaultNasSambaTemplateWriter
```

- [ ] **Step 5: Remove dead models only after compile check**

Run:

```bash
rtk rg -n "NasSambaCreate|NasWebdavCreate|NasWebdavStatus|NasCreateSamba|NasCreateWebdav" backend web/src
```

If backend no longer references generated model files, leave generated `backend/models/nas_*` files in place unless the model generator is also updated in this task. Generated-model cleanup should be a separate commit if needed.

- [ ] **Step 6: Run backend route and service tests**

Run:

```bash
rtk go test ./modules/nas ./service -count=1
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add backend/modules/nas backend/service
git rm backend/modules/nas/serviceconfig/commands.go backend/modules/nas/serviceconfig/commands_test.go backend/modules/nas/serviceconfig/service.go backend/modules/nas/serviceconfig/service_test.go backend/modules/nas/serviceconfig/types.go
git commit -m "refactor: remove legacy nas share quick create"
```

---

### Task 4: Frontend Request Types and Share API Helpers

**Files:**
- Modify: `web/src/request/request.ts`
- Modify: `web/src/types/response.d.ts`
- Modify: `web/src/types/nas.d.ts`

- [ ] **Step 1: Add Share request helpers**

Add to `web/src/request/request.ts`:

```ts
export const Share = {
  User: {
    List: {
      GET() {
        return Request<ShareUserListResponse>("/share/user/list/", { method: "GET" });
      },
    },
    Create: {
      POST(data: ShareUserCreateRequest) {
        return Request("/share/user/create/", {
          method: "POST",
          headers: { "Content-Type": "application/json;charset=utf-8" },
          body: JSON.stringify(data),
        });
      },
    },
    Update: {
      POST(data: ShareUserCreateRequest) {
        return Request("/share/user/update/", {
          method: "POST",
          headers: { "Content-Type": "application/json;charset=utf-8" },
          body: JSON.stringify(data),
        });
      },
    },
  },
  Service: {
    List: {
      GET() {
        return Request<ShareServiceListResponse>("/share/service/list/", { method: "GET" });
      },
    },
    Create: {
      POST(data: ShareServiceCreateRequest) {
        return Request("/share/service/create/", {
          method: "POST",
          headers: { "Content-Type": "application/json;charset=utf-8" },
          body: JSON.stringify(data),
        });
      },
    },
    Update: {
      POST(data: ShareServiceCreateRequest) {
        return Request("/share/service/update/", {
          method: "POST",
          headers: { "Content-Type": "application/json;charset=utf-8" },
          body: JSON.stringify(data),
        });
      },
    },
  },
};
```

Include `Share` in the default export if `request.ts` exports an aggregate object at the bottom.

- [ ] **Step 2: Remove old NAS quick-create helpers**

Delete:

```ts
Nas.Samba.Create.POST
Nas.Webdav.Create.POST
Nas.Webdav.Status.GET
```

Keep `Nas.Service.Status.GET`.

- [ ] **Step 3: Add frontend Share types**

Uncomment or add to both `web/src/types/response.d.ts` and `web/src/types/nas.d.ts`:

```ts
declare interface ShareUserInfo {
  userName?: string;
  password?: string;
}

declare interface ShareUserCreateRequest {
  userName: string;
  password: string;
}

declare interface ShareUserListResponse {
  users: ShareUserInfo[];
}

declare interface ShareServiceUserPermission {
  userName?: string;
  ro?: boolean;
  rw?: boolean;
}

declare interface ShareServiceInfo {
  name?: string;
  path?: string;
  samba?: boolean;
  webdav?: boolean;
  users: ShareServiceUserPermission[];
}

declare interface ShareServiceCreateRequest {
  name: string;
  path: string;
  samba: boolean;
  webdav: boolean;
  users: ShareServiceUserPermission[];
}

declare interface ShareServiceListResponse {
  services: ShareServiceInfo[];
}

declare interface NasCreateUniShare {
  shareName: string;
  username: string;
  password: string;
  rootPath: string;
  samba: boolean;
  webdav: boolean;
}
```

Remove `NasCreateSamba` and `NasCreateWebdav` after the old components stop compiling against them.

- [ ] **Step 4: Run TypeScript check and verify expected failures**

Run:

```bash
cd web && rtk npm run tsc
```

Expected: FAIL at old Samba/WebDAV action components because they still reference removed request helpers and old types.

- [ ] **Step 5: Commit**

```bash
git add web/src/request/request.ts web/src/types/response.d.ts web/src/types/nas.d.ts
git commit -m "feat: add frontend unishare request helpers"
```

---

### Task 5: Frontend UniShare Quick Setup Component

**Files:**
- Create: `web/src/components/action-nas/unishare/index.ts`
- Create: `web/src/components/action-nas/unishare/index.vue`
- Modify: `web/src/components/action-nas/index.vue`
- Delete: `web/src/components/action-nas/samba/index.ts`
- Delete: `web/src/components/action-nas/samba/index.vue`
- Delete: `web/src/components/action-nas/webdav/index.ts`
- Delete: `web/src/components/action-nas/webdav/index.vue`

- [ ] **Step 1: Create component wrapper**

Create `web/src/components/action-nas/unishare/index.ts`:

```ts
import { createApp } from "vue";
import component from "./index.vue";

declare interface NasUniShareProps {
  rootPath: string;
  defaultSamba?: boolean;
  defaultWebdav?: boolean;
}

export default (props: NasUniShareProps) => {
  const el = document.createElement("div");
  document.body.appendChild(el);
  const vm = createApp(component, {
    ...props,
    Close: () => {
      Close();
    },
  });
  vm.mount(el);
  const Close = () => {
    vm.unmount();
    el.remove();
  };
  return { Close };
};
```

- [ ] **Step 2: Create unified form**

Create `web/src/components/action-nas/unishare/index.vue` with:

```vue
<template>
  <action-component :type="1">
    <transition name="rotate" mode="out-in">
      <form class="action" @submit.prevent="onSubmit">
        <div class="action-header">
          <div class="action-header_title">{{ $gettext("统一共享配置") }}</div>
        </div>
        <div class="action-body">
          <div class="label-item">
            <div class="label-item_key"><span>{{ $gettext("服务目录路径") }}</span></div>
            <div class="label-item_value">
              <input type="text" :value="form.rootPath" disabled required style="background-color: #eee" />
            </div>
          </div>
          <div class="label-item">
            <div class="label-item_key"><span>{{ $gettext("共享名（建议使用英文字母）") }}</span></div>
            <div class="label-item_value">
              <input type="text" v-model.trim="form.shareName" required :placeholder="$gettext('共享名称')" />
            </div>
          </div>
          <div class="label-item">
            <div class="label-item_key"><span>{{ $gettext("用户名") }}</span></div>
            <div class="label-item_value">
              <input type="text" required :placeholder="$gettext('账号用户名')" v-model.trim="form.username" />
            </div>
          </div>
          <div class="label-item">
            <div class="label-item_key"><span>{{ $gettext("密码") }}</span></div>
            <div class="label-item_value">
              <input type="password" required v-model.trim="form.password" />
            </div>
          </div>
          <div class="protocols">
            <label><input type="checkbox" v-model="form.samba" /> Samba</label>
            <label><input type="checkbox" v-model="form.webdav" /> WebDAV</label>
          </div>
        </div>
        <div class="action-footer">
          <div class="auto"></div>
          <button class="cbi-button cbi-button-remove app-btn app-back" type="button" @click="onClose" :disabled="disabled">{{ $gettext("关闭") }}</button>
          <button class="cbi-button cbi-button-apply app-btn app-next" :disabled="disabled">{{ $gettext("创建") }}</button>
        </div>
      </form>
    </transition>
  </action-component>
</template>

<script setup lang="ts">
import { ref } from "vue";
import { useGettext } from "/@/plugins/i18n";
import request from "/@/request";
import ActionComponent from "/@/components/action/modal.vue";
import Toast from "/@/components/toast";
import utils from "/@/utils";

const { $gettext } = useGettext();
const props = defineProps({
  rootPath: { type: String, required: true },
  defaultSamba: { type: Boolean, default: true },
  defaultWebdav: { type: Boolean, default: true },
  Close: Function,
});

const disabled = ref(false);
const form = ref<NasCreateUniShare>({
  shareName: "",
  username: "",
  password: "",
  rootPath: props.rootPath,
  samba: props.defaultSamba,
  webdav: props.defaultWebdav,
});

const onClose = (e: Event) => {
  e.preventDefault();
  props.Close?.();
};

const onSubmit = async () => {
  const value = form.value;
  if (!value.rootPath) return Toast.Warning($gettext("共享路径不能为空"));
  if (!value.shareName) return Toast.Warning($gettext("共享名称不能为空"));
  if (!value.username) return Toast.Warning($gettext("用户名不能为空"));
  if (!value.password) return Toast.Warning($gettext("密码不能为空"));
  if (!value.samba && !value.webdav) return Toast.Warning($gettext("请至少选择一种共享协议"));

  const nameCheck = utils.checkSmabaUserName(value.username);
  if (nameCheck !== true) return Toast.Warning(`${nameCheck}`);

  disabled.value = true;
  const load = Toast.Loading($gettext("创建中..."));
  try {
    const users = await request.Share.User.List.GET();
    const exists = Boolean(users?.data?.result?.users?.some((user) => user.userName === value.username));
    const userPayload = { userName: value.username, password: value.password };
    if (exists) {
      await request.Share.User.Update.POST(userPayload);
    } else {
      await request.Share.User.Create.POST(userPayload);
    }

    const services = await request.Share.Service.List.GET();
    const serviceExists = Boolean(services?.data?.result?.services?.some((service) => service.name === value.shareName));
    const servicePayload: ShareServiceCreateRequest = {
      name: value.shareName,
      path: value.rootPath,
      samba: value.samba,
      webdav: value.webdav,
      users: [{ userName: value.username, rw: true }],
    };
    if (serviceExists) {
      await request.Share.Service.Update.POST(servicePayload);
    } else {
      await request.Share.Service.Create.POST(servicePayload);
    }

    Toast.Success($gettext("创建成功"));
    window.setTimeout(() => location.reload(), 1000);
  } catch (error) {
    Toast.Error(error as string);
  } finally {
    load.Close();
    disabled.value = false;
  }
};
</script>
```

Use the old component CSS as the starting point and keep the same modal dimensions.

- [ ] **Step 3: Wire action-nas to unified component**

Modify `web/src/components/action-nas/index.vue`:

```ts
import ActionUniShare from "./unishare";
```

Replace imports of `ActionWebdav` and `ActionSamba`. Replace the service options with:

```html
<option value="unishare">{{ $gettext("局域网文件共享（Samba + WebDAV）") }}</option>
```

Set:

```ts
const service = ref("unishare");
```

Replace `onNext` with a single UniShare flow:

```ts
const onNext = async () => {
  await checkIsInstallUniShare();
};

const checkIsInstallUniShare = async () => {
  disabled.value = true;
  const checks = [
    ["luci-app-unishare", "UniShare"],
    ["unishare", "UniShare"],
    ["samba4-server", "Samba"],
    ["webdav2", "WebDAV"],
  ] as const;
  for (const [pkg, label] of checks) {
    if (!(await appUtils.checkAndInstallApp(pkg, label))) {
      disabled.value = false;
      return;
    }
  }
  onDisk();
  disabled.value = false;
};
```

Replace `Next` handling with:

```ts
Next: (rootPath: string) => onUniShare(rootPath)
```

Add:

```ts
const onUniShare = (rootPath: string) => {
  ActionUniShare({ rootPath, defaultSamba: true, defaultWebdav: true });
  onClose();
};
```

- [ ] **Step 4: Remove old action component files**

Run:

```bash
git rm web/src/components/action-nas/samba/index.ts web/src/components/action-nas/samba/index.vue web/src/components/action-nas/webdav/index.ts web/src/components/action-nas/webdav/index.vue
```

- [ ] **Step 5: Run TypeScript check**

Run:

```bash
cd web && rtk npm run tsc
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add web/src/components/action-nas web/src/request/request.ts web/src/types
git commit -m "feat: use unishare for nas quick setup"
```

---

### Task 6: Frontend Storage Status Uses UniShare Semantics

**Files:**
- Modify: `web/src/pages/index/Storage/index.vue`
- Modify: `web/src/pages/index/Storage/samba.vue`
- Modify: `web/src/pages/index/Storage/webdav.vue`

- [ ] **Step 1: Update advanced configuration links**

In `web/src/pages/index/Storage/index.vue`, replace separate Samba/WebDAV advanced menu behavior with UniShare:

```html
<div><a href="/cgi-bin/luci/admin/nas/unishare">{{ $gettext("统一共享高级配置") }}</a></div>
```

Remove `onClickWebDAV`, `MoreDevice`, and `showWebdavBlock` if unused.

- [ ] **Step 2: Update status display copy**

Keep the three tabs if desired, but ensure Samba and WebDAV tabs are backed by the same UniShare-derived `service.sambas` and `service.webdav` status. For WebDAV, display:

```ts
const target = computed(() => `http://${location.hostname}:${props.webdav?.port || "8080"}`);
```

Do not display a WebDAV-specific username/password because credentials are now unified users.

- [ ] **Step 3: Run TypeScript check**

Run:

```bash
cd web && rtk npm run tsc
```

Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add web/src/pages/index/Storage
git commit -m "refactor: show unishare-backed storage status"
```

---

### Task 7: Full Verification and Cleanup

**Files:**
- Inspect all changed files.
- No new files unless tests reveal missing fixtures.

- [ ] **Step 1: Search for old quick-create references**

Run:

```bash
rtk rg -n "nas/samba/create|nas/webdav/create|nas/webdav/status|NasCreateSamba|NasCreateWebdav|app-meta-gowebdav|gowebdav" backend web/src
```

Expected: no live references in quick setup code. References in docs or generated OpenAPI YAML may remain only if model generation is intentionally deferred; list them in the final summary.

- [ ] **Step 2: Run backend tests**

Run:

```bash
cd backend && rtk go test ./...
```

Expected: PASS.

- [ ] **Step 3: Run frontend type check**

Run:

```bash
cd web && rtk npm run tsc
```

Expected: PASS.

- [ ] **Step 4: Run frontend build**

Run:

```bash
cd web && rtk npm run build
```

Expected: PASS and `web/dist/quickstart_web.zip` generated.

- [ ] **Step 5: Manual remote smoke test**

On a device with `luci-app-unishare`, `unishare`, `samba4-server`, and `webdav2` installed, create a share through the quickstart UI. Then run:

```bash
rtk ssh root@192.168.30.244 'uci show unishare; uci show samba4 | grep -E "unishare|path|name|write_list|read_list"; cat /var/run/unishare/webdav.yml | sed -n "1,160p"; ps w | grep -E "webdav2|smbd|nmbd" | grep -v grep'
```

Expected:
- `unishare.@user[*]` contains the submitted username and password.
- `unishare.@share[*]` contains the selected path and share name.
- `unishare.@share[*].proto` contains both `samba` and `webdav` when both protocols were checked.
- `samba4.@sambashare[*].unishare='1'` exists after reload.
- `/var/run/unishare/webdav.yml` contains the same username and protocol rules.
- `webdav2`, `smbd`, and `nmbd` are running.

- [ ] **Step 6: Commit final cleanup**

```bash
git status --short
git add -A
git commit -m "chore: verify unishare quick setup replacement"
```

---

## Self-Review

- Spec coverage: The plan replaces separate Samba/GowebDAV quick setup with UniShare user/share writes, removes old NAS quick-create routes, keeps the homepage storage card, and verifies remote UniShare behavior.
- Placeholder scan: No unfinished marker words or undefined later work remain. Generated OpenAPI cleanup is explicitly deferred unless compile/search shows live references.
- Type consistency: Frontend request payload names match existing generated model JSON fields: `userName`, `password`, `name`, `path`, `samba`, `webdav`, `users`, `rw`, and `ro`.
