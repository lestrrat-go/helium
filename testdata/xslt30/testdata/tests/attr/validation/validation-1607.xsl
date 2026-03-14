<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" version="3.0"
  xmlns:v16="http://ns.example.com/val16/">
  
  <!-- test to establish whether the implicit result document is validated. Bug 30211 -->

  <xsl:import-schema namespace="http://ns.example.com/val16/" schema-location="validation-16.xsd"/>
 
  <xsl:template match="/" default-validation="strip">
    <v16:doc xsl:validation="strict">
      <xsl:for-each select="1 to 10">
        <v16:test id="Z{position() mod 5}"/>
      </xsl:for-each>
    </v16:doc>
  </xsl:template>

</xsl:stylesheet>
