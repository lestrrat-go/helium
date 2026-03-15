<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  version="3.0">
  <!-- Saxon issue 6151 -->
  <!-- This test deliberately does not preload any schema in its environment, to
       ensure that it is the import-schema declaration itself that detects the
       mismatched namespace -->

  <xsl:import-schema namespace="http://unexpected/" schema-location="schema002.xsd"/>
  
  <xsl:template match="/">
    <out validation="lax"/>
  </xsl:template>
  


</xsl:stylesheet>
