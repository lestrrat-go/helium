<?xml version="1.0"?> 

<?spec xslt#stylesheet-functions?><xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:x = "http://xxx.com/"
xmlns:xs="http://www.w3.org/2001/XMLSchema">

<!-- Error: stylesheet function returning wrong type of value -->
<?spec xslt#stylesheet-functions?>
<?error XTTE0780?>

<xsl:function name="x:test" as="xs:integer">
  <xsl:param name="n"/>
  <xsl:sequence select="$n"/>
</xsl:function>

  <xsl:template match="/">
      <out one="{x:test('banana')}"/>
  </xsl:template>


</xsl:stylesheet>
