<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet exclude-result-prefixes="xs" version="2.0" xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  
  <xsl:template name="xsl:initial-template">
    <out>
      <xsl:for-each-group select="1 to 100" group-by=". mod 10">
        <xsl:variable name="g" select="current-group#0"/>
        <xsl:for-each-group select="500 to 600" group-by=". mod 20">
           <xsl:value-of select="$g()"/>
        </xsl:for-each-group>
      </xsl:for-each-group>
    </out>
   </xsl:template>
  
</xsl:stylesheet>