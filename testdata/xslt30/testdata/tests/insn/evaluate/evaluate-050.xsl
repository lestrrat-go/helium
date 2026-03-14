<?xml version="1.0"?> 
<xsl:stylesheet 
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  version="3.0">

  <!-- PURPOSE trigger Saxon bug 5626 - incorrect inference of context item type -->


  <xsl:param name="p">.</xsl:param>
  
  <xsl:template name="xsl:initial-template">
    <out>
      <xsl:for-each select="1">
        <xsl:evaluate xpath="$p || ' instance of xs:integer'" context-item="'a string'"/>
      </xsl:for-each>
    </out>
  </xsl:template>

 
</xsl:stylesheet>
