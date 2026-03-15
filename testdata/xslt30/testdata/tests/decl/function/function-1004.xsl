<?xml version="1.0"?> 

<?spec xslt#stylesheet-functions?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:saxon = "http://icl.com/saxon">

<!-- test XPath comments in awkward places -->

<xsl:function name="saxon:factorial">
    <xsl:param name="n"/>
    <xsl:sequence select="if (:surprise:) ($n=1) then 1 else $n * saxon:factorial($n - 1)"/>
</xsl:function>

  <xsl:template match="/">
      <out><xsl:value-of select="saxon:factorial (:bingo:) (5)"/></out>
  </xsl:template>


</xsl:stylesheet>
