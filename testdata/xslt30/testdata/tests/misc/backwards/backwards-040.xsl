<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="1.0">

  <!-- PURPOSE: effect of BC on parameter passing in built-in templates -->
  
  <!-- the correct result for this test depends on the outcome of 
       http://www.w3.org/Bugs/Public/show_bug.cgi?id=3073 -->

  <xsl:template match="/">
    <out>
      <xsl:apply-templates>
        <xsl:with-param name="p" select="3"/>
      </xsl:apply-templates>
    </out>
  </xsl:template>
  
  <xsl:template match="item" version="2.0">
    <xsl:param name="p"/>
    <item-found param="{$p}">
      <xsl:copy-of select="."/>
    </item-found>
  </xsl:template>

</xsl:stylesheet>
