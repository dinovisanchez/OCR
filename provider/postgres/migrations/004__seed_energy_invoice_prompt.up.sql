INSERT INTO document_type_prompts (document_type, prompt_complement, fields, is_active)
VALUES (
    'energy_invoice',
    E'Este es un recibo/factura de servicio público de energía eléctrica en Colombia (operadores comunes: Air-e, Afinia, EPM, Enel, Celsia, ESSA, EMCALI, CHEC, Electricaribe, entre otros).\n\nInstrucciones generales:\n- Busca la información en toda la factura: encabezados, cuerpo, tablas, pie de página y secciones de datos del cliente o del servicio.\n- Acepta variaciones ortográficas, mayúsculas, minúsculas, acentos y formatos abreviados.\n- Devuelve los valores como strings (tal cual aparecen en el documento), excepto donde se indique otro tipo.\n- Si un campo no aparece o no puedes determinarlo con certeza, usa null. Para los campos de historial (month1..month6) usa el número 0 cuando no encuentres el dato.\n- No inventes datos. No hagas cálculos propios.\n\nGuía de campos y sinónimos/variantes comunes:\n\n- account_number: Número de cuenta del cliente. También llamado: ''Cuenta contrato'', ''Número de cuenta'', ''No. Cuenta'', ''Nro. Cuenta'', ''Cuenta'', ''Número de cliente'', ''Código del cliente'', ''Contrato'', ''Cuenta Nro''.\n\n- niu: NIU (Número Único de Identificación del servicio). También: ''NIU'', ''Número Único de Identificación'', ''Identificador del servicio'', ''ID del servicio''.\n\n- meter_number: Número/serie del medidor eléctrico. También: ''Medidor'', ''Nro. Medidor'', ''No. Medidor'', ''Serie del medidor'', ''Serie medidor'', ''Contador'', ''Nro. Contador''. Puede contener letras, números y guiones (ej: ''34001262-MCA51'').\n\n- city: Ciudad o municipio del predio. También: ''Municipio'', ''Localidad'', ''Ciudad del servicio''.\n\n- address: Dirección del predio donde se presta el servicio. También: ''Dirección'', ''Domicilio'', ''Dirección del predio'', ''Dirección del inmueble'', ''Dirección de suministro''. Devuelve la dirección completa tal cual aparece.\n\n- level: Nivel de tensión del servicio (suele ser ''1'', ''2'', ''3'' o ''4''). También: ''Nivel de tensión'', ''Nivel Tensión'', ''Nivel'', ''Tensión'', ''NT''. Devolver solo el número como string.\n\n- property: Tipo de propiedad o clase de usuario (ej: ''Usuario'', ''Propietario'', ''Arrendatario'', ''Poseedor''). También: ''Propiedad'', ''Clase de propiedad'', ''Tipo de usuario'', ''Calidad''.\n\n- use: Uso del inmueble con su clasificación completa (ej: ''Resid. Estrato 5'', ''Residencial Estrato 3'', ''Comercial'', ''Industrial'', ''Oficial''). También: ''Uso'', ''Tipo de uso'', ''Clase de servicio'', ''Uso del inmueble'', ''Sector''.\n\n- estrato: Estrato socioeconómico (1 a 6). También: ''Estrato'', ''Estrato socioeconómico'', ''Est''. Devolver solo el número como string. Si el uso no es residencial puede ser null.\n\n- operator: Operador de red (empresa que opera la infraestructura). También: ''Operador de Red'', ''OR'', ''Operador'', ''Empresa Operadora'', ''Distribuidor''. Ejemplos: ''Air-e'', ''Afinia'', ''EPM'', ''Enel'', ''Celsia''.\n\n- marketer: Comercializador (empresa que factura la energía). También: ''Comercializador'', ''Empresa comercializadora'', ''Comercializadora''. Puede coincidir con el operador.\n\n- month1, month2, month3, month4, month5, month6: Historial de consumo de los últimos 6 meses en kWh. También aparece como: ''Historial de consumo'', ''Consumo últimos 6 meses'', ''Histórico de consumo'', ''Consumo mensual'', ''Consumo histórico (kWh)'', tabla o gráfica de barras con consumos mensuales.\n  - month1 = mes más reciente (el mes actual de facturación o el más reciente del histórico).\n  - month2 = mes anterior a month1, y así sucesivamente hasta month6 (el más antiguo).\n  - Devuelve solo el número (como string o número); si un mes no aparece o no se puede determinar, devuelve 0.\n  - Si los valores vienen en una gráfica/tabla, extrae los números asociados a cada barra/fila.\n\n- cu: CU (Costo Unitario de prestación del servicio), generalmente con decimales. También: ''CU'', ''Costo Unitario'', ''Costo Unitario de Prestación'', ''Tarifa CU'', ''$/kWh''. Devolver el valor numérico tal cual aparece (ej: ''1044.88'').\n\n- period: Período facturado. También: ''Período de facturación'', ''Mes de facturación'', ''Período'', ''Mes facturado'', ''Ciclo''. Formato sugerido: ''Mes Año'' (ej: ''Diciembre 2024'').\n\n- is_commercial: Booleano (true/false) que indica si el uso es comercial/industrial. Devolver true si ''use'' contiene palabras como ''Comercial'', ''Industrial'', ''Oficial'', ''No residencial''. Devolver false si es ''Residencial'' o ''Resid.'' en cualquier estrato. Si no puedes determinarlo, devolver false.',
    ARRAY[
        'account_number',
        'niu',
        'meter_number',
        'city',
        'address',
        'level',
        'property',
        'use',
        'estrato',
        'operator',
        'marketer',
        'month1',
        'month2',
        'month3',
        'month4',
        'month5',
        'month6',
        'cu',
        'period',
        'is_commercial'
    ]::TEXT[],
    true
)
ON CONFLICT (document_type) DO NOTHING;
