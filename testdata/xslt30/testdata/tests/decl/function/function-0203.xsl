<?xml version="1.0"?> 

<?spec xslt#stylesheet-functions?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:sss = "http://sss/saxon" exclude-result-prefixes="sss">

<!-- xsl:function with many arguments -->

<xsl:function name="sss:total">
    <xsl:param name="n1"/>
    <xsl:param name="n2"/>
    <xsl:param name="n3"/>
    <xsl:param name="n4"/>
    <xsl:param name="n5"/>
    <xsl:param name="n6"/>
    <xsl:param name="n7"/>
    <xsl:sequence select="$n1+$n2+$n3+$n4+$n5+$n6+$n7"/>
</xsl:function>

  <xsl:template match="/">
      <out><xsl:value-of select="sss:total(1,2,3,4,5,0,0)"/></out>
  </xsl:template>


</xsl:stylesheet>
