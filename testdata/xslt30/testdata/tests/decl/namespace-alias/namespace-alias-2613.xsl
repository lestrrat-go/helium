<xsl:stylesheet xmlns:axml="http://www.example.com/alias-xml"
                xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                version="2.0">

<!-- Example of xsl:namespace-alias from the 2.0 spec -->
<?spec xslt#namespace-aliasing?>

<xsl:namespace-alias stylesheet-prefix="axml" result-prefix="xml"/>

<xsl:template match="/">
  <name axml:space="preserve">
    <first>James</first>
    <xsl:text> </xsl:text>
    <last>Clark</last>
  </name>
</xsl:template>

</xsl:stylesheet>