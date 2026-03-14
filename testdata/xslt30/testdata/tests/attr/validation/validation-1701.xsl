<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
  xmlns:v16="http://ns.example.com/val16/"
  version="3.0" 
  exclude-result-prefixes="#all">
  
  <xsl:param static="yes" name="VALIDATION" as="xs:string" required="yes"/>
  
  <xsl:import-schema schema-location="validation-16.xsd" namespace="http://ns.example.com/val16/"/>
  
  <!-- Unknown xsi:type is an error for validation=strict/lax, but not for validation=strip/preserve -->
  
  <xsl:variable name="var" as="element()">
    <v16:doc xsi:type="xs:elastic"><v16:test id="ABC"/></v16:doc>
  </xsl:variable>

  <xsl:template match="/">
    <xsl:copy-of select="$var" _validation="{$VALIDATION}"/>
  </xsl:template>

</xsl:stylesheet>
