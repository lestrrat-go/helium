<?xml version="1.0"?> 

<?spec xslt#stylesheet-functions?><xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:x = "http://xxx.com/"
xmlns:xs="http://www.w3.org/2001/XMLSchema">

<!-- Error: reserved namespace used for a function -->
<?spec xslt#stylesheet-functions?>
<?error XTSE0080?>

<xsl:function name="xs:test">
  <xsl:param name="n" as="xs:string"/>
  <xsl:variable name="m" as="xs:integer" select="string-length($n)"/>
  <xsl:sequence select="$m"/>
</xsl:function>

  <xsl:template match="/">
      <out/>
  </xsl:template>


</xsl:stylesheet>
