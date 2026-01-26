require('dotenv').config();
const pm2 = require('pm2');
const fs = require('fs');
const path = require('path');
const { v4: uuidv4 } = require('uuid');

// 1. Configurações
const LOG_DIR = process.env.LOG_OUTPUT_PATH || '../logs';

// Processa a lista de apps do .env
const TARGET_APPS = (process.env.PM2_APPS_TO_MONITOR || '')
  .split(',')
  .map(name => name.trim())
  .filter(name => name.length > 0);

// ---- VALIDAÇÃO BLOQUEANTE (FAIL FAST) ----
if (TARGET_APPS.length === 0) {
  console.error("❌ ERRO CRÍTICO: A variável 'PM2_APPS_TO_MONITOR' está vazia ou não definida no .env.");
  console.error("   O serviço será encerrado pois não há nada para monitorar.");
  process.exit(1); // Encerra o processo com código de erro
}

console.log(`🎯 Monitorando os seguintes apps: [ ${TARGET_APPS.join(', ')} ]`);
console.log(`📂 Diretório de saída: ${LOG_DIR}`);

// Garante diretório
if (!fs.existsSync(LOG_DIR)){
  try {
    fs.mkdirSync(LOG_DIR, { recursive: true });
  } catch (e) {
    console.error(`❌ Erro ao criar diretório de logs: ${e.message}`);
    process.exit(1);
  }
}

// 2. Conexão PM2
pm2.connect(function(err) {
  if (err) {
    console.error(`❌ Erro ao conectar no PM2: ${err.message}`);
    process.exit(2);
  }

  pm2.launchBus(function(err, bus) {
    console.log('✅ Conectado ao PM2 Bus e escutando...');

    bus.on('log:out', (packet) => processLog(packet, 'INFO'));
    bus.on('log:err', (packet) => processLog(packet, 'ERROR'));
  });
});

// 3. Processamento
function processLog(packet, severity) {
  // Se o nome do processo não estiver na lista permitida, ignoramos
  if (!TARGET_APPS.includes(packet.process.name)) {
    return;
  }

  const logEntry = {
    severity: severity,
    jsonPayload: {
      message: packet.data,
      process_name: packet.process.name,
      pm_id: packet.process.pm_id,
      timestamp: new Date().toISOString()
    },
    labels: {
      source: "pm2-listener",
      environment: process.env.NODE_ENV || "production"
    }
  };

  const fileName = `${Date.now()}-${uuidv4()}.json`;
  const filePath = path.join(LOG_DIR, fileName);

  fs.writeFile(filePath, JSON.stringify(logEntry, null, 2), (err) => {
    if (err) console.error(`⚠️ Erro ao salvar log de ${packet.process.name}: ${err.message}`);
  });
}
