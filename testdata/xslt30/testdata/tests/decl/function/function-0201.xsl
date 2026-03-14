<?xml version="1.0"?> 

<?spec xslt#stylesheet-functions?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:saxon = "http://icl.com/saxon">

<!-- simple function, 2 args, one atomized -->

<xsl:function name="saxon:initial">
    <xsl:param name="node"/>
    <xsl:param name="size"/>
    <xsl:sequence select="substring($node,1,$size)"/>
</xsl:function>

  <xsl:template match="text()">
      <xsl:value-of select="saxon:initial(.,3)"/>
  </xsl:template>

  <xsl:template match="*">
  <xsl:copy>
      <xsl:apply-templates/>
  </xsl:copy>    
  </xsl:template>

</xsl:stylesheet>
