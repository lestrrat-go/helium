<?xml version="1.0"?> 

<?spec xslt#stylesheet-functions?><xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:x = "http://xxx.com/"
xmlns:xs="http://www.w3.org/2001/XMLSchema">

<!-- Error: wrong type declaration in a variable within a function -->
<?spec xslt#stylesheet-functions?>
<?error XTTE0570?>

<xsl:function name="x:test">
  <xsl:param name="n" as="xs:string"/>
  <xsl:variable name="m" as="xs:string" select="string-length($n)"/>
  <xsl:sequence select="$m"/>
</xsl:function>

  <xsl:template match="/">
      <out one="{x:test('banana')}"/>
  </xsl:template>


</xsl:stylesheet>
