<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- DOCUMENT: http://www.w3.org/TR/xpath -->
  <!-- PURPOSE: Test passing a context-dependent node-set expression as a parameter -->
  <?spec xslt#named-templates?>
  
<xsl:template match="doc">
    <out>
      <xsl:variable name="a" select="3"/>
      <xsl:variable name="paras" select="//para[@cat=current()/@cat]"/>
      <xsl:for-each select="document('')"> <!-- change the current node to confuse things -->
          <temp><xsl:value-of select="count($paras)"/></temp>
          <xsl:call-template name="t">
            <xsl:with-param name="ns" select="$paras[$a &lt; @id]"/>
          </xsl:call-template>
      </xsl:for-each>
    </out>
</xsl:template>

<xsl:template name="t">
    <xsl:param name="ns"/>
    <xsl:value-of select="count($ns)"/>
</xsl:template>
 
</xsl:stylesheet>
