<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:saxon="http://saxon.sf.net/"
  exclude-result-prefixes="#all" version="3.0">

  <xsl:mode streamable="yes" on-no-match="shallow-skip"/>
  
  <xsl:template match="/">
    <items>
      <xsl:iterate select="root/items/item">
        <xsl:copy-of select=".[@cat != 'foo']"/>
        <xsl:if test="@cat = 'bar'">
          <xsl:break/>
        </xsl:if>
      </xsl:iterate>
    </items>
  </xsl:template>
  
</xsl:stylesheet>