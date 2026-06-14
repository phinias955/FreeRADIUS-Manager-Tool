<template>
  <div class="min-h-screen bg-gradient-to-br from-slate-950 via-blue-950 to-slate-900 flex flex-col items-center justify-center p-4">

    <!-- Header -->
    <div class="mb-8 text-center">
      <div class="flex items-center justify-center gap-3 mb-2">
        <div class="w-10 h-10 rounded-xl bg-blue-500 flex items-center justify-center shadow-lg shadow-blue-500/30">
          <svg class="w-6 h-6 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
              d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
          </svg>
        </div>
        <span class="text-2xl font-bold text-white tracking-tight">FreeRADIUS Manager</span>
      </div>
      <p class="text-slate-400 text-sm">First-time setup wizard</p>
    </div>

    <!-- Step progress -->
    <div class="w-full max-w-2xl mb-6">
      <div class="flex items-center justify-between">
        <template v-for="(s, i) in steps" :key="i">
          <div class="flex flex-col items-center">
            <div
              class="w-9 h-9 rounded-full flex items-center justify-center text-sm font-semibold transition-all duration-300"
              :class="stepCircleClass(i)"
            >
              <svg v-if="i < currentStep" class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M5 13l4 4L19 7" />
              </svg>
              <span v-else>{{ i + 1 }}</span>
            </div>
            <span class="mt-1 text-xs hidden sm:block" :class="i <= currentStep ? 'text-blue-400' : 'text-slate-600'">
              {{ s.label }}
            </span>
          </div>
          <div v-if="i < steps.length - 1"
            class="flex-1 h-0.5 mx-2 mb-4 rounded transition-all duration-500"
            :class="i < currentStep ? 'bg-blue-500' : 'bg-slate-700'"
          />
        </template>
      </div>
    </div>

    <!-- Card -->
    <div class="w-full max-w-2xl">
      <div class="bg-slate-900/80 backdrop-blur border border-slate-700/60 rounded-2xl shadow-2xl overflow-hidden">

        <!-- Step content -->
        <Transition :name="transitionName" mode="out-in">

          <!-- Step 0: Welcome -->
          <div v-if="currentStep === 0" key="0" class="p-8 sm:p-10">
            <div class="text-center">
              <div class="w-20 h-20 rounded-2xl bg-gradient-to-br from-blue-500 to-indigo-600 flex items-center justify-center mx-auto mb-6 shadow-lg shadow-blue-500/20">
                <svg class="w-10 h-10 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5"
                    d="M5 3v4M3 5h4M6 17v4m-2-2h4m5-16l2.286 6.857L21 12l-5.714 2.143L13 21l-2.286-6.857L5 12l5.714-2.143L13 3z" />
                </svg>
              </div>
              <h2 class="text-3xl font-bold text-white mb-3">Welcome!</h2>
              <p class="text-slate-400 text-base leading-relaxed mb-2">
                This wizard will guide you through the initial configuration of your
                <span class="text-blue-400 font-medium">FreeRADIUS Manager</span> instance.
              </p>
              <p class="text-slate-500 text-sm mb-8">
                You will set up your organisation details, create the primary administrator account,
                and configure your RADIUS and security policies. It only takes a few minutes.
              </p>
              <div class="grid grid-cols-2 sm:grid-cols-4 gap-3 text-center mb-8">
                <div v-for="item in ['Organisation', 'Admin Account', 'RADIUS Config', 'Security']" :key="item"
                  class="bg-slate-800/60 rounded-xl p-3 border border-slate-700/50">
                  <p class="text-slate-300 text-xs font-medium">{{ item }}</p>
                </div>
              </div>
            </div>
          </div>

          <!-- Step 1: Organisation -->
          <div v-else-if="currentStep === 1" key="1" class="p-8 sm:p-10">
            <h2 class="text-2xl font-bold text-white mb-1">Organisation Details</h2>
            <p class="text-slate-400 text-sm mb-7">Tell us about your organisation.</p>
            <div class="space-y-5">
              <div>
                <label class="block text-sm font-medium text-slate-300 mb-1.5">Organisation Name <span class="text-red-400">*</span></label>
                <input v-model="form.org.name" type="text" placeholder="e.g. Acme Corp" @blur="touch('org.name')"
                  class="w-full bg-slate-800 border rounded-xl px-4 py-3 text-white placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-blue-500 transition"
                  :class="errors['org.name'] ? 'border-red-500' : 'border-slate-700'" />
                <p v-if="errors['org.name']" class="mt-1 text-xs text-red-400">{{ errors['org.name'] }}</p>
              </div>
              <div>
                <label class="block text-sm font-medium text-slate-300 mb-1.5">Short Brand / Logo Text</label>
                <input v-model="form.org.logo_text" type="text" placeholder="e.g. ACME" maxlength="10"
                  class="w-full bg-slate-800 border border-slate-700 rounded-xl px-4 py-3 text-white placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-blue-500 transition" />
                <p class="mt-1 text-xs text-slate-500">Up to 10 characters — shown in the sidebar icon.</p>
              </div>
              <div>
                <label class="block text-sm font-medium text-slate-300 mb-1.5">Timezone</label>
                <select v-model="form.org.timezone"
                  class="w-full bg-slate-800 border border-slate-700 rounded-xl px-4 py-3 text-white focus:outline-none focus:ring-2 focus:ring-blue-500 transition">
                  <option v-for="tz in timezones" :key="tz" :value="tz">{{ tz }}</option>
                </select>
              </div>
            </div>
          </div>

          <!-- Step 2: Admin Account -->
          <div v-else-if="currentStep === 2" key="2" class="p-8 sm:p-10">
            <h2 class="text-2xl font-bold text-white mb-1">Administrator Account</h2>
            <p class="text-slate-400 text-sm mb-7">Create your primary super-admin account.</p>
            <div class="space-y-5">
              <div class="grid grid-cols-1 sm:grid-cols-2 gap-5">
                <div>
                  <label class="block text-sm font-medium text-slate-300 mb-1.5">Username <span class="text-red-400">*</span></label>
                  <input v-model="form.admin.username" type="text" placeholder="e.g. admin" @blur="touch('admin.username')"
                    class="w-full bg-slate-800 border rounded-xl px-4 py-3 text-white placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-blue-500 transition"
                    :class="errors['admin.username'] ? 'border-red-500' : 'border-slate-700'" />
                  <p v-if="errors['admin.username']" class="mt-1 text-xs text-red-400">{{ errors['admin.username'] }}</p>
                </div>
                <div>
                  <label class="block text-sm font-medium text-slate-300 mb-1.5">Full Name <span class="text-red-400">*</span></label>
                  <input v-model="form.admin.full_name" type="text" placeholder="e.g. John Doe" @blur="touch('admin.full_name')"
                    class="w-full bg-slate-800 border rounded-xl px-4 py-3 text-white placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-blue-500 transition"
                    :class="errors['admin.full_name'] ? 'border-red-500' : 'border-slate-700'" />
                  <p v-if="errors['admin.full_name']" class="mt-1 text-xs text-red-400">{{ errors['admin.full_name'] }}</p>
                </div>
              </div>
              <div>
                <label class="block text-sm font-medium text-slate-300 mb-1.5">Email Address <span class="text-red-400">*</span></label>
                <input v-model="form.admin.email" type="email" placeholder="admin@example.com" @blur="touch('admin.email')"
                  class="w-full bg-slate-800 border rounded-xl px-4 py-3 text-white placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-blue-500 transition"
                  :class="errors['admin.email'] ? 'border-red-500' : 'border-slate-700'" />
                <p v-if="errors['admin.email']" class="mt-1 text-xs text-red-400">{{ errors['admin.email'] }}</p>
              </div>
              <div class="grid grid-cols-1 sm:grid-cols-2 gap-5">
                <div>
                  <label class="block text-sm font-medium text-slate-300 mb-1.5">Password <span class="text-red-400">*</span></label>
                  <div class="relative">
                    <input v-model="form.admin.password" :type="showPass ? 'text' : 'password'"
                      placeholder="Min. 8 characters" @blur="touch('admin.password')"
                      class="w-full bg-slate-800 border rounded-xl px-4 py-3 pr-11 text-white placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-blue-500 transition"
                      :class="errors['admin.password'] ? 'border-red-500' : 'border-slate-700'" />
                    <button type="button" @click="showPass = !showPass"
                      class="absolute right-3 top-1/2 -translate-y-1/2 text-slate-400 hover:text-slate-200">
                      <svg v-if="showPass" class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l3.59 3.59m0 0A9.953 9.953 0 0112 5c4.478 0 8.268 2.943 9.543 7a10.025 10.025 0 01-4.132 5.411m0 0L21 21" />
                      </svg>
                      <svg v-else class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
                      </svg>
                    </button>
                  </div>
                  <!-- Password strength bar -->
                  <div class="mt-2 flex gap-1">
                    <div v-for="n in 4" :key="n" class="h-1 flex-1 rounded-full transition-all duration-300"
                      :class="passwordStrength >= n ? strengthColor : 'bg-slate-700'" />
                  </div>
                  <p class="mt-1 text-xs" :class="strengthTextColor">{{ strengthLabel }}</p>
                  <p v-if="errors['admin.password']" class="mt-1 text-xs text-red-400">{{ errors['admin.password'] }}</p>
                </div>
                <div>
                  <label class="block text-sm font-medium text-slate-300 mb-1.5">Confirm Password <span class="text-red-400">*</span></label>
                  <input v-model="form.admin.confirm" :type="showPass ? 'text' : 'password'"
                    placeholder="Re-enter password" @blur="touch('admin.confirm')"
                    class="w-full bg-slate-800 border rounded-xl px-4 py-3 text-white placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-blue-500 transition"
                    :class="errors['admin.confirm'] ? 'border-red-500' : 'border-slate-700'" />
                  <p v-if="errors['admin.confirm']" class="mt-1 text-xs text-red-400">{{ errors['admin.confirm'] }}</p>
                </div>
              </div>
            </div>
          </div>

          <!-- Step 3: RADIUS Config -->
          <div v-else-if="currentStep === 3" key="3" class="p-8 sm:p-10">
            <h2 class="text-2xl font-bold text-white mb-1">RADIUS Configuration</h2>
            <p class="text-slate-400 text-sm mb-7">Configure default RADIUS server settings.</p>
            <div class="bg-blue-500/10 border border-blue-500/30 rounded-xl p-4 mb-6 flex gap-3">
              <svg class="w-5 h-5 text-blue-400 shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
              <p class="text-blue-300 text-sm">The RADIUS shared secret must also be set in your <code class="text-blue-200 bg-slate-800 px-1 rounded">RADIUS_SECRET</code> environment variable and in your NAS device configurations to take effect.</p>
            </div>
            <div class="space-y-5">
              <div>
                <label class="block text-sm font-medium text-slate-300 mb-1.5">Default RADIUS Shared Secret</label>
                <div class="relative">
                  <input v-model="form.radius.default_secret" :type="showSecret ? 'text' : 'password'"
                    placeholder="Enter RADIUS shared secret"
                    class="w-full bg-slate-800 border border-slate-700 rounded-xl px-4 py-3 pr-11 text-white placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-blue-500 transition" />
                  <button type="button" @click="showSecret = !showSecret"
                    class="absolute right-3 top-1/2 -translate-y-1/2 text-slate-400 hover:text-slate-200">
                    <svg v-if="showSecret" class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l3.59 3.59m0 0A9.953 9.953 0 0112 5c4.478 0 8.268 2.943 9.543 7a10.025 10.025 0 01-4.132 5.411m0 0L21 21" />
                    </svg>
                    <svg v-else class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
                    </svg>
                  </button>
                </div>
              </div>
              <div>
                <label class="block text-sm font-medium text-slate-300 mb-1.5">
                  Maximum Devices per User
                  <span class="text-slate-500 font-normal ml-1">(default: 20)</span>
                </label>
                <input v-model.number="form.radius.max_devices" type="number" min="1" max="500"
                  class="w-full bg-slate-800 border border-slate-700 rounded-xl px-4 py-3 text-white focus:outline-none focus:ring-2 focus:ring-blue-500 transition" />
              </div>
              <div>
                <label class="block text-sm font-medium text-slate-300 mb-1.5">
                  Session Timeout
                  <span class="text-slate-500 font-normal ml-1">(seconds, default: 3600 = 1 hour)</span>
                </label>
                <input v-model.number="form.security.session_timeout" type="number" min="300" max="86400"
                  class="w-full bg-slate-800 border border-slate-700 rounded-xl px-4 py-3 text-white focus:outline-none focus:ring-2 focus:ring-blue-500 transition" />
              </div>
            </div>
          </div>

          <!-- Step 4: Security -->
          <div v-else-if="currentStep === 4" key="4" class="p-8 sm:p-10">
            <h2 class="text-2xl font-bold text-white mb-1">Security Policies</h2>
            <p class="text-slate-400 text-sm mb-7">Set default security rules for admin accounts.</p>
            <div class="space-y-5">
              <div class="grid grid-cols-1 sm:grid-cols-2 gap-5">
                <div>
                  <label class="block text-sm font-medium text-slate-300 mb-1.5">Minimum Password Length</label>
                  <input v-model.number="form.security.password_min_length" type="number" min="8" max="64"
                    class="w-full bg-slate-800 border border-slate-700 rounded-xl px-4 py-3 text-white focus:outline-none focus:ring-2 focus:ring-blue-500 transition" />
                </div>
                <div>
                  <label class="block text-sm font-medium text-slate-300 mb-1.5">Password Expiry (days)</label>
                  <input v-model.number="form.security.password_expiry_days" type="number" min="0" max="365"
                    class="w-full bg-slate-800 border border-slate-700 rounded-xl px-4 py-3 text-white focus:outline-none focus:ring-2 focus:ring-blue-500 transition" />
                  <p class="mt-1 text-xs text-slate-500">Set to 0 to disable expiry.</p>
                </div>
              </div>
              <div>
                <label class="block text-sm font-medium text-slate-300 mb-1.5">Max Failed Login Attempts Before Lockout</label>
                <input v-model.number="form.security.brute_force_attempts" type="number" min="3" max="20"
                  class="w-full bg-slate-800 border border-slate-700 rounded-xl px-4 py-3 text-white focus:outline-none focus:ring-2 focus:ring-blue-500 transition" />
              </div>
              <div class="flex items-center justify-between bg-slate-800/60 border border-slate-700/50 rounded-xl px-5 py-4">
                <div>
                  <p class="text-slate-200 font-medium text-sm">Require MFA for All Admin Users</p>
                  <p class="text-slate-500 text-xs mt-0.5">Forces two-factor authentication on every admin login.</p>
                </div>
                <button type="button" @click="form.security.mfa_required = !form.security.mfa_required"
                  class="relative w-12 h-6 rounded-full transition-all duration-300 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 focus:ring-offset-slate-800"
                  :class="form.security.mfa_required ? 'bg-blue-500' : 'bg-slate-600'">
                  <span class="absolute top-0.5 left-0.5 w-5 h-5 bg-white rounded-full shadow transition-transform duration-300"
                    :class="form.security.mfa_required ? 'translate-x-6' : 'translate-x-0'" />
                </button>
              </div>
            </div>
          </div>

          <!-- Step 5: Review & Complete -->
          <div v-else-if="currentStep === 5" key="5" class="p-8 sm:p-10">
            <div v-if="!completed">
              <h2 class="text-2xl font-bold text-white mb-1">Review & Launch</h2>
              <p class="text-slate-400 text-sm mb-7">Confirm your settings, then click Launch.</p>

              <div class="space-y-4 mb-8">
                <div class="bg-slate-800/60 border border-slate-700/50 rounded-xl p-4">
                  <p class="text-xs text-slate-500 uppercase tracking-wider mb-3">Organisation</p>
                  <div class="grid grid-cols-2 gap-2 text-sm">
                    <span class="text-slate-400">Name</span><span class="text-white font-medium">{{ form.org.name }}</span>
                    <span class="text-slate-400">Timezone</span><span class="text-white">{{ form.org.timezone }}</span>
                  </div>
                </div>
                <div class="bg-slate-800/60 border border-slate-700/50 rounded-xl p-4">
                  <p class="text-xs text-slate-500 uppercase tracking-wider mb-3">Admin Account</p>
                  <div class="grid grid-cols-2 gap-2 text-sm">
                    <span class="text-slate-400">Username</span><span class="text-white font-medium">{{ form.admin.username }}</span>
                    <span class="text-slate-400">Email</span><span class="text-white">{{ form.admin.email }}</span>
                    <span class="text-slate-400">Full Name</span><span class="text-white">{{ form.admin.full_name }}</span>
                    <span class="text-slate-400">Password</span><span class="text-green-400">●●●●●●●●</span>
                  </div>
                </div>
                <div class="bg-slate-800/60 border border-slate-700/50 rounded-xl p-4">
                  <p class="text-xs text-slate-500 uppercase tracking-wider mb-3">Security Policies</p>
                  <div class="grid grid-cols-2 gap-2 text-sm">
                    <span class="text-slate-400">Min Password Length</span><span class="text-white">{{ form.security.password_min_length }} chars</span>
                    <span class="text-slate-400">Password Expiry</span><span class="text-white">{{ form.security.password_expiry_days === 0 ? 'Never' : form.security.password_expiry_days + ' days' }}</span>
                    <span class="text-slate-400">Session Timeout</span><span class="text-white">{{ form.security.session_timeout }}s</span>
                    <span class="text-slate-400">MFA Required</span>
                    <span :class="form.security.mfa_required ? 'text-green-400' : 'text-slate-400'">
                      {{ form.security.mfa_required ? 'Yes' : 'No' }}
                    </span>
                    <span class="text-slate-400">Max Devices/User</span><span class="text-white">{{ form.radius.max_devices }}</span>
                  </div>
                </div>
              </div>

              <p v-if="submitError" class="text-red-400 text-sm mb-4 bg-red-500/10 border border-red-500/30 rounded-xl px-4 py-3">
                {{ submitError }}
              </p>
            </div>

            <!-- Success screen -->
            <div v-else class="text-center py-4">
              <div class="w-20 h-20 rounded-full bg-green-500/20 border-2 border-green-500 flex items-center justify-center mx-auto mb-6">
                <svg class="w-10 h-10 text-green-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
                </svg>
              </div>
              <h2 class="text-2xl font-bold text-white mb-3">All Done!</h2>
              <p class="text-slate-400 mb-2">Your FreeRADIUS Manager is ready.</p>
              <p class="text-slate-500 text-sm mb-8">
                Redirecting to the login page in {{ redirectCountdown }} second{{ redirectCountdown !== 1 ? 's' : '' }}…
              </p>
              <div class="bg-slate-800/60 border border-slate-700/50 rounded-xl p-5 text-left space-y-2">
                <p class="text-xs text-slate-500 uppercase tracking-wider mb-3">Your Credentials</p>
                <div class="flex justify-between text-sm">
                  <span class="text-slate-400">Username</span>
                  <span class="text-white font-mono font-medium">{{ form.admin.username }}</span>
                </div>
                <div class="flex justify-between text-sm">
                  <span class="text-slate-400">URL</span>
                  <span class="text-blue-400 font-mono">http://localhost:8081</span>
                </div>
              </div>
            </div>
          </div>

        </Transition>

        <!-- Navigation buttons -->
        <div v-if="!completed" class="px-8 sm:px-10 pb-8 flex items-center justify-between gap-4 border-t border-slate-800 pt-6">
          <button
            v-if="currentStep > 0"
            type="button"
            @click="prevStep"
            class="px-5 py-2.5 rounded-xl text-sm font-medium text-slate-300 hover:text-white bg-slate-800 hover:bg-slate-700 border border-slate-700 transition"
          >
            ← Back
          </button>
          <div v-else />

          <button
            v-if="currentStep < steps.length - 1"
            type="button"
            @click="nextStep"
            class="px-6 py-2.5 rounded-xl text-sm font-semibold bg-blue-600 hover:bg-blue-500 text-white transition shadow-lg shadow-blue-500/20"
          >
            {{ currentStep === 0 ? 'Get Started →' : 'Next →' }}
          </button>

          <button
            v-else
            type="button"
            @click="submitSetup"
            :disabled="submitting"
            class="px-7 py-2.5 rounded-xl text-sm font-semibold bg-green-600 hover:bg-green-500 disabled:opacity-60 disabled:cursor-not-allowed text-white transition shadow-lg shadow-green-500/20 flex items-center gap-2"
          >
            <svg v-if="submitting" class="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
            </svg>
            {{ submitting ? 'Setting up…' : '🚀 Launch' }}
          </button>
        </div>

      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { setupAPI } from '@/api/index'

const router = useRouter()

const steps = [
  { label: 'Welcome' },
  { label: 'Organisation' },
  { label: 'Admin' },
  { label: 'RADIUS' },
  { label: 'Security' },
  { label: 'Review' },
]

const currentStep = ref(0)
const transitionName = ref('slide-left')
const showPass = ref(false)
const showSecret = ref(false)
const submitting = ref(false)
const submitError = ref('')
const completed = ref(false)
const redirectCountdown = ref(5)
let countdownTimer = null

const form = reactive({
  org: {
    name: '',
    logo_text: '',
    timezone: Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC',
  },
  admin: {
    username: '',
    email: '',
    full_name: '',
    password: '',
    confirm: '',
  },
  radius: {
    default_secret: '',
    max_devices: 20,
  },
  security: {
    password_min_length: 12,
    password_expiry_days: 90,
    session_timeout: 3600,
    mfa_required: false,
    brute_force_attempts: 5,
  },
})

const errors = reactive({})

const touched = reactive({})
function touch(field) {
  touched[field] = true
  validateField(field)
}

function validateField(field) {
  delete errors[field]
  if (field === 'org.name' && !form.org.name.trim()) {
    errors['org.name'] = 'Organisation name is required'
  }
  if (field === 'admin.username') {
    if (!form.admin.username.trim()) errors['admin.username'] = 'Username is required'
    else if (form.admin.username.length < 3) errors['admin.username'] = 'Minimum 3 characters'
  }
  if (field === 'admin.full_name' && !form.admin.full_name.trim()) {
    errors['admin.full_name'] = 'Full name is required'
  }
  if (field === 'admin.email') {
    if (!form.admin.email.trim()) errors['admin.email'] = 'Email is required'
    else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(form.admin.email)) errors['admin.email'] = 'Invalid email address'
  }
  if (field === 'admin.password') {
    if (!form.admin.password) errors['admin.password'] = 'Password is required'
    else if (form.admin.password.length < 8) errors['admin.password'] = 'Minimum 8 characters'
  }
  if (field === 'admin.confirm') {
    if (form.admin.confirm !== form.admin.password) errors['admin.confirm'] = 'Passwords do not match'
  }
}

function validateStep(step) {
  const fields = {
    1: ['org.name'],
    2: ['admin.username', 'admin.full_name', 'admin.email', 'admin.password', 'admin.confirm'],
  }
  const toCheck = fields[step] || []
  toCheck.forEach(f => { touched[f] = true; validateField(f) })
  return toCheck.every(f => !errors[f])
}

function nextStep() {
  if (!validateStep(currentStep.value)) return
  transitionName.value = 'slide-left'
  currentStep.value++
}
function prevStep() {
  transitionName.value = 'slide-right'
  currentStep.value--
}

function stepCircleClass(i) {
  if (i < currentStep.value) return 'bg-blue-600 text-white'
  if (i === currentStep.value) return 'bg-blue-500 text-white ring-4 ring-blue-500/30'
  return 'bg-slate-800 text-slate-500 border border-slate-700'
}

// Password strength
const passwordStrength = computed(() => {
  const p = form.admin.password
  if (!p) return 0
  let score = 0
  if (p.length >= 8) score++
  if (p.length >= 12) score++
  if (/[A-Z]/.test(p) && /[a-z]/.test(p)) score++
  if (/[0-9]/.test(p) && /[^A-Za-z0-9]/.test(p)) score++
  return score
})
const strengthColor = computed(() => {
  const colors = ['bg-red-500', 'bg-orange-500', 'bg-yellow-500', 'bg-green-500']
  return colors[passwordStrength.value - 1] || 'bg-slate-700'
})
const strengthTextColor = computed(() => {
  const colors = ['text-red-400', 'text-orange-400', 'text-yellow-400', 'text-green-400']
  return colors[passwordStrength.value - 1] || 'text-slate-600'
})
const strengthLabel = computed(() => {
  const labels = ['Weak', 'Fair', 'Good', 'Strong']
  return passwordStrength.value > 0 ? labels[passwordStrength.value - 1] : ''
})

async function submitSetup() {
  submitError.value = ''
  submitting.value = true
  try {
    await setupAPI.complete({
      organization: {
        name: form.org.name,
        timezone: form.org.timezone,
        logo_text: form.org.logo_text,
      },
      admin: {
        username: form.admin.username,
        email: form.admin.email,
        full_name: form.admin.full_name,
        password: form.admin.password,
      },
      radius: {
        default_secret: form.radius.default_secret,
        max_devices: form.radius.max_devices,
      },
      security: {
        password_min_length: form.security.password_min_length,
        password_expiry_days: form.security.password_expiry_days,
        session_timeout: form.security.session_timeout,
        mfa_required: form.security.mfa_required,
        brute_force_attempts: form.security.brute_force_attempts,
      },
    })
    completed.value = true
    countdownTimer = setInterval(() => {
      redirectCountdown.value--
      if (redirectCountdown.value <= 0) {
        clearInterval(countdownTimer)
        router.push('/login')
      }
    }, 1000)
  } catch (err) {
    submitError.value = err.response?.data?.error || 'Setup failed — please try again.'
  } finally {
    submitting.value = false
  }
}

onUnmounted(() => { if (countdownTimer) clearInterval(countdownTimer) })

const timezones = [
  'UTC', 'Africa/Nairobi', 'Africa/Lagos', 'Africa/Cairo', 'Africa/Johannesburg',
  'America/New_York', 'America/Chicago', 'America/Denver', 'America/Los_Angeles',
  'America/Toronto', 'America/Sao_Paulo',
  'Europe/London', 'Europe/Paris', 'Europe/Berlin', 'Europe/Moscow',
  'Asia/Dubai', 'Asia/Kolkata', 'Asia/Singapore', 'Asia/Tokyo', 'Asia/Shanghai',
  'Asia/Karachi', 'Asia/Dhaka', 'Asia/Bangkok',
  'Australia/Sydney', 'Pacific/Auckland',
]
</script>

<style scoped>
.slide-left-enter-active,
.slide-left-leave-active,
.slide-right-enter-active,
.slide-right-leave-active {
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}
.slide-left-enter-from  { opacity: 0; transform: translateX(40px); }
.slide-left-leave-to    { opacity: 0; transform: translateX(-40px); }
.slide-right-enter-from { opacity: 0; transform: translateX(-40px); }
.slide-right-leave-to   { opacity: 0; transform: translateX(40px); }
</style>
