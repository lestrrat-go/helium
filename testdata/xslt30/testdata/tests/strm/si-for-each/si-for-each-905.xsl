<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes=" xs" version="3.0">

  <!-- streaming, xsl:for-each, multiple downward selection in loop (disallowed) -->

  <xsl:strip-space elements="*"/>


  <xsl:template name="main">
    <out>
      <xsl:source-document streamable="yes" href="../docs/bullets.xml">
        <xsl:for-each select="child::node()">
           <in><xsl:value-of select="count(*) + count(*/*)"/></in>
        </xsl:for-each>  
      </xsl:source-document>
    </out>
  </xsl:template>

</xsl:transform>
