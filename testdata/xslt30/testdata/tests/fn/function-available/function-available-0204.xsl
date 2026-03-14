<?xml version="1.0"?> 

<?spec xslt#stylesheet-functions?>
<?spec xslt#testing-function-availability?>

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:f = "http://f.uri/" exclude-result-prefixes="f">

<!-- test function-available() on stylesheet function -->

<xsl:function name="f:total">
    <xsl:param name="n1"/>
    <xsl:param name="n2"/>
    <xsl:sequence select="$n1+$n2"/>
</xsl:function>

  <xsl:template match="/">
      <out>
        <true><xsl:value-of select="function-available('f:total')"/></true>
        <false><xsl:value-of select="function-available('f:jabberwocky')"/></false>
      </out>
  </xsl:template>


</xsl:stylesheet>
