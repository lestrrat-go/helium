<xsl:stylesheet xmlns:axml="http://www.example.com/alias-xml"
                xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                version="2.0">

<!-- Two namespaces with same prefix: non-recoverable error XTDE0430 -->
<?spec xslt#creating-namespace-nodes?>
<?error XTDE0430?>

<xsl:template match="/">
  <name>
    <xsl:namespace name="a" select="'http://alpha/'"/>
    <xsl:attribute name="x" select="2"/>
    <xsl:namespace name="a" select="'http://beta/'"/>
  </name>
</xsl:template>

</xsl:stylesheet>