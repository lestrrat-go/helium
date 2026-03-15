<xsl:transform 
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
  xmlns:axml="http://alias/">
  
<?spec xslt#namespace-aliasing?>
<xsl:namespace-alias stylesheet-prefix="axml" result-prefix="xml"/>

<xsl:template match="/">
  <doc axml:space="preserve">text</doc>
</xsl:template>

</xsl:transform>