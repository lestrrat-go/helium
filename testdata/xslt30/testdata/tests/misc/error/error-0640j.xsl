<?xml version="1.0" encoding="UTF-8"?>
<!--In general, a circularity in a stylesheet is a non-recoverable dynamic error.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

<!-- Test circular reference through xsl:evaluate in a variable XPDE0640 -->

  <xsl:variable name="p">
    <xsl:evaluate xpath="'$y'" >
      <xsl:with-param name="y" select="$y"/>
    </xsl:evaluate>
  </xsl:variable>
  
  <xsl:variable name="y">
    <xsl:evaluate xpath="'$p'" >
      <xsl:with-param name="p" select="$p"/>
    </xsl:evaluate>
  </xsl:variable>

  <xsl:template name="xsl:initial-template">
      <out wrong="{$p}"/>
  </xsl:template>
   
</xsl:stylesheet>
