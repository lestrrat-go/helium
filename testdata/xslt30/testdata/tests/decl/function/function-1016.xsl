<?xml version="1.0"?> 

<?spec xslt#stylesheet-functions?><xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:x = "http://xxx.com/"
xmlns:xs="http://www.w3.org/2001/XMLSchema">

<!-- Error: two stylesheet functions with the same name and the same arity -->
<?spec xslt#stylesheet-functions?>
<?error XTSE0770?>

<xsl:function name="x:test" as="xs:integer">
  <xsl:param name="n" as="xs:integer"/>
  <xsl:sequence select="1"/>
</xsl:function>

<xsl:function name="x:test" as="xs:integer">
  <xsl:param name="n" as="xs:integer"/>
  <xsl:sequence select="2"/>
</xsl:function>

  <xsl:template match="/">
      <out one="{x:test(1)}" two="{x:test(2)}"/>
  </xsl:template>


</xsl:stylesheet>
