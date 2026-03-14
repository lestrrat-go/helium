<?xml version="1.0"?> 

<?spec xslt#stylesheet-functions?><xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:x = "http://xxx.com/"
xmlns:xs="http://www.w3.org/2001/XMLSchema">

<!-- Error: wrong type of value passed as argument to a function -->
<?spec xslt#stylesheet-functions?>
<?error XTTE0790?>

<xsl:function name="x:test">
  <xsl:param name="n" as="xs:integer"/>
  <xsl:sequence select="$n"/>
</xsl:function>

  <xsl:template match="/">
      <out one="{x:test('banana')}"/>
  </xsl:template>


</xsl:stylesheet>
