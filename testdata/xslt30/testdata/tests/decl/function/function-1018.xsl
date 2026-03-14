<?xml version="1.0"?> 

<?spec xslt#stylesheet-functions?><xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:x = "http://xxx.com/"
xmlns:xs="http://www.w3.org/2001/XMLSchema">

<!-- Error: xsl:sequence within stylesheet function returning wrong type of value -->
<!-- Test is obsolete, as attribute no longer allowed -->
<?spec xslt#stylesheet-functions?>
<?error XTSE0090?>

<xsl:function name="x:test">
  <xsl:param name="n"/>
  <xsl:sequence as="xs:integer" select="$n"/>
</xsl:function>

  <xsl:template match="/">
      <out one="{x:test('banana')}"/>
  </xsl:template>


</xsl:stylesheet>
