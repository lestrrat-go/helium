<?xml version="1.0"?> 

<?spec xslt#stylesheet-functions?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:saxon = "http://icl.com/saxon">

<!-- simple xsl:function, namespace not reserved despite appearances -->

<xsl:function name="saxon:count-elements-and-attributes">
    <xsl:param name="doc"/>
    <xsl:variable name="elements" select="$doc//*"/>
    <xsl:variable name="attributes" select="$doc//@*"/>
    <xsl:sequence select="count($elements | $attributes)"/>
</xsl:function>

  <xsl:template match="/">
    <out>
    <el><xsl:value-of select="count(//*)"/></el>;
    <at><xsl:value-of select="count(//@*)"/></at>;
    <union><xsl:value-of select="saxon:count-elements-and-attributes(/)"/></union>
    </out>
  </xsl:template>

</xsl:stylesheet>
