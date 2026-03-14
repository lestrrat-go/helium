<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes=" xs" version="3.0">

  <!-- streaming, xsl:iterate, downward selection in loop (disallowed) -->

  <xsl:strip-space elements="*"/>


  <xsl:template name="main">
    <out>
      <xsl:source-document streamable="yes" href="../docs/bullets.xml">
        <xsl:variable name="current" select="."/>
        <xsl:iterate select="1 to 5">
           <in><xsl:value-of select=". + count($current/*[current()])"/></in>
        </xsl:iterate>  
      </xsl:source-document>
    </out>
  </xsl:template>

</xsl:transform>
